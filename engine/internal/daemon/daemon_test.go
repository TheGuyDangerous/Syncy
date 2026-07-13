package daemon

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
	"github.com/TheGuyDangerous/Syncy/engine/internal/identity"
	"github.com/TheGuyDangerous/Syncy/engine/internal/protocol"
	"github.com/TheGuyDangerous/Syncy/engine/internal/transport"
)

func testConfig(dir string) Config {
	return Config{DataDir: dir, ListenAddr: "127.0.0.1:0", APIAddr: "127.0.0.1:0"}
}

func TestNewPersistsIdentityAndToken(t *testing.T) {
	dir := t.TempDir()

	d, err := New(testConfig(dir))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	token, id := d.Token(), d.DeviceID()
	if token == "" || id == "" {
		t.Fatal("New should establish a token and device id")
	}
	for _, name := range []string{"device.key", "syncy.db", "api-token"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("expected %s in data dir: %v", name, err)
		}
	}
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	d2, err := New(testConfig(dir))
	if err != nil {
		t.Fatalf("second New: %v", err)
	}
	defer d2.Close()
	if d2.Token() != token {
		t.Error("token should persist across restarts")
	}
	if d2.DeviceID() != id {
		t.Error("device identity should persist across restarts")
	}
}

func TestRunReturnsOnCancel(t *testing.T) {
	d, err := New(testConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer d.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- d.Run(ctx) }()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}

