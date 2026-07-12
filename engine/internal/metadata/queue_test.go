package metadata

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
)

func TestEnqueueAndPending(t *testing.T) {
	s := newTestStore(t)
	for _, op := range []core.QueuedOp{
		{DeviceID: "a", FolderID: "photos", Kind: "index-changed"},
		{DeviceID: "a", FolderID: "docs", Kind: "pull-needed"},
		{DeviceID: "b", FolderID: "photos", Kind: "index-changed"},
	} {
		if _, err := s.EnqueueOp(op); err != nil {
			t.Fatalf("EnqueueOp: %v", err)
		}
	}

	pending, err := s.PendingOps("a")
	if err != nil {
		t.Fatalf("PendingOps: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("PendingOps(a) = %d, want 2", len(pending))
	}
	if pending[0].Kind != "index-changed" || pending[1].Kind != "pull-needed" {
		t.Error("pending ops out of insertion order")
	}
	if n, _ := s.QueueLen("a"); n != 2 {
		t.Errorf("QueueLen(a) = %d, want 2", n)
	}
	if n, _ := s.QueueLen("b"); n != 1 {
		t.Errorf("QueueLen(b) = %d, want 1", n)
	}
}

func TestQueueRoundTrip(t *testing.T) {
	s := newTestStore(t)
	want := core.QueuedOp{
		DeviceID:  "device-x",
		FolderID:  "media",
		Kind:      "pull-needed",
		Payload:   `{"path":"a/b.jpg"}`,
		CreatedAt: time.Unix(1_700_000_000, 0).UTC(),
		Attempts:  3,
	}
	id, err := s.EnqueueOp(want)
	if err != nil {
		t.Fatalf("EnqueueOp: %v", err)
	}
	pending, err := s.PendingOps("device-x")
	if err != nil {
		t.Fatalf("PendingOps: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("got %d ops, want 1", len(pending))
	}
	got := pending[0]
	if got.ID != id {
		t.Errorf("ID = %d, want %d", got.ID, id)
	}
	want.ID = id
	if got != want {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", got, want)
	}
}

func TestCompleteOp(t *testing.T) {
	s := newTestStore(t)
	id, err := s.EnqueueOp(core.QueuedOp{DeviceID: "a", Kind: "x"})
	if err != nil {
		t.Fatalf("EnqueueOp: %v", err)
	}
	if err := s.CompleteOp(id); err != nil {
		t.Fatalf("CompleteOp: %v", err)
	}
	if n, _ := s.QueueLen("a"); n != 0 {
		t.Errorf("queue not empty after complete: %d", n)
	}
	if err := s.CompleteOp(id); !errors.Is(err, ErrNotFound) {
		t.Errorf("completing a missing op = %v, want ErrNotFound", err)
	}
}

func TestIncrementAttempts(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.EnqueueOp(core.QueuedOp{DeviceID: "a", Kind: "x"})
	if err := s.IncrementAttempts(id); err != nil {
		t.Fatalf("IncrementAttempts: %v", err)
	}
	if err := s.IncrementAttempts(id); err != nil {
		t.Fatalf("IncrementAttempts: %v", err)
	}
	pending, _ := s.PendingOps("a")
	if pending[0].Attempts != 2 {
		t.Errorf("attempts = %d, want 2", pending[0].Attempts)
	}
}

func TestQueuePersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "meta.db")
	s1, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := s1.EnqueueOp(core.QueuedOp{DeviceID: "a", Kind: "index-changed"}); err != nil {
		t.Fatalf("EnqueueOp: %v", err)
	}
	s1.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()
	if n, _ := s2.QueueLen("a"); n != 1 {
		t.Errorf("queued op did not persist across reopen: %d", n)
	}
}

func TestAllPendingOps(t *testing.T) {
	s := newTestStore(t)
	s.EnqueueOp(core.QueuedOp{DeviceID: "a", Kind: "x"})
	s.EnqueueOp(core.QueuedOp{DeviceID: "b", Kind: "y"})
	all, err := s.AllPendingOps()
	if err != nil {
		t.Fatalf("AllPendingOps: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("AllPendingOps = %d, want 2", len(all))
	}
}
