// Package daemon runs the Syncy engine as a background service, wiring together
// the device identity, metadata store, sync engine, QUIC listener, LAN
// discovery and the local control API.
package daemon

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/api"
	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
	"github.com/TheGuyDangerous/Syncy/engine/internal/discovery"
	"github.com/TheGuyDangerous/Syncy/engine/internal/identity"
	"github.com/TheGuyDangerous/Syncy/engine/internal/metadata"
	"github.com/TheGuyDangerous/Syncy/engine/internal/nat"
	"github.com/TheGuyDangerous/Syncy/engine/internal/protocol"
	"github.com/TheGuyDangerous/Syncy/engine/internal/session"
	"github.com/TheGuyDangerous/Syncy/engine/internal/syncengine"
	"github.com/TheGuyDangerous/Syncy/engine/internal/transport"
)

const (
	remoteDialInterval   = 30 * time.Second
	natCheckInterval     = time.Minute
	natRefreshInterval   = 40 * time.Minute
	untrustedIdleTimeout = 15 * time.Second
)

type Config struct {
	DataDir    string
	ListenAddr string
	APIAddr    string
}

type Daemon struct {
	cfg    Config
	id     *identity.Identity
	store  *metadata.Store
	engine *syncengine.Engine
	token  string

	mu      sync.Mutex
	syncing map[string]bool

	epMu     sync.Mutex
	lanEps   []string
	external string
}

func New(cfg Config) (*Daemon, error) {
	if cfg.DataDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		cfg.DataDir = filepath.Join(base, "syncy")
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":22067"
	}
	if cfg.APIAddr == "" {
		cfg.APIAddr = "127.0.0.1:22068"
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, err
	}

	id, err := identity.LoadOrCreate(filepath.Join(cfg.DataDir, "device.key"))
	if err != nil {
		return nil, err
	}
	store, err := metadata.Open(filepath.Join(cfg.DataDir, "syncy.db"))
	if err != nil {
		return nil, err
	}
	token, err := loadOrCreateToken(filepath.Join(cfg.DataDir, "api-token"))
	if err != nil {
		_ = store.Close()
		return nil, err
	}

	return &Daemon{
		cfg:     cfg,
		id:      id,
		store:   store,
		engine:  syncengine.New(id, store),
		token:   token,
		syncing: make(map[string]bool),
	}, nil
}

func (d *Daemon) Engine() *syncengine.Engine { return d.engine }
func (d *Daemon) DeviceID() core.DeviceID    { return d.id.ID() }
func (d *Daemon) Token() string              { return d.token }
func (d *Daemon) DataDir() string            { return d.cfg.DataDir }
func (d *Daemon) Close() error               { return d.store.Close() }

func (d *Daemon) Run(ctx context.Context) error {
	d.engine.SetEndpointSource(d.endpoints)

	apiLn, err := net.Listen("tcp", d.cfg.APIAddr)
	if err != nil {
		return fmt.Errorf("daemon: control API cannot bind %s: %w", d.cfg.APIAddr, err)
	}
	httpSrv := &http.Server{
		Handler: api.New(d.engine, d.token, filepath.Join(d.cfg.DataDir, "ai.json")),
	}
	go func() { _ = httpSrv.Serve(apiLn) }()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	ln, err := transport.Listen(d.id, d.cfg.ListenAddr, d.authorize)
	if err != nil {
		slog.Warn("peer transport unavailable; syncing is paused until restart",
			"addr", d.cfg.ListenAddr, "error", err)
		<-ctx.Done()
		return nil
	}
	defer ln.Close()

	go d.acceptLoop(ctx, ln)

	settings, err := d.engine.DiscoverySettings()
	if err != nil {
		settings = core.DiscoverySettings{Local: true}
	}
	if port, err := listenPort(ln); err == nil {
		d.setLANEndpoints(port)
		go d.natLoop(ctx, port)
		if settings.Local {
			if announcer, err := discovery.Announce(string(d.id.ID()), port); err == nil {
				defer announcer.Close()
			}
			if peers, err := discovery.Browse(ctx); err == nil {
				go d.dialLoop(ctx, peers)
			}
		}
	}
	go d.remoteDialLoop(ctx)

	<-ctx.Done()
	return nil
}

// authorize admits any peer that proved a device identity during the TLS
// handshake. Trust is enforced per-connection in serveConn: untrusted peers
// may only exchange friend requests and never reach the sync path.
func (d *Daemon) authorize(peerID core.DeviceID, _ *x509.Certificate) error {
	if peerID == "" {
		return errors.New("daemon: peer has no device identity")
	}
	return nil
}

func (d *Daemon) trusted(id core.DeviceID) bool {
	dev, err := d.store.GetDevice(id)
	return err == nil && dev.Trusted
}

