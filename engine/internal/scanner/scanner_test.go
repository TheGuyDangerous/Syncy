package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheGuyDangerous/Syncy/engine/internal/chunker"
	"github.com/TheGuyDangerous/Syncy/engine/internal/hashing"
)

func smallChunker(t *testing.T) *chunker.Chunker {
	t.Helper()
	c, err := chunker.New(chunker.Config{Min: 1 * chunker.KiB, Avg: 4 * chunker.KiB, Max: 16 * chunker.KiB})
	if err != nil {
		t.Fatalf("chunker.New: %v", err)
	}
	return c
}

func deterministicBytes(n int, seed uint64) []byte {
	out := make([]byte, n)
	x := seed
	for i := range out {
		x += 0x9E3779B97F4A7C15
		z := x
		z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
		z = (z ^ (z >> 27)) * 0x94D049BB133111EB
		z ^= z >> 31
		out[i] = byte(z)
	}
	return out
}

func writeFile(t *testing.T, root, rel string, data []byte) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func newScanner(t *testing.T, opts ...Option) *Scanner {
	t.Helper()
	s, err := New(smallChunker(t), opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func TestScanBuildsIndex(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"a.txt":          deterministicBytes(20*chunker.KiB, 1),
		"sub/b.bin":      deterministicBytes(9*chunker.KiB, 2),
		"sub/deep/c.dat": deterministicBytes(1, 3),
		"empty.txt":      {},
	}
	for rel, data := range files {
		writeFile(t, root, rel, data)
	}

	idx, err := newScanner(t).Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(idx.Files) != len(files) {
		t.Fatalf("indexed %d files, want %d", len(idx.Files), len(files))
	}

	for rel, data := range files {
		fi, ok := idx.Files[rel]
		if !ok {
			t.Fatalf("missing index entry for %q (keys must be slash-separated)", rel)
		}
		if fi.Size != int64(len(data)) {
			t.Errorf("%q size = %d, want %d", rel, fi.Size, len(data))
		}
		if fi.Hash != hashing.OfBytes(data) {
			t.Errorf("%q whole-file hash mismatch", rel)
		}
		var offset int64
		for _, b := range fi.Blocks {
			if b.Offset != offset {
				t.Fatalf("%q block gap/overlap at offset %d (want %d)", rel, b.Offset, offset)
			}
			if b.Hash != hashing.OfBytes(data[offset:offset+int64(b.Length)]) {
				t.Fatalf("%q block hash mismatch at offset %d", rel, offset)
			}
			offset += int64(b.Length)
		}
		if offset != int64(len(data)) {
			t.Errorf("%q blocks cover %d bytes, want %d", rel, offset, len(data))
		}
	}
}

func TestScanEmptyFileHasNoBlocks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "empty", nil)
	idx, err := newScanner(t).Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	fi := idx.Files["empty"]
	if len(fi.Blocks) != 0 {
		t.Errorf("empty file should have 0 blocks, got %d", len(fi.Blocks))
	}
	if !fi.Hash.Equal(hashing.OfBytes(nil)) {
		t.Error("empty file hash mismatch")
	}
}

func TestScanDeterministic(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "x", deterministicBytes(30*chunker.KiB, 4))

	s := newScanner(t)
	a, err := s.Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	b, err := s.Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	fa, fb := a.Files["x"], b.Files["x"]
	if fa.Hash != fb.Hash || len(fa.Blocks) != len(fb.Blocks) {
		t.Fatal("scans of identical content differ")
	}
	for i := range fa.Blocks {
		if fa.Blocks[i] != fb.Blocks[i] {
			t.Fatalf("block %d differs between scans", i)
		}
	}
}

func TestScanSkip(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "keep.txt", deterministicBytes(2*chunker.KiB, 5))
	writeFile(t, root, "skip.log", deterministicBytes(2*chunker.KiB, 6))
	writeFile(t, root, ".git/config", []byte("ignored"))

	skip := func(rel string, d fs.DirEntry) bool {
		return rel == ".git" || filepath.Ext(rel) == ".log"
	}
	idx, err := newScanner(t, WithSkip(skip)).Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if _, ok := idx.Files["keep.txt"]; !ok {
		t.Error("keep.txt should be indexed")
	}
	if _, ok := idx.Files["skip.log"]; ok {
		t.Error("skip.log should have been skipped")
	}
	for k := range idx.Files {
		if len(k) >= 4 && k[:4] == ".git" {
			t.Errorf("entry under skipped dir was indexed: %q", k)
		}
	}
	if len(idx.Files) != 1 {
		t.Errorf("expected exactly 1 indexed file, got %d", len(idx.Files))
	}
}

func TestScanErrors(t *testing.T) {
	s := newScanner(t)

	if _, err := s.Scan(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Error("scanning a missing root should error")
	}

	file := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := s.Scan(file); err == nil {
		t.Error("scanning a file (not a directory) should error")
	}
}
