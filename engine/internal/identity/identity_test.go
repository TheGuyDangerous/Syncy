package identity

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateProducesStableID(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if id.ID() == "" {
		t.Fatal("device ID must not be empty")
	}
	if got := len(string(id.ID())); got != 52 {
		t.Errorf("device ID length = %d, want 52", got)
	}
	if len(id.PublicKey()) != ed25519.PublicKeySize {
		t.Errorf("public key size = %d, want %d", len(id.PublicKey()), ed25519.PublicKeySize)
	}
	if id.ID() != DeviceIDFromPublicKey(id.PublicKey()) {
		t.Error("ID() disagrees with DeviceIDFromPublicKey")
	}
}

func TestDistinctIdentities(t *testing.T) {
	a, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	b, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if a.ID() == b.ID() {
		t.Error("two generated identities must have different IDs")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	original, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	path := filepath.Join(t.TempDir(), "device.key")
	if err := original.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.ID() != original.ID() {
		t.Errorf("loaded ID = %s, want %s", loaded.ID(), original.ID())
	}
	if !loaded.PublicKey().Equal(original.PublicKey()) {
		t.Error("loaded public key differs from original")
	}
}

func TestLoadOrCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "device.key")

	created, err := LoadOrCreate(path)
	if err != nil {
		t.Fatalf("LoadOrCreate (create): %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("key file was not written: %v", err)
	}
	loaded, err := LoadOrCreate(path)
	if err != nil {
		t.Fatalf("LoadOrCreate (load): %v", err)
	}
	if created.ID() != loaded.ID() {
		t.Errorf("LoadOrCreate is not stable: %s vs %s", created.ID(), loaded.ID())
	}
}

func TestLoadErrors(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.key")); err == nil {
		t.Error("Load of a missing file should error")
	}
	bad := filepath.Join(t.TempDir(), "bad.key")
	if err := os.WriteFile(bad, []byte("not a pem file"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := Load(bad); err == nil {
		t.Error("Load of non-PEM data should error")
	}
}

func TestTLSCertificateMatchesIdentity(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	cert, err := id.TLSCertificate()
	if err != nil {
		t.Fatalf("TLSCertificate: %v", err)
	}
	if cert.Leaf == nil {
		t.Fatal("certificate leaf must be parsed")
	}
	leafPub, ok := cert.Leaf.PublicKey.(ed25519.PublicKey)
	if !ok {
		t.Fatalf("leaf public key type = %T, want ed25519.PublicKey", cert.Leaf.PublicKey)
	}
	if !leafPub.Equal(id.PublicKey()) {
		t.Error("certificate public key differs from identity")
	}
	if DeviceIDFromPublicKey(leafPub) != id.ID() {
		t.Error("device ID derived from certificate differs from identity ID")
	}
	if cert.Leaf.Subject.CommonName != string(id.ID()) {
		t.Errorf("certificate CommonName = %q, want %q", cert.Leaf.Subject.CommonName, id.ID())
	}
}