func (d *Daemon) acceptLoop(ctx context.Context, ln *transport.Listener) {
	for {
		conn, err := ln.Accept(ctx)
		if err != nil {
			return
		}
		go d.serveConn(ctx, conn)
	}
}

func (d *Daemon) serveConn(ctx context.Context, conn *transport.Conn) {
	defer conn.Close()
	if d.trusted(conn.Peer()) {
		d.serveTrusted(ctx, conn)
		return
	}
	d.serveUntrusted(ctx, conn)
}

func (d *Daemon) serveTrusted(ctx context.Context, conn *transport.Conn) {
	snapshot, err := d.engine.FolderSnapshot()
	if err != nil {
		return
	}
	source := session.Folders(snapshot)
	for {
		s, err := conn.AcceptStream(ctx)
		if err != nil {
			return
		}
		go d.serveTrustedStream(s, conn.Peer(), source)
	}
}

func (d *Daemon) serveTrustedStream(s transport.Stream, peer core.DeviceID, source session.FolderSource) {
	defer s.Close()
	frame, err := protocol.ReadFrame(s)
	if err != nil {
		return
	}
	switch frame.Type {
	case protocol.TypeFriendRequest:
		var req protocol.FriendRequest
		if protocol.Decode(frame, &req) != nil || req.FromID != string(peer) {
			return
		}
		d.engine.RefreshFriend(peer, req.FromName, req.Endpoints)
		_ = protocol.WriteMessage(s, protocol.TypeFriendResponse, protocol.FriendResponse{
			Accepted:  true,
			Name:      d.engine.DeviceName(),
			Endpoints: d.engine.LocalEndpoints(),
		})
	case protocol.TypeFriendResponse:
	case protocol.TypeFolderListRequest:
		folders, err := d.engine.Folders()
		if err != nil {
			return
		}
		shared := make([]protocol.SharedFolder, 0, len(folders))
		for _, f := range folders {
			shared = append(shared, protocol.SharedFolder{ID: f.ID, Label: f.Label})
		}
		_ = protocol.WriteMessage(s, protocol.TypeFolderListResponse, protocol.FolderListResponse{Folders: shared})
	default:
		session.ServeFrame(s, frame, source)
	}
}

// serveUntrusted lets an identity-verified but untrusted peer open exactly one
// stream, which must carry a friend request or a friend response to an ask we
// made. Folder data is never served here.
func (d *Daemon) serveUntrusted(ctx context.Context, conn *transport.Conn) {
	s, err := conn.AcceptStream(ctx)
	if err != nil {
		return
	}
	defer s.Close()
	setStreamDeadline(s, untrustedIdleTimeout)
	frame, err := protocol.ReadFrame(s)
	if err != nil {
		return
	}
	peer := conn.Peer()
	switch frame.Type {
	case protocol.TypeFriendRequest:
		var req protocol.FriendRequest
		if protocol.Decode(frame, &req) != nil || req.FromID != string(peer) {
			d.replyUntrusted(s, protocol.TypeError, protocol.ErrorMsg{
				Code: "bad-request", Message: "friend request does not match the connection identity",
			})
			return
		}
		if err := d.engine.RecordFriendRequest(peer, req.FromName, req.Endpoints); err != nil {
			d.replyUntrusted(s, protocol.TypeError, protocol.ErrorMsg{
				Code: "internal", Message: "could not record the friend request",
			})
			return
		}
		slog.Info("friend request received", "from", peer, "name", req.FromName)
		d.replyUntrusted(s, protocol.TypeAck, protocol.Ack{Marker: "pending"})
	case protocol.TypeFriendResponse:
		var resp protocol.FriendResponse
		if protocol.Decode(frame, &resp) != nil {
			return
		}
		if resp.Accepted && d.engine.ApplyFriendResponse(peer, resp.Name, resp.Endpoints) {
			slog.Info("friend request accepted by peer", "device", peer)
			d.replyUntrusted(s, protocol.TypeAck, protocol.Ack{Marker: "trusted"})
			return
		}
		d.replyUntrusted(s, protocol.TypeError, protocol.ErrorMsg{
			Code: "unexpected", Message: "no outgoing friend request for this device",
		})
	default:
		d.replyUntrusted(s, protocol.TypeError, protocol.ErrorMsg{
			Code: "untrusted", Message: "this device is not trusted; only friend requests are accepted",
		})
	}
}

func (d *Daemon) replyUntrusted(s transport.Stream, typ protocol.MessageType, msg any) {
	if protocol.WriteMessage(s, typ, msg) != nil {
		return
	}
	setStreamDeadline(s, untrustedIdleTimeout)
	_, _ = io.Copy(io.Discard, s)
}

