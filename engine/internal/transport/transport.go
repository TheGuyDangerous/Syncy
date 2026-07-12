// Package transport establishes authenticated QUIC connections between devices,
// exposing the peer's device ID and multiplexed streams.
package transport

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/quic-go/quic-go"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
	"github.com/TheGuyDangerous/Syncy/engine/internal/identity"
)

type Stream interface {
	io.Reader
	io.Writer
	io.Closer
}

type Conn struct {
	conn *quic.Conn
	peer core.DeviceID
}

func (c *Conn) Peer() core.DeviceID { return c.peer }

func (c *Conn) RemoteAddr() net.Addr { return c.conn.RemoteAddr() }

func (c *Conn) OpenStream(ctx context.Context) (Stream, error) {
	s, err := c.conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (c *Conn) AcceptStream(ctx context.Context) (Stream, error) {
	s, err := c.conn.AcceptStream(ctx)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (c *Conn) Close() error {
	return c.conn.CloseWithError(0, "")
}

type Listener struct {
	ln *quic.Listener
}

func Listen(id *identity.Identity, addr string, auth identity.PeerAuthenticator) (*Listener, error) {
	tlsConf, err := id.ServerTLSConfig(auth)
	if err != nil {
		return nil, err
	}
	ln, err := quic.ListenAddr(addr, tlsConf, quicConfig())
	if err != nil {
		return nil, err
	}
	return &Listener{ln: ln}, nil
}

func (l *Listener) Addr() net.Addr { return l.ln.Addr() }

func (l *Listener) Accept(ctx context.Context) (*Conn, error) {
	qc, err := l.ln.Accept(ctx)
	if err != nil {
		return nil, err
	}
	return wrap(qc)
}

func (l *Listener) Close() error { return l.ln.Close() }

func Dial(ctx context.Context, id *identity.Identity, addr string, auth identity.PeerAuthenticator) (*Conn, error) {
	tlsConf, err := id.ClientTLSConfig(auth)
	if err != nil {
		return nil, err
	}
	qc, err := quic.DialAddr(ctx, addr, tlsConf, quicConfig())
	if err != nil {
		return nil, err
	}
	return wrap(qc)
}

func wrap(qc *quic.Conn) (*Conn, error) {
	peer, err := peerID(qc)
	if err != nil {
		_ = qc.CloseWithError(1, "unauthenticated peer")
		return nil, err
	}
	return &Conn{conn: qc, peer: peer}, nil
}

func peerID(qc *quic.Conn) (core.DeviceID, error) {
	certs := qc.ConnectionState().TLS.PeerCertificates
	if len(certs) == 0 {
		return "", errors.New("transport: peer presented no certificate")
	}
	id, _, err := identity.PeerIdentity([][]byte{certs[0].Raw})
	return id, err
}

func quicConfig() *quic.Config {
	return &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 15 * time.Second,
	}
}
