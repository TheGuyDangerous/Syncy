package versioning

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeAt(t *testing.T, dir, rel string, data []byte, mod time.Time) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chtimes(full, mod, mod); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	return b
}

func newStore(t *testing.T, keep int) (*Store, string) {
	t.Helper()
	root := t.TempDir()
	return New(filepath.Join(root, ".syncy-versions"), keep), root
}

func TestArchiveAndList(t *testing.T) {
	s, root := newStore(t, 0)
	mod := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	writeAt(t, root, "docs/report.txt", []byte("version one"), mod)

	if err := s.Archive(root, "docs/report.txt"); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	versions, err := s.Versions("docs/report.txt")
	if err != nil {
		t.Fatalf("Versions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("got %d versions, want 1", len(versions))
	}
	if string(readFile(t, versions[0].Path)) != "version one" {
		t.Error("archived content mismatch")
	}
}

func TestArchiveMultipleVersionsNewestFirst(t *testing.T) {
	s, root := newStore(t, 0)
	t1 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)

	writeAt(t, root, "a.txt", []byte("first"), t1)
	if err := s.Archive(root, "a.txt"); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	writeAt(t, root, "a.txt", []byte("second"), t2)
	if err := s.Archive(root, "a.txt"); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	versions, err := s.Versions("a.txt")
	if err != nil {
		t.Fatalf("Versions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("got %d versions, want 2", len(versions))
	}
	if string(readFile(t, versions[0].Path)) != "second" {
		t.Error("newest version should be listed first")
	}
	if string(readFile(t, versions[1].Path)) != "first" {
		t.Error("oldest version content mismatch")
	}
}

func TestRestore(t *testing.T) {
	s, root := newStore(t, 0)
	mod := time.Date(2026, 3, 4, 9, 30, 0, 0, time.UTC)
	writeAt(t, root, "keep.txt", []byte("original"), mod)
	if err := s.Archive(root, "keep.txt"); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("changed"), 0o644); err != nil {
		t.Fatalf("overwrite: %v", err)
	}

	versions, _ := s.Versions("keep.txt")
	if err := s.Restore(root, "keep.txt", versions[0].Stamp); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if got := readFile(t, filepath.Join(root, "keep.txt")); string(got) != "original" {
		t.Errorf("restored content = %q, want original", got)
	}
}

func TestPruneKeepsNewest(t *testing.T) {
	s, root := newStore(t, 2)
	for i, day := range []int{1, 2, 3} {
		mod := time.Date(2026, 1, day, 12, 0, 0, 0, time.UTC)
		writeAt(t, root, "f.txt", []byte{byte('a' + i)}, mod)
		if err := s.Archive(root, "f.txt"); err != nil {
			t.Fatalf("Archive: %v", err)
		}
	}
	versions, err := s.Versions("f.txt")
	if err != nil {
		t.Fatalf("Versions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("got %d versions, want 2 after prune", len(versions))
	}
	if string(readFile(t, versions[0].Path)) != "c" {
		t.Error("newest kept version should be 'c'")
	}
}

func TestVersionsEmpty(t *testing.T) {
	s, _ := newStore(t, 0)
	versions, err := s.Versions("never.txt")
	if err != nil {
		t.Fatalf("Versions: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("expected no versions, got %d", len(versions))
	}
}

func TestArchiveMissingFile(t *testing.T) {
	s, root := newStore(t, 0)
	if err := s.Archive(root, "nope.txt"); err == nil {
		t.Error("archiving a missing file should error")
	}
}

func TestArchiveSameStampUniquifies(t *testing.T) {
	s, root := newStore(t, 0)
	mod := time.Date(2026, 5, 5, 5, 5, 5, 0, time.UTC)
	writeAt(t, root, "dup.txt", []byte("one"), mod)
	if err := s.Archive(root, "dup.txt"); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	writeAt(t, root, "dup.txt", []byte("two"), mod)
	if err := s.Archive(root, "dup.txt"); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	versions, err := s.Versions("dup.txt")
	if err != nil {
		t.Fatalf("Versions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("got %d versions, want 2 (uniquified same stamp)", len(versions))
	}
}
