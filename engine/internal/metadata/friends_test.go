package metadata

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
)

func TestFriendRequestRoundTrip(t *testing.T) {
	s := newTestStore(t)
	want := core.FriendRequest{
		FromID:    "peer-1",
		Name:      "laptop",
		Endpoints: []string{"192.168.1.4:22067", "203.0.113.9:41000"},
		CreatedAt: time.Unix(1_700_000_000, 0).UTC(),
	}
	if err := s.PutFriendRequest(want); err != nil {
		t.Fatalf("PutFriendRequest: %v", err)
	}
	got, err := s.GetFriendRequest("peer-1")
	if err != nil {
		t.Fatalf("GetFriendRequest: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", got, want)
	}
}

func TestFriendRequestDedupesByDevice(t *testing.T) {
	s := newTestStore(t)
	first := core.FriendRequest{FromID: "peer-1", Name: "old", CreatedAt: time.Unix(100, 0).UTC()}
	if err := s.PutFriendRequest(first); err != nil {
		t.Fatalf("PutFriendRequest: %v", err)
	}
	update := core.FriendRequest{
		FromID:    "peer-1",
		Name:      "new",
		Endpoints: []string{"10.0.0.2:22067"},
		CreatedAt: time.Unix(200, 0).UTC(),
	}
	if err := s.PutFriendRequest(update); err != nil {
		t.Fatalf("second PutFriendRequest: %v", err)
	}

	reqs, err := s.ListFriendRequests()
	if err != nil {
		t.Fatalf("ListFriendRequests: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("got %d requests, want 1", len(reqs))
	}
	if reqs[0].Name != "new" || len(reqs[0].Endpoints) != 1 {
		t.Errorf("update not applied: %+v", reqs[0])
	}
	if !reqs[0].CreatedAt.Equal(first.CreatedAt) {
		t.Errorf("CreatedAt = %v, want original %v", reqs[0].CreatedAt, first.CreatedAt)
	}
}

func TestFriendRequestListOrdered(t *testing.T) {
	s := newTestStore(t)
	for _, fr := range []core.FriendRequest{
		{FromID: "b", CreatedAt: time.Unix(200, 0).UTC()},
		{FromID: "a", CreatedAt: time.Unix(100, 0).UTC()},
	} {
		if err := s.PutFriendRequest(fr); err != nil {
			t.Fatalf("PutFriendRequest: %v", err)
		}
	}
	reqs, err := s.ListFriendRequests()
	if err != nil {
		t.Fatalf("ListFriendRequests: %v", err)
	}
	if len(reqs) != 2 || reqs[0].FromID != "a" || reqs[1].FromID != "b" {
		t.Errorf("unexpected order: %+v", reqs)
	}
}

func TestFriendRequestRemove(t *testing.T) {
	s := newTestStore(t)
	if err := s.PutFriendRequest(core.FriendRequest{FromID: "peer-1"}); err != nil {
		t.Fatalf("PutFriendRequest: %v", err)
	}
	if err := s.RemoveFriendRequest("peer-1"); err != nil {
		t.Fatalf("RemoveFriendRequest: %v", err)
	}
	if _, err := s.GetFriendRequest("peer-1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("after remove, error = %v, want ErrNotFound", err)
	}
	if err := s.RemoveFriendRequest("peer-1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("removing missing request error = %v, want ErrNotFound", err)
	}
}

func TestFriendRequestEmptyID(t *testing.T) {
	s := newTestStore(t)
	if err := s.PutFriendRequest(core.FriendRequest{}); err == nil {
		t.Error("PutFriendRequest with empty ID should error")
	}
}

func TestSettingRoundTrip(t *testing.T) {
	s := newTestStore(t)
	if err := s.SetSetting("discovery", `{"local":true}`); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}
	got, err := s.GetSetting("discovery")
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if got != `{"local":true}` {
		t.Errorf("value = %q", got)
	}

	if err := s.SetSetting("discovery", `{"local":false}`); err != nil {
		t.Fatalf("overwrite SetSetting: %v", err)
	}
	got, err = s.GetSetting("discovery")
	if err != nil {
		t.Fatalf("GetSetting after overwrite: %v", err)
	}
	if got != `{"local":false}` {
		t.Errorf("value after overwrite = %q", got)
	}
}

func TestSettingMissing(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.GetSetting("nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
	if err := s.SetSetting("", "x"); err == nil {
		t.Error("SetSetting with empty key should error")
	}
}
