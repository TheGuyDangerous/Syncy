package syncengine

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
	"github.com/TheGuyDangerous/Syncy/engine/internal/identity"
	"github.com/TheGuyDangerous/Syncy/engine/internal/metadata"
	"github.com/TheGuyDangerous/Syncy/engine/internal/scanner"
	"github.com/TheGuyDangerous/Syncy/engine/internal/session"
	"github.com/TheGuyDangerous/Syncy/engine/internal/transport"
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
	if err := os.WriteFile(filepath.Join(dir, rel), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func scanDir(t *testing.T, dir string) *scanner.Index {
	t.Helper()
	sc, err := scanner.New(nil)
	if err != nil {
		t.Fatalf("scanner.New: %v", err)
	}
	idx, err := sc.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return idx
}

func TestBidirectionalConvergence(t *testing.T) {
	dirA, dirB := t.TempDir(), t.TempDir()
	data1 := deterministicBytes(9000, 1)
	data2 := deterministicBytes(7000, 2)
	writeFile(t, dirA, "file1.bin", data1)
	writeFile(t, dirB, "file2.bin", data2)

	idA, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	idB, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity: %v", err)
	}

	ln, err := transport.Listen(idA, "127.0.0.1:0", nil)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	errA := make(chan error, 1)
	go func() {
		conn, err := ln.Accept(ctx)
		if err != nil {
			errA <- err
			return
		}
		_, err = Converge(ctx, conn, session.Folder{ID: "f", Dir: dirA, Index: scanDir(t, dirA)})
		errA <- err
	}()

	connB, err := transport.Dial(ctx, idB, ln.Addr().String(), nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = connB.Close() })

	if _, err := Converge(ctx, connB, session.Folder{ID: "f", Dir: dirB, Index: scanDir(t, dirB)}); err != nil {
		t.Fatalf("Converge B: %v", err)
	}
	if err := <-errA; err != nil {
		t.Fatalf("Converge A: %v", err)
	}

	assertFile(t, dirB, "file1.bin", data1)
	assertFile(t, dirA, "file2.bin", data2)
	assertFile(t, dirA, "file1.bin", data1)
	assertFile(t, dirB, "file2.bin", data2)
}

func assertFile(t *testing.T, dir, rel string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatalf("read %s/%s: %v", dir, rel, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s: content mismatch (%d vs %d bytes)", rel, len(got), len(want))
	}
}

func newEngine(t *testing.T) *Engine {
	t.Helper()
	id, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	store, err := metadata.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return New(id, store)
}

func TestEngineFolderManagement(t *testing.T) {
	e := newEngine(t)
	if err := e.AddFolder(core.Folder{ID: "photos", Path: "/data/photos"}); err != nil {
		t.Fatalf("AddFolder: %v", err)
	}
	folders, err := e.Folders()
	if err != nil {
		t.Fatalf("Folders: %v", err)
	}
	if len(folders) != 1 || folders[0].ID != "photos" {
		t.Fatalf("unexpected folders: %+v", folders)
	}
	if _, err := e.GetFolder("photos"); err != nil {
		t.Errorf("GetFolder: %v", err)
	}
	if err := e.RemoveFolder("photos"); err != nil {
		t.Fatalf("RemoveFolder: %v", err)
	}
	folders, _ = e.Folders()
	if len(folders) != 0 {
		t.Errorf("expected no folders after removal, got %d", len(folders))
	}
}

func runSync(t *testing.T, ctx context.Context, idA, idB *identity.Identity, engA, engB *Engine, folderID string) {
	t.Helper()
	ln, err := transport.Listen(idA, "127.0.0.1:0", nil)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()

	errA := make(chan error, 1)
	go func() {
		conn, err := ln.Accept(ctx)
		if err != nil {
			errA <- err
			return
		}
		_, err = engA.Sync(ctx, conn, folderID)
		errA <- err
	}()

	connB, err := transport.Dial(ctx, idB, ln.Addr().String(), nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer connB.Close()

	if _, err := engB.Sync(ctx, connB, folderID); err != nil {
		t.Fatalf("engB.Sync: %v", err)
	}
	if err := <-errA; err != nil {
		t.Fatalf("engA.Sync: %v", err)
	}
}

func TestEngineSyncBaselineAndConflict(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	dirA, dirB := t.TempDir(), t.TempDir()
	idA, _ := identity.Generate()
	idB, _ := identity.Generate()
	storeA, _ := metadata.OpenMemory()
	storeB, _ := metadata.OpenMemory()
	t.Cleanup(func() { storeA.Close(); storeB.Close() })
	engA, engB := New(idA, storeA), New(idB, storeB)

	if err := engA.AddFolder(core.Folder{ID: "f", Path: dirA}); err != nil {
		t.Fatalf("AddFolder A: %v", err)
	}
	if err := engB.AddFolder(core.Folder{ID: "f", Path: dirB}); err != nil {
		t.Fatalf("AddFolder B: %v", err)
	}

	writeFile(t, dirA, "file.txt", []byte("v1"))
	runSync(t, ctx, idA, idB, engA, engB, "f")

	assertFile(t, dirB, "file.txt", []byte("v1"))
	base, _ := storeB.GetSyncedBaseline("f")
	if _, ok := base["file.txt"]; !ok {
		t.Fatal("baseline should record file.txt after the first sync")
	}

	writeFile(t, dirA, "file.txt", []byte("edit-from-A"))
	writeFile(t, dirB, "file.txt", []byte("edit-from-B"))
	runSync(t, ctx, idA, idB, engA, engB, "f")

	matches, _ := filepath.Glob(filepath.Join(dirB, "file.sync-conflict-*.txt"))
	if len(matches) == 0 {
		t.Error("expected a conflict copy on B after concurrent edits on both sides")
	}
	assertFile(t, dirB, "file.txt", []byte("edit-from-B"))
}

func TestEngineDeviceManagement(t *testing.T) {
	e := newEngine(t)
	if err := e.AddDevice(core.Device{ID: "peer-1", Name: "phone"}); err != nil {
		t.Fatalf("AddDevice: %v", err)
	}
	devices, err := e.Devices()
	if err != nil {
		t.Fatalf("Devices: %v", err)
	}
	if len(devices) != 1 || devices[0].ID != "peer-1" {
		t.Fatalf("unexpected devices: %+v", devices)
	}
	if e.ID() == "" {
		t.Error("engine device ID must not be empty")
	}
}
