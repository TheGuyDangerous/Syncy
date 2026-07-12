package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testConfig(dir string) Config {
	return Config{DataDir: dir, ListenAddr: "127.0.0.1:0", APIAddr: "127.0.0.1:0"}
}

func TestNewPersistsIdentityAndToken(t *testing.T) {
	dir := t.TempDir()

	d, err := New(testConfig(dir))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	token, id := d.Token(), d.DeviceID()
	if token == "" || id == "" {
		t.Fatal("New should establish a token and device id")
	}
	for _, name := range []string{"device.key", "syncy.db", "api-token"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("expected %s in data dir: %v", name, err)
		}
	}
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	d2, err := New(testConfig(dir))
	if err != nil {
		t.Fatalf("second New: %v", err)
	}
	defer d2.Close()
	if d2.Token() != token {
		t.Error("token should persist across restarts")
	}
	if d2.DeviceID() != id {
		t.Error("device identity should persist across restarts")
	}
}

func TestRunReturnsOnCancel(t *testing.T) {
	d, err := New(testConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer d.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- d.Run(ctx) }()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}

func TestDefaultAddresses(t *testing.T) {
	d, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer d.Close()
	if d.cfg.ListenAddr == "" || d.cfg.APIAddr == "" {
		t.Error("New should fill in default listen and API addresses")
	}
}
