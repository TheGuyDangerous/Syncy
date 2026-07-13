package syncengine

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
	"github.com/TheGuyDangerous/Syncy/engine/internal/identity"
	"github.com/TheGuyDangerous/Syncy/engine/internal/invite"
	"github.com/TheGuyDangerous/Syncy/engine/internal/metadata"
	"github.com/TheGuyDangerous/Syncy/engine/internal/protocol"
	"github.com/TheGuyDangerous/Syncy/engine/internal/session"
	"github.com/TheGuyDangerous/Syncy/engine/internal/transport"
)

var (
	ErrOwnCode     = errors.New("syncengine: that is this device's own invite code")
	ErrNotFriend   = errors.New("syncengine: that device is not a trusted friend")
	ErrUnreachable = errors.New("syncengine: friend is offline or unreachable")
)

const (
	friendDialTimeout = 6 * time.Second
	maxNameLen        = 120
	maxEndpoints      = 16
	maxEndpointLen    = 128
	discoveryKey      = "discovery"
)

func (e *Engine) SetEndpointSource(fn func() []string) {
	e.epMu.Lock()
	defer e.epMu.Unlock()
	e.eps = fn
}

func (e *Engine) LocalEndpoints() []string {
	e.epMu.RLock()
	fn := e.eps
	e.epMu.RUnlock()
	if fn == nil {
		return nil
	}
	return fn()
}

func (e *Engine) DeviceName() string {
	name, err := os.Hostname()
	if err != nil {
		return ""
	}
	return name
}

func (e *Engine) InviteCode() string {
	return invite.Encode(invite.Code{
		DeviceID:  string(e.ID()),
		Name:      e.DeviceName(),
		Endpoints: e.LocalEndpoints(),
	})
}

// AddFriendByCode records the device behind an invite code and asks it to
// become a friend. The returned flag reports whether the request (or, for a
// crossing request, our acceptance) reached the peer.
func (e *Engine) AddFriendByCode(ctx context.Context, token string) (core.Device, bool, error) {
	code, err := invite.Decode(token)
	if err != nil {
		return core.Device{}, false, err
	}
	if code.DeviceID == string(e.ID()) {
		return core.Device{}, false, ErrOwnCode
	}
	id := core.DeviceID(code.DeviceID)

	if existing, err := e.store.GetDevice(id); err == nil && existing.Trusted {
		return existing, false, nil
	}
	if _, err := e.store.GetFriendRequest(id); err == nil {
		return e.AcceptFriendRequest(ctx, id)
	}

	dev := core.Device{
		ID:              id,
		Name:            clampName(code.Name),
		Endpoints:       clampEndpoints(code.Endpoints),
		PendingOutgoing: true,
		AddedAt:         time.Now(),
	}
	if err := e.store.PutDevice(dev); err != nil {
		return core.Device{}, false, err
	}
	delivered := e.SendFriendRequest(ctx, dev)
	dev, err = e.store.GetDevice(id)
	if err != nil {
		return core.Device{}, delivered, err
	}
	return dev, delivered, nil
}

// SendFriendRequest dials the device's known endpoints and delivers a friend
// request; when the peer already trusts us it answers with an acceptance,
// which is applied immediately.
func (e *Engine) SendFriendRequest(ctx context.Context, dev core.Device) bool {
	req := protocol.FriendRequest{
		FromID:    string(e.ID()),
		FromName:  e.DeviceName(),
		Endpoints: e.LocalEndpoints(),
	}
	for _, ep := range dev.Endpoints {
		frame := e.sendFriendFrame(ctx, dev.ID, ep, protocol.TypeFriendRequest, req)
		if frame == nil {
			continue
		}
		switch frame.Type {
		case protocol.TypeAck:
			return true
		case protocol.TypeFriendResponse:
			var resp protocol.FriendResponse
			if protocol.Decode(*frame, &resp) == nil && resp.Accepted {
				e.ApplyFriendResponse(dev.ID, resp.Name, resp.Endpoints)
			}
			return true
		}
	}
	return false
}

func (e *Engine) FriendRequests() ([]core.FriendRequest, error) {
	return e.store.ListFriendRequests()
}

// FriendFolders dials a trusted friend at its saved endpoints and returns the
// folders it shares.
func (e *Engine) FriendFolders(ctx context.Context, id core.DeviceID) ([]protocol.SharedFolder, error) {
	dev, err := e.store.GetDevice(id)
	if err != nil {
		return nil, err
	}
	if !dev.Trusted {
		return nil, ErrNotFriend
	}
	for _, ep := range dev.Endpoints {
		if ctx.Err() != nil {
			break
		}
		dctx, cancel := context.WithTimeout(ctx, friendDialTimeout)
		conn, err := transport.Dial(dctx, e.id, ep, identity.ExpectPeer(id))
		if err != nil {
			cancel()
			continue
		}
		folders, err := session.RequestFolderList(dctx, conn)
		conn.Close()
		cancel()
		if err != nil {
			continue
		}
		if folders == nil {
			folders = []protocol.SharedFolder{}
		}
		return folders, nil
	}
	return nil, ErrUnreachable
}

