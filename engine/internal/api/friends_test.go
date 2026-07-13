package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
	"github.com/TheGuyDangerous/Syncy/engine/internal/identity"
	"github.com/TheGuyDangerous/Syncy/engine/internal/invite"
	"github.com/TheGuyDangerous/Syncy/engine/internal/metadata"
	"github.com/TheGuyDangerous/Syncy/engine/internal/syncengine"
)

func newTestServerWithEngine(t *testing.T) (*Server, *syncengine.Engine) {
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
	engine := syncengine.New(id, store)
	return New(engine, testToken, filepath.Join(t.TempDir(), "ai.json")), engine
}

func TestInvite(t *testing.T) {
	s := newTestServer(t)
	rec := do(t, s, "GET", "/invite", "", testToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /invite code = %d", rec.Code)
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	code, err := invite.Decode(body.Code)
	if err != nil {
		t.Fatalf("invite code should decode: %v", err)
	}
	if code.DeviceID != string(s.engine.ID()) {
		t.Errorf("code device = %q, want %q", code.DeviceID, s.engine.ID())
	}
}

func TestFriendFolders(t *testing.T) {
	s, engine := newTestServerWithEngine(t)

	if rec := do(t, s, "GET", "/friends/nobody/folders", "", testToken); rec.Code != http.StatusNotFound {
		t.Errorf("unknown device code = %d, want 404", rec.Code)
	}

	if err := engine.AddDevice(core.Device{ID: "stranger", Name: "Stranger"}); err != nil {
		t.Fatalf("AddDevice: %v", err)
	}
	if rec := do(t, s, "GET", "/friends/stranger/folders", "", testToken); rec.Code != http.StatusBadRequest {
		t.Errorf("untrusted device code = %d, want 400", rec.Code)
	}

	if err := engine.AddDevice(core.Device{ID: "friend", Name: "Friend", Trusted: true}); err != nil {
		t.Fatalf("AddDevice: %v", err)
	}
	if rec := do(t, s, "GET", "/friends/friend/folders", "", testToken); rec.Code != http.StatusBadGateway {
		t.Errorf("unreachable friend code = %d, want 502", rec.Code)
	}
}

func TestAddFriendByCode(t *testing.T) {
	s := newTestServer(t)

	if rec := do(t, s, "POST", "/friends", `not json`, testToken); rec.Code != http.StatusBadRequest {
		t.Errorf("bad json code = %d, want 400", rec.Code)
	}
	if rec := do(t, s, "POST", "/friends", `{"code":""}`, testToken); rec.Code != http.StatusBadRequest {
		t.Errorf("empty code = %d, want 400", rec.Code)
	}
	if rec := do(t, s, "POST", "/friends", `{"code":"garbage"}`, testToken); rec.Code != http.StatusBadRequest {
		t.Errorf("garbage code = %d, want 400", rec.Code)
	}

	own, _ := json.Marshal(map[string]string{"code": s.engine.InviteCode()})
	if rec := do(t, s, "POST", "/friends", string(own), testToken); rec.Code != http.StatusBadRequest {
		t.Errorf("own code = %d, want 400", rec.Code)
	}

	foreign, _ := json.Marshal(map[string]string{
		"code": invite.Encode(invite.Code{DeviceID: "REMOTEDEVICE", Name: "workstation"}),
	})
	rec := do(t, s, "POST", "/friends", string(foreign), testToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /friends code = %d, want 201 (%s)", rec.Code, rec.Body)
	}
	var out struct {
		Device    core.Device `json:"device"`
		Delivered bool        `json:"delivered"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Delivered {
		t.Error("an unreachable peer cannot have received the request")
	}
	if out.Device.Trusted || !out.Device.PendingOutgoing {
		t.Errorf("device = %+v, want untrusted pending-outgoing", out.Device)
	}
}

func TestFriendRequestLifecycle(t *testing.T) {
	s, engine := newTestServerWithEngine(t)
	if err := engine.RecordFriendRequest("peer-x", "laptop", nil); err != nil {
		t.Fatalf("RecordFriendRequest: %v", err)
	}

	rec := do(t, s, "GET", "/friend-requests", "", testToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /friend-requests code = %d", rec.Code)
	}
	var reqs []core.FriendRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &reqs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(reqs) != 1 || reqs[0].FromID != "peer-x" {
		t.Fatalf("requests = %+v, want one from peer-x", reqs)
	}

	rec = do(t, s, "POST", "/friend-requests/peer-x/accept", "", testToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("accept code = %d (%s)", rec.Code, rec.Body)
	}
	var out struct {
		Device core.Device `json:"device"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !out.Device.Trusted {
		t.Errorf("accepted device = %+v, want trusted", out.Device)
	}

	if rec := do(t, s, "GET", "/friend-requests", "", testToken); rec.Body.String() == "" || rec.Body.String()[0] != '[' {
		t.Errorf("friend request list should be a JSON array, got %s", rec.Body)
	}
	if rec := do(t, s, "POST", "/friend-requests/peer-x/accept", "", testToken); rec.Code != http.StatusNotFound {
		t.Errorf("accepting twice code = %d, want 404", rec.Code)
	}

	if err := engine.RecordFriendRequest("peer-y", "phone", nil); err != nil {
		t.Fatalf("RecordFriendRequest: %v", err)
	}
	if rec := do(t, s, "POST", "/friend-requests/peer-y/reject", "", testToken); rec.Code != http.StatusNoContent {
		t.Errorf("reject code = %d, want 204", rec.Code)
	}
	if rec := do(t, s, "POST", "/friend-requests/peer-y/reject", "", testToken); rec.Code != http.StatusNotFound {
		t.Errorf("rejecting twice code = %d, want 404", rec.Code)
	}
	devices, err := engine.Devices()
	if err != nil {
		t.Fatalf("Devices: %v", err)
	}
	for _, d := range devices {
		if d.ID == "peer-y" {
			t.Error("a rejected request must not create a device")
		}
	}
}

func TestDiscoverySettings(t *testing.T) {
	s := newTestServer(t)

	rec := do(t, s, "GET", "/settings/discovery", "", testToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET code = %d", rec.Code)
	}
	var settings core.DiscoverySettings
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !settings.Local || settings.Internet {
		t.Errorf("default settings = %+v, want local on and internet off", settings)
	}

	rec = do(t, s, "PUT", "/settings/discovery", `{"local":true,"internet":true}`, testToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT code = %d (%s)", rec.Code, rec.Body)
	}
	rec = do(t, s, "GET", "/settings/discovery", "", testToken)
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode after PUT: %v", err)
	}
	if !settings.Local || !settings.Internet {
		t.Errorf("settings after PUT = %+v, want both on", settings)
	}

	if rec := do(t, s, "PUT", "/settings/discovery", `nope`, testToken); rec.Code != http.StatusBadRequest {
		t.Errorf("bad body code = %d, want 400", rec.Code)
	}
}
