package metadata

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return s
}

func TestOpenAppliesMigrations(t *testing.T) {
	s := newTestStore(t)
	got, err := s.schemaVersion()
	if err != nil {
		t.Fatalf("schemaVersion: %v", err)
	}
	if got != len(migrations) {
		t.Errorf("schema version = %d, want %d", got, len(migrations))
	}
}

func TestReopenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "meta.db")

	s1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if err := s1.PutDevice(core.Device{ID: "dev-1", Name: "laptop"}); err != nil {
		t.Fatalf("PutDevice: %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer s2.Close()

	v, err := s2.schemaVersion()
	if err != nil {
		t.Fatalf("schemaVersion: %v", err)
	}
	if v != len(migrations) {
		t.Errorf("schema version after reopen = %d, want %d", v, len(migrations))
	}
	if _, err := s2.GetDevice("dev-1"); err != nil {
		t.Errorf("expected persisted device after reopen, got %v", err)
	}
}

func TestDeviceRoundTrip(t *testing.T) {
	s := newTestStore(t)
	want := core.Device{
		ID:       "device-abc",
		Name:     "Workstation",
		Trusted:  true,
		LastSeen: time.Unix(1_700_000_500, 0).UTC(),
		AddedAt:  time.Unix(1_700_000_000, 0).UTC(),
	}
	if err := s.PutDevice(want); err != nil {
		t.Fatalf("PutDevice: %v", err)
	}
	got, err := s.GetDevice(want.ID)
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got != want {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", got, want)
	}
}

func TestDeviceNeverSeenRoundTrip(t *testing.T) {
	s := newTestStore(t)
	d := core.Device{ID: "d", AddedAt: time.Unix(1_700_000_000, 0).UTC()}
	if err := s.PutDevice(d); err != nil {
		t.Fatalf("PutDevice: %v", err)
	}
	got, err := s.GetDevice("d")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if !got.LastSeen.IsZero() {
		t.Errorf("LastSeen = %v, want zero", got.LastSeen)
	}
}

func TestPutDeviceUpsertPreservesAddedAt(t *testing.T) {
	s := newTestStore(t)
	original := core.Device{ID: "d", Name: "old", AddedAt: time.Unix(1_700_000_000, 0).UTC()}
	if err := s.PutDevice(original); err != nil {
		t.Fatalf("PutDevice: %v", err)
	}
	updated := core.Device{ID: "d", Name: "new", Trusted: true, AddedAt: time.Unix(1_800_000_000, 0).UTC()}
	if err := s.PutDevice(updated); err != nil {
		t.Fatalf("PutDevice update: %v", err)
	}
	got, err := s.GetDevice("d")
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.Name != "new" || !got.Trusted {
		t.Errorf("update not applied: %+v", got)
	}
	if !got.AddedAt.Equal(original.AddedAt) {
		t.Errorf("AddedAt = %v, want preserved %v", got.AddedAt, original.AddedAt)
	}
}

func TestListDevicesOrdered(t *testing.T) {
	s := newTestStore(t)
	devs := []core.Device{
		{ID: "b", AddedAt: time.Unix(200, 0).UTC()},
		{ID: "a", AddedAt: time.Unix(100, 0).UTC()},
		{ID: "c", AddedAt: time.Unix(300, 0).UTC()},
	}
	for _, d := range devs {
		if err := s.PutDevice(d); err != nil {
			t.Fatalf("PutDevice: %v", err)
		}
	}
	got, err := s.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	wantOrder := []core.DeviceID{"a", "b", "c"}
	if len(got) != len(wantOrder) {
		t.Fatalf("got %d devices, want %d", len(got), len(wantOrder))
	}
	for i, id := range wantOrder {
		if got[i].ID != id {
			t.Errorf("position %d = %q, want %q", i, got[i].ID, id)
		}
	}
}

func TestGetDeviceNotFound(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.GetDevice("missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetDevice(missing) error = %v, want ErrNotFound", err)
	}
}

func TestRemoveDevice(t *testing.T) {
	s := newTestStore(t)
	if err := s.PutDevice(core.Device{ID: "d"}); err != nil {
		t.Fatalf("PutDevice: %v", err)
	}
	if err := s.RemoveDevice("d"); err != nil {
		t.Fatalf("RemoveDevice: %v", err)
	}
	if _, err := s.GetDevice("d"); !errors.Is(err, ErrNotFound) {
		t.Errorf("after remove, GetDevice error = %v, want ErrNotFound", err)
	}
	if err := s.RemoveDevice("d"); !errors.Is(err, ErrNotFound) {
		t.Errorf("removing missing device error = %v, want ErrNotFound", err)
	}
}

func TestPutDeviceEmptyID(t *testing.T) {
	s := newTestStore(t)
	if err := s.PutDevice(core.Device{ID: ""}); err == nil {
		t.Error("PutDevice with empty ID should error")
	}
}

func TestFolderRoundTripAndDefaults(t *testing.T) {
	s := newTestStore(t)
	in := core.Folder{
		ID:      "photos",
		Label:   "Photos",
		Path:    filepath.FromSlash("/home/user/Photos"),
		Paused:  true,
		AddedAt: time.Unix(1_700_000_000, 0).UTC(),
	}
	if err := s.PutFolder(in); err != nil {
		t.Fatalf("PutFolder: %v", err)
	}
	got, err := s.GetFolder("photos")
	if err != nil {
		t.Fatalf("GetFolder: %v", err)
	}
	if got.Direction != core.SendReceive {
		t.Errorf("Direction = %q, want default %q", got.Direction, core.SendReceive)
	}
	if got.Label != in.Label || got.Path != in.Path || !got.Paused {
		t.Errorf("folder round-trip mismatch: %+v", got)
	}
	if !got.AddedAt.Equal(in.AddedAt) {
		t.Errorf("AddedAt = %v, want %v", got.AddedAt, in.AddedAt)
	}
}

func TestPutFolderValidation(t *testing.T) {
	s := newTestStore(t)
	cases := map[string]core.Folder{
		"empty id":      {ID: "", Path: "/x"},
		"empty path":    {ID: "f", Path: ""},
		"bad direction": {ID: "f", Path: "/x", Direction: core.SyncDirection("sideways")},
	}
	for name, f := range cases {
		t.Run(name, func(t *testing.T) {
			if err := s.PutFolder(f); err == nil {
				t.Errorf("PutFolder(%+v) should have errored", f)
			}
		})
	}
}

func TestRemoveFolder(t *testing.T) {
	s := newTestStore(t)
	if err := s.PutFolder(core.Folder{ID: "f", Path: "/x"}); err != nil {
		t.Fatalf("PutFolder: %v", err)
	}
	if err := s.RemoveFolder("f"); err != nil {
		t.Fatalf("RemoveFolder: %v", err)
	}
	if err := s.RemoveFolder("f"); !errors.Is(err, ErrNotFound) {
		t.Errorf("removing missing folder error = %v, want ErrNotFound", err)
	}
}
