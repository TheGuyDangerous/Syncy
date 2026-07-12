package fswatch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testTimeout = 10 * time.Second

func newTestWatcher(t *testing.T, dir string) *Watcher {
	t.Helper()
	w, err := New(dir, WithDebounce(50*time.Millisecond))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

func waitForBasenames(t *testing.T, w *Watcher, want ...string) {
	t.Helper()
	remaining := make(map[string]bool, len(want))
	for _, n := range want {
		remaining[n] = true
	}
	deadline := time.After(testTimeout)
	for len(remaining) > 0 {
		select {
		case ev, ok := <-w.Events():
			if !ok {
				t.Fatal("events channel closed before all changes were seen")
			}
			for _, p := range ev.Paths {
				delete(remaining, filepath.Base(p))
			}
		case err := <-w.Errors():
			t.Fatalf("watcher error: %v", err)
		case <-deadline:
			t.Fatalf("timed out waiting for changes; still missing %v", keys(remaining))
		}
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestWatchFileCreation(t *testing.T) {
	dir := t.TempDir()
	w := newTestWatcher(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "created.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	waitForBasenames(t, w, "created.txt")
}

func TestWatchMultipleFilesDebounced(t *testing.T) {
	dir := t.TempDir()
	w := newTestWatcher(t, dir)

	for _, name := range []string{"one.txt", "two.txt", "three.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}
	waitForBasenames(t, w, "one.txt", "two.txt", "three.txt")
}

func TestWatchNewSubdirectory(t *testing.T) {
	dir := t.TempDir()
	w := newTestWatcher(t, dir)

	sub := filepath.Join(dir, "nested")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	waitForBasenames(t, w, "nested")

	if err := os.WriteFile(filepath.Join(sub, "inner.txt"), []byte("y"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	waitForBasenames(t, w, "inner.txt")
}

func TestCloseIsIdempotentAndClosesEvents(t *testing.T) {
	dir := t.TempDir()
	w, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
	select {
	case _, ok := <-w.Events():
		if ok {
			for range w.Events() {
			}
		}
	case <-time.After(testTimeout):
		t.Fatal("events channel was not closed after Close")
	}
}

func TestNewMissingDirectoryErrors(t *testing.T) {
	if _, err := New(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("New on a missing directory should error")
	}
}
