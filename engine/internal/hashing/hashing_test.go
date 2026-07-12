package hashing

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Known SHA-256 vectors from FIPS 180-4 / RFC test data.
const (
	emptyHex = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	abcHex   = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
)

func TestOfBytesKnownVectors(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want string
	}{
		{"empty", []byte{}, emptyHex},
		{"abc", []byte("abc"), abcHex},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := OfBytes(tc.in).String(); got != tc.want {
				t.Errorf("OfBytes(%q) = %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}

func TestOfStringMatchesOfBytes(t *testing.T) {
	if OfString("abc") != OfBytes([]byte("abc")) {
		t.Error("OfString and OfBytes disagree for the same input")
	}
	if OfString("abc").String() != abcHex {
		t.Errorf("OfString(abc) = %s, want %s", OfString("abc"), abcHex)
	}
}

func TestOfReaderMatchesOfBytes(t *testing.T) {
	data := []byte("the quick brown fox jumps over the lazy dog")
	got, err := OfReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("OfReader returned error: %v", err)
	}
	if got != OfBytes(data) {
		t.Error("OfReader and OfBytes disagree for the same input")
	}
}

func TestOfReaderEmpty(t *testing.T) {
	got, err := OfReader(strings.NewReader(""))
	if err != nil {
		t.Fatalf("OfReader returned error: %v", err)
	}
	if got.String() != emptyHex {
		t.Errorf("OfReader(empty) = %s, want %s", got, emptyHex)
	}
}

func TestOfFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(path, []byte("abc"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := OfFile(path)
	if err != nil {
		t.Fatalf("OfFile returned error: %v", err)
	}
	if got.String() != abcHex {
		t.Errorf("OfFile = %s, want %s", got, abcHex)
	}
}

func TestOfFileMissing(t *testing.T) {
	if _, err := OfFile(filepath.Join(t.TempDir(), "nope.txt")); err == nil {
		t.Error("OfFile on missing file should return an error")
	}
}

func TestParseRoundTrip(t *testing.T) {
	original := OfString("round trip")
	parsed, err := Parse(original.String())
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if parsed != original {
		t.Error("Parse did not round-trip the hash")
	}
}

func TestParseInvalid(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"too short", "abc123"},
		{"too long", strings.Repeat("a", hexLen+2)},
		{"non-hex", strings.Repeat("z", hexLen)},
		{"empty", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Parse(tc.in); err == nil {
				t.Errorf("Parse(%q) should have failed", tc.in)
			}
		})
	}
}

func TestTextMarshalRoundTrip(t *testing.T) {
	original := OfString("marshal me")
	text, err := original.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	if string(text) != original.String() {
		t.Errorf("MarshalText = %s, want %s", text, original)
	}
	var restored Hash
	if err := restored.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}
	if restored != original {
		t.Error("text marshal did not round-trip")
	}
}

func TestUnmarshalTextInvalid(t *testing.T) {
	var h Hash
	if err := h.UnmarshalText([]byte("not-a-hash")); err == nil {
		t.Error("UnmarshalText should reject invalid input")
	}
}

func TestZeroAndEqual(t *testing.T) {
	var zero Hash
	if !zero.IsZero() {
		t.Error("zero value should report IsZero() == true")
	}
	nonZero := OfString("x")
	if nonZero.IsZero() {
		t.Error("non-zero hash should report IsZero() == false")
	}
	if !nonZero.Equal(OfString("x")) {
		t.Error("Equal should be true for identical content")
	}
	if nonZero.Equal(OfString("y")) {
		t.Error("Equal should be false for different content")
	}
}

func TestShortLength(t *testing.T) {
	if got := len(OfString("anything").Short()); got != shortLen {
		t.Errorf("Short() length = %d, want %d", got, shortLen)
	}
}

func BenchmarkOfBytes1KiB(b *testing.B) {
	data := bytes.Repeat([]byte("a"), 1024)
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		_ = OfBytes(data)
	}
}
