package monitor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/fswatch"
	"github.com/TheGuyDangerous/Syncy/engine/internal/scanner"
)

const testTimeout = 15 * time.Second

func newTestMonitor(t *testing.T, dir string) *Monitor {
	t.Helper()
	m, _, err := New(dir, nil, fswatch.WithDebounce(50*time.Millisecond))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })
	return m
}

func waitForChange(t *testing.T, m *Monitor, kind scanner.ChangeKind, base string) scanner.Change {
	t.Helper()
	deadline := time.After(testTimeout)
	for {
		select {
		case set, ok := <-m.Changes():
			if !ok {
				t.Fatal("changes channel closed before the expected change")
			}
			for _, c := range set {
				if c.Kind == kind && filepath.Base(c.Path) == base {
					return c
				}
			}
		case err := <-m.Errors():
			t.Fatalf("monitor error: %v", err)
		case <-deadline:
			t.Fatalf("timed out waiting for %s %q", kind, base)
		}
	}
}

func TestMonitorBaseline(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("seed"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	m, baseline, err := New(dir, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer m.Close()
	if _, ok := baseline.Files["seed.txt"]; !ok {
		t.Error("baseline index should contain seed.txt")
	}
}

func TestMonitorDetectsCreate(t *testing.T) {
	dir := t.TempDir()
	m := newTestMonitor(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "fresh.txt"), []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	waitForChange(t, m, scanner.Added, "fresh.txt")
}

func TestMonitorDetectsDelete(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(target, []byte("bye"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	m := newTestMonitor(t, dir)

	if err := os.Remove(target); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	waitForChange(t, m, scanner.Deleted, "victim.txt")
}

func TestMonitorDetectsRename(t *testing.T) {
	dir := t.TempDir()
	m := newTestMonitor(t, dir)

	original := filepath.Join(dir, "before.txt")
	if err := os.WriteFile(original, []byte("stable content for rename"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	waitForChange(t, m, scanner.Added, "before.txt")

	if err := os.Rename(original, filepath.Join(dir, "after.txt")); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	c := waitForChange(t, m, scanner.Renamed, "after.txt")
	if filepath.Base(c.OldPath) != "before.txt" {
		t.Errorf("rename OldPath = %q, want before.txt", c.OldPath)
	}
}