func setStreamDeadline(s transport.Stream, timeout time.Duration) {
	if dl, ok := s.(interface{ SetReadDeadline(time.Time) error }); ok {
		_ = dl.SetReadDeadline(time.Now().Add(timeout))
	}
}

func (d *Daemon) dialLoop(ctx context.Context, peers <-chan discovery.Peer) {
	for {
		select {
		case <-ctx.Done():
			return
		case peer, ok := <-peers:
			if !ok {
				return
			}
			if peer.DeviceID == string(d.id.ID()) || !d.beginSync(peer.DeviceID) {
				continue
			}
			go func(p discovery.Peer) {
				defer d.endSync(p.DeviceID)
				d.syncWithAddr(ctx, core.DeviceID(p.DeviceID), p.Addr)
			}(peer)
		}
	}
}

func (d *Daemon) remoteDialLoop(ctx context.Context) {
	ticker := time.NewTicker(remoteDialInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		settings, err := d.engine.DiscoverySettings()
		if err != nil {
			continue
		}
		devices, err := d.store.ListDevices()
		if err != nil {
			continue
		}
		for _, dev := range devices {
			if dev.ID == d.id.ID() || len(dev.Endpoints) == 0 {
				continue
			}
			if dev.Trusted && !settings.Internet {
				continue
			}
			if !dev.Trusted && !dev.PendingOutgoing {
				continue
			}
			if !d.beginSync(string(dev.ID)) {
				continue
			}
			go func(dev core.Device) {
				defer d.endSync(string(dev.ID))
				if !dev.Trusted {
					d.engine.SendFriendRequest(ctx, dev)
					return
				}
				for _, ep := range dev.Endpoints {
					if d.syncWithAddr(ctx, dev.ID, ep) {
						return
					}
				}
			}(dev)
		}
	}
}

func (d *Daemon) syncWithAddr(ctx context.Context, peer core.DeviceID, addr string) bool {
	dev, err := d.store.GetDevice(peer)
	if err != nil || !dev.Trusted {
		return false
	}
	conn, err := transport.Dial(ctx, d.id, addr, identity.ExpectPeer(peer))
	if err != nil {
		return false
	}
	defer conn.Close()

	folders, err := d.store.ListFolders()
	if err != nil {
		return false
	}
	for _, f := range folders {
		if f.Paused {
			continue
		}
		_, _ = d.engine.PullFolder(ctx, conn, f.ID)
	}
	return true
}

func (d *Daemon) natLoop(ctx context.Context, port int) {
	var lastAttempt time.Time
	for {
		settings, err := d.engine.DiscoverySettings()
		internet := err == nil && settings.Internet
		switch {
		case !internet:
			d.setExternal("")
			lastAttempt = time.Time{}
		case lastAttempt.IsZero() || time.Since(lastAttempt) >= natRefreshInterval:
			d.refreshNAT(ctx, port)
			lastAttempt = time.Now()
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(natCheckInterval):
		}
	}
}

func (d *Daemon) refreshNAT(ctx context.Context, port int) {
	mctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	ep, err := nat.ExternalEndpoint(mctx, port)
	if err != nil {
		slog.Warn("no router port mapping; this device is reachable from the internet only via a public IP or manual port-forward",
			"error", err)
		d.setExternal("")
		return
	}
	slog.Info("mapped external endpoint", "endpoint", ep)
	d.setExternal(ep)
}

func (d *Daemon) setLANEndpoints(port int) {
	ips, err := nat.LANIPs()
	if err != nil {
		return
	}
	eps := make([]string, 0, len(ips))
	for _, ip := range ips {
		eps = append(eps, nat.Endpoint(ip, port))
	}
	d.epMu.Lock()
	d.lanEps = eps
	d.epMu.Unlock()
}

func (d *Daemon) setExternal(ep string) {
	d.epMu.Lock()
	d.external = ep
	d.epMu.Unlock()
}

func (d *Daemon) endpoints() []string {
	d.epMu.Lock()
	defer d.epMu.Unlock()
	out := make([]string, 0, len(d.lanEps)+1)
	out = append(out, d.lanEps...)
	if d.external != "" {
		out = append(out, d.external)
	}
	return out
}

func (d *Daemon) beginSync(deviceID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.syncing[deviceID] {
		return false
	}
	d.syncing[deviceID] = true
	return true
}

func (d *Daemon) endSync(deviceID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.syncing, deviceID)
}

func listenPort(ln *transport.Listener) (int, error) {
	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(portStr)
}

func loadOrCreateToken(path string) (string, error) {
	if data, err := os.ReadFile(path); err == nil {
		return strings.TrimSpace(string(data)), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	token, err := api.GenerateToken()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		return "", err
	}
	return token, nil
}
