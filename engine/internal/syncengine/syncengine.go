// Package syncengine orchestrates folder synchronization: it manages folders and
// devices and drives convergence with peers over connections.
package syncengine

import (
	"context"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
	"github.com/TheGuyDangerous/Syncy/engine/internal/identity"
	"github.com/TheGuyDangerous/Syncy/engine/internal/metadata"
	"github.com/TheGuyDangerous/Syncy/engine/internal/session"
	"github.com/TheGuyDangerous/Syncy/engine/internal/transport"
)

type Engine struct {
	id    *identity.Identity
	store *metadata.Store
}

func New(id *identity.Identity, store *metadata.Store) *Engine {
	return &Engine{id: id, store: store}
}

func (e *Engine) ID() core.DeviceID { return e.id.ID() }

func (e *Engine) AddFolder(f core.Folder) error { return e.store.PutFolder(f) }

func (e *Engine) RemoveFolder(id string) error { return e.store.RemoveFolder(id) }

func (e *Engine) Folders() ([]core.Folder, error) { return e.store.ListFolders() }

func (e *Engine) GetFolder(id string) (core.Folder, error) { return e.store.GetFolder(id) }

func (e *Engine) AddDevice(d core.Device) error { return e.store.PutDevice(d) }

func (e *Engine) RemoveDevice(id core.DeviceID) error { return e.store.RemoveDevice(id) }

func (e *Engine) Devices() ([]core.Device, error) { return e.store.ListDevices() }

// Converge serves the local folder to the peer and pulls the peer's folder into
// it, so both sides move toward the union of their newest files. The caller must
// keep ctx (and the connection) alive until convergence with the peer completes.
func Converge(ctx context.Context, conn *transport.Conn, folder session.Folder, opts ...session.Option) (session.Stats, error) {
	go func() { _ = session.Serve(ctx, conn, folder) }()
	return session.Pull(ctx, conn, folder.ID, folder.Dir, folder.Index, opts...)
}
