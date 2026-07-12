package buildinfo

import (
	"runtime"
	"strings"
	"testing"
)

func TestStringContainsKeyFields(t *testing.T) {
	got := String()

	for _, want := range []string{Version, Commit, Date, runtime.GOOS, runtime.GOARCH} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, want it to contain %q", got, want)
		}
	}
}

func TestDefaultsNotEmpty(t *testing.T) {
	if Version == "" || Commit == "" || Date == "" {
		t.Fatal("build info defaults must not be empty")
	}
}
