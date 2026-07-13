// Package daemon runs the Syncy engine as a background service, wiring together
// the device identity, metadata store, sync engine, QUIC listener, LAN
// discovery and the local control API.
package daemon

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
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
	"github.com/TheGuyDangerous/Syncy/engine/internal/session"
	"github.com/TheGuyDangerous/Syncy/engine/internal/syncengine"
	"github.com/TheGuyDangerous/Syncy/engine/internal/transport"
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

	if port, err := listenPort(ln); err == nil {
		if announcer, err := discovery.Announce(string(d.id.ID()), port); err == nil {
			defer announcer.Close()
		}
	}
	if peers, err := discovery.Browse(ctx); err == nil {
		go d.dialLoop(ctx, peers)
	}

	<-ctx.Done()
	return nil
}

func (d *Daemon) authorize(peerID core.DeviceID, _ *x509.Certificate) error {
	dev, err := d.store.GetDevice(peerID)
	if err != nil {
		return errors.New("daemon: unknown device")
	}
	if !dev.Trusted {
		return errors.New("daemon: device not trusted")
	}
	return nil
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
	snapshot, err := d.engine.FolderSnapshot()
	if err != nil {
		return
	}
	_ = session.Serve(ctx, conn, session.Folders(snapshot))
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
				d.syncWithPeer(ctx, p)
			}(peer)
		}
	}
}

func (d *Daemon) syncWithPeer(ctx context.Context, peer discovery.Peer) {
	dev, err := d.store.GetDevice(core.DeviceID(peer.DeviceID))
	if err != nil || !dev.Trusted {
		return
	}
	conn, err := transport.Dial(ctx, d.id, peer.Addr, d.authorize)
	if err != nil {
		return
	}
	defer conn.Close()

	folders, err := d.store.ListFolders()
	if err != nil {
		return
	}
	for _, f := range folders {
		if f.Paused {
			continue
		}
		_, _ = d.engine.PullFolder(ctx, conn, f.ID)
	}
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