func TestDefaultAddresses(t *testing.T) {
	d, err := New(Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer d.Close()
	if d.cfg.ListenAddr == "" || d.cfg.APIAddr == "" {
		t.Error("New should fill in default listen and API addresses")
	}
}

func startPeer(t *testing.T, ctx context.Context, d *Daemon) string {
	t.Helper()
	ln, err := transport.Listen(d.id, "127.0.0.1:0", d.authorize)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go d.acceptLoop(ctx, ln)
	addr := ln.Addr().String()
	d.engine.SetEndpointSource(func() []string { return []string{addr} })
	return addr
}

func newPeer(t *testing.T, ctx context.Context) (*Daemon, string) {
	t.Helper()
	d, err := New(testConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d, startPeer(t, ctx, d)
}

func requestFolderIndex(t *testing.T, ctx context.Context, from *Daemon, to *Daemon, addr string) protocol.Frame {
	t.Helper()
	conn, err := transport.Dial(ctx, from.id, addr, identity.ExpectPeer(to.id.ID()))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()
	s, err := conn.OpenStream(ctx)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer s.Close()
	if err := protocol.WriteMessage(s, protocol.TypeFolderSummary, protocol.FolderSummary{FolderID: "f"}); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	frame, err := protocol.ReadFrame(s)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	return frame
}

func TestFriendHandshakeAndSync(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	a, _ := newPeer(t, ctx)
	b, addrB := newPeer(t, ctx)

	dirA, dirB := t.TempDir(), t.TempDir()
	content := []byte("shared across the internet")
	if err := os.WriteFile(filepath.Join(dirB, "doc.txt"), content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	for d, dir := range map[*Daemon]string{a: dirA, b: dirB} {
		if err := d.store.PutFolder(core.Folder{ID: "f", Path: dir}); err != nil {
			t.Fatalf("PutFolder: %v", err)
		}
	}

	dev, delivered, err := a.engine.AddFriendByCode(ctx, b.engine.InviteCode())
	if err != nil {
		t.Fatalf("AddFriendByCode: %v", err)
	}
	if !delivered {
		t.Fatal("friend request should reach the online peer")
	}
	if dev.Trusted || !dev.PendingOutgoing {
		t.Fatalf("device after request = %+v, want untrusted pending-outgoing", dev)
	}

	reqs, err := b.engine.FriendRequests()
	if err != nil {
		t.Fatalf("FriendRequests: %v", err)
	}
	if len(reqs) != 1 || reqs[0].FromID != a.DeviceID() {
		t.Fatalf("pending requests on B = %+v, want one from A", reqs)
	}
	if len(reqs[0].Endpoints) == 0 {
		t.Error("the stored request should carry the requester's endpoints")
	}
	if b.trusted(a.DeviceID()) {
		t.Fatal("B must not trust A before the request is accepted")
	}

	if frame := requestFolderIndex(t, ctx, a, b, addrB); frame.Type != protocol.TypeError {
		t.Fatalf("untrusted index request answered with %s, want error", frame.Type)
	}

	accepted, notified, err := b.engine.AcceptFriendRequest(ctx, a.DeviceID())
	if err != nil {
		t.Fatalf("AcceptFriendRequest: %v", err)
	}
	if !accepted.Trusted || !notified {
		t.Fatalf("accept = %+v notified=%v, want trusted and notified", accepted, notified)
	}
	if reqs, _ := b.engine.FriendRequests(); len(reqs) != 0 {
		t.Errorf("accepted request should be removed, still have %+v", reqs)
	}
	if !b.trusted(a.DeviceID()) {
		t.Fatal("B should trust A after accepting")
	}
	if !a.trusted(b.DeviceID()) {
		t.Fatal("A should trust B after B's acceptance arrives")
	}
	devB, err := a.store.GetDevice(b.DeviceID())
	if err != nil {
		t.Fatalf("GetDevice on A: %v", err)
	}
	if devB.PendingOutgoing || len(devB.Endpoints) == 0 {
		t.Errorf("B on A = %+v, want settled trust with endpoints", devB)
	}

	if frame := requestFolderIndex(t, ctx, a, b, addrB); frame.Type != protocol.TypeIndexUpdate {
		t.Fatalf("trusted index request answered with %s, want index-update", frame.Type)
	}

	if !a.syncWithAddr(ctx, b.DeviceID(), addrB) {
		t.Fatal("syncWithAddr should succeed between friends")
	}
	got, err := os.ReadFile(filepath.Join(dirA, "doc.txt"))
	if err != nil {
		t.Fatalf("synced file missing: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("synced content = %q, want %q", got, content)
	}
}

func requestFolderList(t *testing.T, ctx context.Context, from *Daemon, to *Daemon, addr string) protocol.Frame {
	t.Helper()
	conn, err := transport.Dial(ctx, from.id, addr, identity.ExpectPeer(to.id.ID()))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()
	s, err := conn.OpenStream(ctx)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer s.Close()
	if err := protocol.WriteFrame(s, protocol.TypeFolderListRequest, nil); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	frame, err := protocol.ReadFrame(s)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	return frame
}

func TestFolderSharingWithFriend(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	a, addrA := newPeer(t, ctx)
	b, _ := newPeer(t, ctx)

	dirA := t.TempDir()
	content := []byte("a picture worth syncing")
	if err := os.WriteFile(filepath.Join(dirA, "pic.txt"), content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := a.store.PutFolder(core.Folder{ID: "photos", Label: "Photos", Path: dirA}); err != nil {
		t.Fatalf("PutFolder: %v", err)
	}

	if frame := requestFolderList(t, ctx, b, a, addrA); frame.Type != protocol.TypeError {
		t.Fatalf("untrusted folder list answered with %s, want error", frame.Type)
	}
	if _, err := b.engine.FriendFolders(ctx, a.DeviceID()); err == nil {
		t.Fatal("FriendFolders should fail for an unknown device")
	}

	if _, delivered, err := b.engine.AddFriendByCode(ctx, a.engine.InviteCode()); err != nil || !delivered {
		t.Fatalf("AddFriendByCode: delivered=%v err=%v", delivered, err)
	}
	if _, _, err := a.engine.AcceptFriendRequest(ctx, b.DeviceID()); err != nil {
		t.Fatalf("AcceptFriendRequest: %v", err)
	}
	if !a.trusted(b.DeviceID()) || !b.trusted(a.DeviceID()) {
		t.Fatal("both sides should trust each other")
	}

	shared, err := b.engine.FriendFolders(ctx, a.DeviceID())
	if err != nil {
		t.Fatalf("FriendFolders: %v", err)
	}
	if len(shared) != 1 || shared[0].ID != "photos" || shared[0].Label != "Photos" {
		t.Fatalf("shared folders = %+v, want [{photos Photos}]", shared)
	}

	dirB := t.TempDir()
	if err := b.store.PutFolder(core.Folder{ID: "photos", Label: "Photos", Path: dirB}); err != nil {
		t.Fatalf("PutFolder on B: %v", err)
	}
	if !b.syncWithAddr(ctx, a.DeviceID(), addrA) {
		t.Fatal("syncWithAddr should succeed between friends")
	}
	got, err := os.ReadFile(filepath.Join(dirB, "pic.txt"))
	if err != nil {
		t.Fatalf("accepted folder did not sync: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("synced content = %q, want %q", got, content)
	}

	c, _ := newPeer(t, ctx)
	if frame := requestFolderList(t, ctx, c, a, addrA); frame.Type != protocol.TypeError {
		t.Fatalf("stranger folder list answered with %s, want error", frame.Type)
	}
	if _, err := c.engine.FriendFolders(ctx, a.DeviceID()); err == nil {
		t.Fatal("a stranger must not be able to list a device's folders")
	}
}

func TestCrossingFriendRequestsResolve(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	a, _ := newPeer(t, ctx)
	b, _ := newPeer(t, ctx)

	if _, delivered, err := b.engine.AddFriendByCode(ctx, a.engine.InviteCode()); err != nil || !delivered {
		t.Fatalf("B's request: delivered=%v err=%v", delivered, err)
	}
	dev, _, err := a.engine.AddFriendByCode(ctx, b.engine.InviteCode())
	if err != nil {
		t.Fatalf("A's crossing request: %v", err)
	}
	if !dev.Trusted {
		t.Fatalf("crossing add should accept the pending request, got %+v", dev)
	}
	if !a.trusted(b.DeviceID()) || !b.trusted(a.DeviceID()) {
		t.Fatal("both sides should trust each other after crossing requests")
	}
	if reqs, _ := a.engine.FriendRequests(); len(reqs) != 0 {
		t.Errorf("A should have no pending requests left, got %+v", reqs)
	}
}

func TestUntrustedPeerCannotAddItselfWithResponse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	a, addrA := newPeer(t, ctx)
	b, _ := newPeer(t, ctx)

	conn, err := transport.Dial(ctx, b.id, addrA, identity.ExpectPeer(a.DeviceID()))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()
	s, err := conn.OpenStream(ctx)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer s.Close()
	msg := protocol.FriendResponse{Accepted: true, Name: "mallory", Endpoints: []string{"203.0.113.66:22067"}}
	if err := protocol.WriteMessage(s, protocol.TypeFriendResponse, msg); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	frame, err := protocol.ReadFrame(s)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if frame.Type != protocol.TypeError {
		t.Fatalf("unsolicited friend response answered with %s, want error", frame.Type)
	}
	if a.trusted(b.DeviceID()) {
		t.Fatal("an unsolicited friend response must never create trust")
	}
}
