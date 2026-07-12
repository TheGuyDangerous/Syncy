package fsafe

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestLocal(t *testing.T) {
	safe := []string{"a.txt", "sub/dir/file.bin", "docs/report.txt", "a.b/c"}
	unsafe := []string{"../evil", "a/../../b", "/etc/passwd", "sub/../../out", ".."}

	for _, p := range safe {
		if !Local(p) {
			t.Errorf("Local(%q) = false, want true", p)
		}
	}
	for _, p := range unsafe {
		if Local(p) {
			t.Errorf("Local(%q) = true, want false", p)
		}
	}
}

func TestJoinRejectsEscape(t *testing.T) {
	base := filepath.FromSlash("/data/photos")
	if _, err := Join(base, "../secret"); !errors.Is(err, ErrUnsafePath) {
		t.Errorf("Join escape error = %v, want ErrUnsafePath", err)
	}
	if _, err := Join(base, "a/../../../etc"); !errors.Is(err, ErrUnsafePath) {
		t.Errorf("Join deep escape error = %v, want ErrUnsafePath", err)
	}
}

func TestJoinAllowsLocal(t *testing.T) {
	base := filepath.FromSlash("/data/photos")
	got, err := Join(base, "album/pic.jpg")
	if err != nil {
		t.Fatalf("Join: %v", err)
	}
	want := filepath.Join(base, filepath.FromSlash("album/pic.jpg"))
	if got != want {
		t.Errorf("Join = %q, want %q", got, want)
	}
}
