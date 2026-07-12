package identity

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"testing"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
)

func certDER(t *testing.T, id *Identity) []byte {
	t.Helper()
	cert, err := id.TLSCertificate()
	if err != nil {
		t.Fatalf("TLSCertificate: %v", err)
	}
	return cert.Certificate[0]
}

func TestPeerIdentity(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	peerID, cert, err := PeerIdentity([][]byte{certDER(t, id)})
	if err != nil {
		t.Fatalf("PeerIdentity: %v", err)
	}
	if peerID != id.ID() {
		t.Errorf("peer ID = %s, want %s", peerID, id.ID())
	}
	if cert == nil {
		t.Error("expected a parsed certificate")
	}
}

func TestPeerIdentityErrors(t *testing.T) {
	if _, _, err := PeerIdentity(nil); err == nil {
		t.Error("PeerIdentity with no certs should error")
	}
	if _, _, err := PeerIdentity([][]byte{[]byte("garbage")}); err == nil {
		t.Error("PeerIdentity with invalid cert should error")
	}
}

func TestConfigsUseTLS13AndALPN(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	server, err := id.ServerTLSConfig(nil)
	if err != nil {
		t.Fatalf("ServerTLSConfig: %v", err)
	}
	client, err := id.ClientTLSConfig(nil)
	if err != nil {
		t.Fatalf("ClientTLSConfig: %v", err)
	}
	for name, cfg := range map[string]*tls.Config{"server": server, "client": client} {
		if cfg.MinVersion != tls.VersionTLS13 {
			t.Errorf("%s MinVersion = %x, want TLS 1.3", name, cfg.MinVersion)
		}
		if len(cfg.NextProtos) != 1 || cfg.NextProtos[0] != ALPN {
			t.Errorf("%s NextProtos = %v, want [%s]", name, cfg.NextProtos, ALPN)
		}
		if len(cfg.Certificates) != 1 {
			t.Errorf("%s should present exactly one certificate", name)
		}
	}
	if server.ClientAuth != tls.RequireAnyClientCert {
		t.Error("server config should require a client certificate")
	}
}

func TestMutualTLSHandshake(t *testing.T) {
	server, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	client, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var seenByServer, seenByClient core.DeviceID
	serverCfg, err := server.ServerTLSConfig(func(id core.DeviceID, _ *x509.Certificate) error {
		seenByServer = id
		return nil
	})
	if err != nil {
		t.Fatalf("ServerTLSConfig: %v", err)
	}
	clientCfg, err := client.ClientTLSConfig(func(id core.DeviceID, _ *x509.Certificate) error {
		seenByClient = id
		return nil
	})
	if err != nil {
		t.Fatalf("ClientTLSConfig: %v", err)
	}

	if err := handshake(serverCfg, clientCfg); err != nil {
		t.Fatalf("handshake failed: %v", err)
	}
	if seenByServer != client.ID() {
		t.Errorf("server saw peer %s, want %s", seenByServer, client.ID())
	}
	if seenByClient != server.ID() {
		t.Errorf("client saw peer %s, want %s", seenByClient, server.ID())
	}
}

func TestHandshakeRejectsUnauthorizedPeer(t *testing.T) {
	server, _ := Generate()
	client, _ := Generate()

	serverCfg, _ := server.ServerTLSConfig(func(core.DeviceID, *x509.Certificate) error {
		return errUnauthorized
	})
	clientCfg, _ := client.ClientTLSConfig(nil)

	if err := handshake(serverCfg, clientCfg); err == nil {
		t.Error("handshake should fail when the server rejects the peer")
	}
}

var errUnauthorized = &authError{}

type authError struct{}

func (*authError) Error() string { return "unauthorized peer" }

func handshake(serverCfg, clientCfg *tls.Config) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer ln.Close()

	errc := make(chan error, 2)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			errc <- err
			return
		}
		defer conn.Close()
		errc <- tls.Server(conn, serverCfg).Handshake()
	}()
	go func() {
		conn, err := tls.Dial("tcp", ln.Addr().String(), clientCfg)
		if err != nil {
			errc <- err
			return
		}
		defer conn.Close()
		errc <- nil
	}()

	var firstErr error
	for i := 0; i < 2; i++ {
		if err := <-errc; err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
