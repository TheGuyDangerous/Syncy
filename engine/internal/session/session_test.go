package session

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/chunker"
	"github.com/TheGuyDangerous/Syncy/engine/internal/identity"
	"github.com/TheGuyDangerous/Syncy/engine/internal/scanner"
	"github.com/TheGuyDangerous/Syncy/engine/internal/transport"
	"github.com/TheGuyDangerous/Syncy/engine/internal/versioning"
)

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

func writeFile(t *testing.T, dir, rel string, data []byte) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func scanDir(t *testing.T, dir string) *scanner.Index {
	t.Helper()
	ch, err := chunker.New(chunker.Config{Min: 2 * chunker.KiB, Avg: 8 * chunker.KiB, Max: 64 * chunker.KiB})
	if err != nil {
		t.Fatalf("chunker: %v", err)
	}
	sc, err := scanner.New(ch)
	if err != nil {
		t.Fatalf("scanner: %v", err)
	}
	idx, err := sc.Scan(dir)
	if err != nil {
		t.Fatalf("scan %s: %v", dir, err)
	}
	return idx
}

func syncDirs(t *testing.T, srcDir, dstDir string, opts ...Option) Stats {
	t.Helper()
	srcIdx := scanDir(t, srcDir)
	dstIdx := scanDir(t, dstDir)

	server, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	client, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity: %v", err)
	}

	ln, err := transport.Listen(server, "127.0.0.1:0", nil)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	go func() {
		conn, err := ln.Accept(ctx)
		if err != nil {
			return
		}
		_ = Serve(ctx, conn, Folder{ID: "f", Dir: srcDir, Index: srcIdx})
	}()

	conn, err := transport.Dial(ctx, client, ln.Addr().String(), nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	stats, err := Pull(ctx, conn, "f", dstDir, dstIdx, opts...)
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	return stats
}

func assertSameFile(t *testing.T, srcDir, dstDir, rel string) {
	t.Helper()
	want, err := os.ReadFile(filepath.Join(srcDir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dstDir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s: synced content (%d bytes) does not match source (%d bytes)", rel, len(got), len(want))
	}
}

func TestPullNewFile(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	writeFile(t, src, "docs/report.bin", deterministicBytes(30*chunker.KiB, 1))

	stats := syncDirs(t, src, dst)

	assertSameFile(t, src, dst, "docs/report.bin")
	if stats.FilesUpdated != 1 {
		t.Errorf("FilesUpdated = %d, want 1", stats.FilesUpdated)
	}
	if stats.BlocksFetched == 0 {
		t.Error("expected to fetch blocks for a brand-new file")
	}
	if stats.BlocksReused != 0 {
		t.Errorf("BlocksReused = %d, want 0 for a new file", stats.BlocksReused)
	}
}

func TestPullMultipleFiles(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	writeFile(t, src, "a.bin", deterministicBytes(12*chunker.KiB, 2))
	writeFile(t, src, "sub/b.bin", deterministicBytes(3*chunker.KiB, 3))
	writeFile(t, src, "empty.txt", nil)

	stats := syncDirs(t, src, dst)

	assertSameFile(t, src, dst, "a.bin")
	assertSameFile(t, src, dst, "sub/b.bin")
	assertSameFile(t, src, dst, "empty.txt")
	if stats.FilesUpdated != 3 {
		t.Errorf("FilesUpdated = %d, want 3", stats.FilesUpdated)
	}
}

func TestPullReusesLocalBlocks(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	full := deterministicBytes(40*chunker.KiB, 4)

	writeFile(t, src, "doc.bin", full)
	shared := append(append([]byte{}, full[:25*chunker.KiB]...), deterministicBytes(15*chunker.KiB, 5)...)
	writeFile(t, dst, "doc.bin", shared)

	stats := syncDirs(t, src, dst)

	assertSameFile(t, src, dst, "doc.bin")
	if stats.BlocksReused == 0 {
		t.Error("expected to reuse shared prefix blocks from the local file")
	}
	if stats.BlocksFetched == 0 {
		t.Error("expected to fetch the changed tail blocks")
	}
}

func TestPullArchivesOverwrittenFile(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	writeFile(t, src, "doc.txt", []byte("brand new content"))
	writeFile(t, dst, "doc.txt", []byte("the old content"))

	store := versioning.New(filepath.Join(dst, ".syncy-versions"), 0)
	syncDirs(t, src, dst, WithVersioning(store))

	assertSameFile(t, src, dst, "doc.txt")

	versions, err := store.Versions("doc.txt")
	if err != nil {
		t.Fatalf("Versions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected 1 archived version, got %d", len(versions))
	}
	if got := readFileAt(t, versions[0].Path); got != "the old content" {
		t.Errorf("archived content = %q, want %q", got, "the old content")
	}
}

func readFileAt(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	return string(b)
}

func TestPullSkipsUpToDateFiles(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	data := deterministicBytes(20*chunker.KiB, 6)
	writeFile(t, src, "same.bin", data)
	writeFile(t, dst, "same.bin", data)

	stats := syncDirs(t, src, dst)

	if stats.FilesUpdated != 0 {
		t.Errorf("FilesUpdated = %d, want 0 for identical files", stats.FilesUpdated)
	}
	if stats.BlocksFetched != 0 {
		t.Errorf("BlocksFetched = %d, want 0", stats.BlocksFetched)
	}
}
