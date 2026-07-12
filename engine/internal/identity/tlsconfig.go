package identity

import (
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"errors"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
)

const ALPN = "syncy/1"

// PeerAuthenticator decides whether a connecting peer, identified by its
// certificate-derived device ID, is allowed to proceed.
type PeerAuthenticator func(peerID core.DeviceID, cert *x509.Certificate) error

func (i *Identity) ServerTLSConfig(auth PeerAuthenticator) (*tls.Config, error) {
	cert, err := i.TLSCertificate()
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:          []tls.Certificate{cert},
		MinVersion:            tls.VersionTLS13,
		ClientAuth:            tls.RequireAnyClientCert,
		InsecureSkipVerify:    true,
		VerifyPeerCertificate: peerVerifier(auth),
		NextProtos:            []string{ALPN},
	}, nil
}

func (i *Identity) ClientTLSConfig(auth PeerAuthenticator) (*tls.Config, error) {
	cert, err := i.TLSCertificate()
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:          []tls.Certificate{cert},
		MinVersion:            tls.VersionTLS13,
		InsecureSkipVerify:    true,
		VerifyPeerCertificate: peerVerifier(auth),
		NextProtos:            []string{ALPN},
	}, nil
}

func peerVerifier(auth PeerAuthenticator) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		id, cert, err := PeerIdentity(rawCerts)
		if err != nil {
			return err
		}
		if auth != nil {
			return auth(id, cert)
		}
		return nil
	}
}

// PeerIdentity derives a device ID from the peer's leaf certificate.
func PeerIdentity(rawCerts [][]byte) (core.DeviceID, *x509.Certificate, error) {
	if len(rawCerts) == 0 {
		return "", nil, errors.New("identity: peer presented no certificate")
	}
	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return "", nil, err
	}
	pub, ok := cert.PublicKey.(ed25519.PublicKey)
	if !ok {
		return "", nil, errors.New("identity: peer certificate is not ed25519")
	}
	return DeviceIDFromPublicKey(pub), cert, nil
}
