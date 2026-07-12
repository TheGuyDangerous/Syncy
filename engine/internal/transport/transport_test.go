package transport

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
	"github.com/TheGuyDangerous/Syncy/engine/internal/identity"
)

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestDialAcceptExchangesData(t *testing.T) {
	server, err := identity.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	client, err := identity.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	ln, err := Listen(server, "127.0.0.1:0", nil)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()

	ctx := testContext(t)

	type accepted struct {
		conn *Conn
		err  error
	}
	accCh := make(chan accepted, 1)
	go func() {
		c, err := ln.Accept(ctx)
		accCh <- accepted{c, err}
	}()

	clientConn, err := Dial(ctx, client, ln.Addr().String(), nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer clientConn.Close()

	a := <-accCh
	if a.err != nil {
		t.Fatalf("Accept: %v", a.err)
	}
	serverConn := a.conn
	defer serverConn.Close()

	if clientConn.Peer() != server.ID() {
		t.Errorf("client sees peer %s, want server %s", clientConn.Peer(), server.ID())
	}
	if serverConn.Peer() != client.ID() {
		t.Errorf("server sees peer %s, want client %s", serverConn.Peer(), client.ID())
	}

	msg := []byte("hello over quic")
	srvErr := make(chan error, 1)
	go func() {
		s, err := serverConn.AcceptStream(ctx)
		if err != nil {
			srvErr <- err
			return
		}
		buf := make([]byte, len(msg))
		if _, err := io.ReadFull(s, buf); err != nil {
			srvErr <- err
			return
		}
		if _, err := s.Write(buf); err != nil {
			srvErr <- err
			return
		}
		srvErr <- s.Close()
	}()

	cs, err := clientConn.OpenStream(ctx)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	if _, err := cs.Write(msg); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := cs.Close(); err != nil {
		t.Fatalf("Close send side: %v", err)
	}

	got := make([]byte, len(msg))
	if _, err := io.ReadFull(cs, got); err != nil {
		t.Fatalf("Read echo: %v", err)
	}
	if string(got) != string(msg) {
		t.Errorf("echo = %q, want %q", got, msg)
	}
	if err := <-srvErr; err != nil {
		t.Fatalf("server stream: %v", err)
	}
}

func TestRejectsUnauthorizedPeer(t *testing.T) {
	server, _ := identity.Generate()
	client, _ := identity.Generate()

	ln, err := Listen(server, "127.0.0.1:0", nil)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()

	ctx := testContext(t)
	go func() { _, _ = ln.Accept(ctx) }()

	reject := func(core.DeviceID, *x509.Certificate) error {
		return errors.New("untrusted server")
	}
	if _, err := Dial(ctx, client, ln.Addr().String(), reject); err == nil {
		t.Error("Dial should fail when the client rejects the server")
	}
}
