// Package identity manages a device's long-lived Ed25519 identity, from which
// its device ID and TLS certificate are derived.
package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base32"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
)

var idEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

type Identity struct {
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
	id   core.DeviceID

	certOnce sync.Once
	cert     tls.Certificate
	certErr  error
}

func Generate() (*Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return fromPrivate(priv, pub), nil
}

func fromPrivate(priv ed25519.PrivateKey, pub ed25519.PublicKey) *Identity {
	return &Identity{priv: priv, pub: pub, id: DeviceIDFromPublicKey(pub)}
}

func (i *Identity) ID() core.DeviceID { return i.id }

func (i *Identity) PublicKey() ed25519.PublicKey { return i.pub }

func DeviceIDFromPublicKey(pub ed25519.PublicKey) core.DeviceID {
	sum := sha256.Sum256(pub)
	return core.DeviceID(idEncoding.EncodeToString(sum[:]))
}

func (i *Identity) Save(path string) error {
	der, err := x509.MarshalPKCS8PrivateKey(i.priv)
	if err != nil {
		return err
	}
	block := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	return os.WriteFile(path, block, 0o600)
}

func Load(path string) (*Identity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("identity: no PEM data found")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	priv, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("identity: unexpected key type %T", key)
	}
	return fromPrivate(priv, priv.Public().(ed25519.PublicKey)), nil
}

func LoadOrCreate(path string) (*Identity, error) {
	id, err := Load(path)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	id, err = Generate()
	if err != nil {
		return nil, err
	}
	if err := id.Save(path); err != nil {
		return nil, err
	}
	return id, nil
}

func (i *Identity) TLSCertificate() (tls.Certificate, error) {
	i.certOnce.Do(func() { i.cert, i.certErr = i.buildCertificate() })
	return i.cert, i.certErr
}

func (i *Identity) buildCertificate() (tls.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: string(i.id)},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(100 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, i.pub, i.priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  i.priv,
		Leaf:        leaf,
	}, nil
}