// AcceptFriendRequest trusts the requesting device and, when reachable, tells
// it so. The returned flag reports whether the peer was notified.
func (e *Engine) AcceptFriendRequest(ctx context.Context, from core.DeviceID) (core.Device, bool, error) {
	fr, err := e.store.GetFriendRequest(from)
	if err != nil {
		return core.Device{}, false, err
	}
	dev := core.Device{
		ID:        from,
		Name:      fr.Name,
		Endpoints: fr.Endpoints,
		Trusted:   true,
		LastSeen:  time.Now(),
		AddedAt:   time.Now(),
	}
	if err := e.store.PutDevice(dev); err != nil {
		return core.Device{}, false, err
	}
	if err := e.store.RemoveFriendRequest(from); err != nil {
		return core.Device{}, false, err
	}

	resp := protocol.FriendResponse{
		Accepted:  true,
		Name:      e.DeviceName(),
		Endpoints: e.LocalEndpoints(),
	}
	notified := false
	for _, ep := range dev.Endpoints {
		if frame := e.sendFriendFrame(ctx, from, ep, protocol.TypeFriendResponse, resp); frame != nil {
			notified = frame.Type == protocol.TypeAck
			break
		}
	}
	return dev, notified, nil
}

func (e *Engine) RejectFriendRequest(from core.DeviceID) error {
	return e.store.RemoveFriendRequest(from)
}

// RecordFriendRequest stores an incoming request from an identity-verified but
// untrusted peer, deduplicated by device.
func (e *Engine) RecordFriendRequest(from core.DeviceID, name string, endpoints []string) error {
	return e.store.PutFriendRequest(core.FriendRequest{
		FromID:    from,
		Name:      clampName(name),
		Endpoints: clampEndpoints(endpoints),
		CreatedAt: time.Now(),
	})
}

// ApplyFriendResponse marks a device we previously asked to befriend as
// trusted. Responses from devices we never asked are ignored.
func (e *Engine) ApplyFriendResponse(from core.DeviceID, name string, endpoints []string) bool {
	dev, err := e.store.GetDevice(from)
	if err != nil {
		return false
	}
	if dev.Trusted {
		return true
	}
	if !dev.PendingOutgoing {
		return false
	}
	dev.Trusted, dev.PendingOutgoing = true, false
	if name != "" {
		dev.Name = clampName(name)
	}
	if eps := clampEndpoints(endpoints); len(eps) > 0 {
		dev.Endpoints = eps
	}
	dev.LastSeen = time.Now()
	return e.store.PutDevice(dev) == nil
}

// RefreshFriend updates a trusted device's name and endpoints from a friend
// message it sent over an authenticated connection.
func (e *Engine) RefreshFriend(from core.DeviceID, name string, endpoints []string) {
	dev, err := e.store.GetDevice(from)
	if err != nil || !dev.Trusted {
		return
	}
	if name != "" {
		dev.Name = clampName(name)
	}
	if eps := clampEndpoints(endpoints); len(eps) > 0 {
		dev.Endpoints = eps
	}
	dev.LastSeen = time.Now()
	_ = e.store.PutDevice(dev)
}

func (e *Engine) DiscoverySettings() (core.DiscoverySettings, error) {
	raw, err := e.store.GetSetting(discoveryKey)
	if errors.Is(err, metadata.ErrNotFound) {
		return core.DiscoverySettings{Local: true}, nil
	}
	if err != nil {
		return core.DiscoverySettings{}, err
	}
	var s core.DiscoverySettings
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return core.DiscoverySettings{Local: true}, nil
	}
	return s, nil
}

func (e *Engine) SetDiscoverySettings(s core.DiscoverySettings) error {
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return e.store.SetSetting(discoveryKey, string(raw))
}

func (e *Engine) sendFriendFrame(ctx context.Context, peer core.DeviceID, addr string, typ protocol.MessageType, msg any) *protocol.Frame {
	dctx, cancel := context.WithTimeout(ctx, friendDialTimeout)
	defer cancel()
	conn, err := transport.Dial(dctx, e.id, addr, identity.ExpectPeer(peer))
	if err != nil {
		return nil
	}
	defer conn.Close()
	s, err := conn.OpenStream(dctx)
	if err != nil {
		return nil
	}
	defer s.Close()
	if err := protocol.WriteMessage(s, typ, msg); err != nil {
		return nil
	}
	if dl, ok := s.(interface{ SetReadDeadline(time.Time) error }); ok {
		_ = dl.SetReadDeadline(time.Now().Add(friendDialTimeout))
	}
	frame, err := protocol.ReadFrame(s)
	if err != nil {
		return nil
	}
	return &frame
}

func clampName(s string) string {
	if len(s) > maxNameLen {
		return s[:maxNameLen]
	}
	return s
}

func clampEndpoints(eps []string) []string {
	var out []string
	for _, ep := range eps {
		if len(out) == maxEndpoints {
			break
		}
		if ep == "" || len(ep) > maxEndpointLen {
			continue
		}
		if _, _, err := net.SplitHostPort(ep); err != nil {
			continue
		}
		out = append(out, ep)
	}
	return out
}
