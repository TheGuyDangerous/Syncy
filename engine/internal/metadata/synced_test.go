package metadata

import (
	"path/filepath"
	"testing"

	"github.com/TheGuyDangerous/Syncy/engine/internal/hashing"
)

func TestSyncedBaselineRoundTrip(t *testing.T) {
	s := newTestStore(t)
	baseline := map[string]hashing.Hash{
		"a.txt":     hashing.OfString("a"),
		"sub/b.bin": hashing.OfString("b"),
	}
	if err := s.SetSyncedBaseline("photos", baseline); err != nil {
		t.Fatalf("SetSyncedBaseline: %v", err)
	}
	got, err := s.GetSyncedBaseline("photos")
	if err != nil {
		t.Fatalf("GetSyncedBaseline: %v", err)
	}
	if len(got) != 2 || got["a.txt"] != baseline["a.txt"] || got["sub/b.bin"] != baseline["sub/b.bin"] {
		t.Errorf("baseline mismatch: %v", got)
	}
}

func TestSyncedBaselineReplaces(t *testing.T) {
	s := newTestStore(t)
	if err := s.SetSyncedBaseline("f", map[string]hashing.Hash{"x": hashing.OfString("1")}); err != nil {
		t.Fatalf("SetSyncedBaseline: %v", err)
	}
	if err := s.SetSyncedBaseline("f", map[string]hashing.Hash{"y": hashing.OfString("2")}); err != nil {
		t.Fatalf("SetSyncedBaseline: %v", err)
	}
	got, _ := s.GetSyncedBaseline("f")
	if len(got) != 1 {
		t.Fatalf("expected 1 entry after replace, got %d", len(got))
	}
	if _, ok := got["y"]; !ok {
		t.Error("replaced baseline should contain the new entry")
	}
}

func TestSyncedBaselineIsolatedPerFolder(t *testing.T) {
	s := newTestStore(t)
	s.SetSyncedBaseline("a", map[string]hashing.Hash{"x": hashing.OfString("1")})
	s.SetSyncedBaseline("b", map[string]hashing.Hash{"y": hashing.OfString("2")})
	if got, _ := s.GetSyncedBaseline("a"); len(got) != 1 {
		t.Errorf("folder a baseline size = %d, want 1", len(got))
	}
	if got, _ := s.GetSyncedBaseline("missing"); len(got) != 0 {
		t.Errorf("unknown folder baseline should be empty, got %d", len(got))
	}
}

func TestSyncedBaselinePersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "meta.db")
	s1, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	s1.SetSyncedBaseline("f", map[string]hashing.Hash{"x": hashing.OfString("v")})
	s1.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()
	if got, _ := s2.GetSyncedBaseline("f"); len(got) != 1 {
		t.Errorf("baseline did not persist across reopen: %d", len(got))
	}
}
