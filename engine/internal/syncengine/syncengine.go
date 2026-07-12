// Package syncengine orchestrates folder synchronization: it manages folders and
// devices and drives convergence with peers over connections.
package syncengine

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
	"github.com/TheGuyDangerous/Syncy/engine/internal/hashing"
	"github.com/TheGuyDangerous/Syncy/engine/internal/identity"
	"github.com/TheGuyDangerous/Syncy/engine/internal/metadata"
	"github.com/TheGuyDangerous/Syncy/engine/internal/scanner"
	"github.com/TheGuyDangerous/Syncy/engine/internal/session"
	"github.com/TheGuyDangerous/Syncy/engine/internal/transport"
	"github.com/TheGuyDangerous/Syncy/engine/internal/versioning"
)

const (
	versionsDir = ".syncy-versions"
	maxVersions = 10
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
	go func() { _ = session.Serve(ctx, conn, session.SingleFolder(folder)) }()
	return session.Pull(ctx, conn, folder.ID, folder.Dir, folder.Index, opts...)
}

// Sync converges a stored folder with a peer using the persisted last-synced
// baseline for conflict detection and version history, then records the new
// baseline for the next round.
func (e *Engine) Sync(ctx context.Context, conn *transport.Conn, folderID string) (session.Stats, error) {
	folder, err := e.store.GetFolder(folderID)
	if err != nil {
		return session.Stats{}, err
	}
	idx, err := e.scan(folder.Path)
	if err != nil {
		return session.Stats{}, err
	}
	baseline, err := e.store.GetSyncedBaseline(folderID)
	if err != nil {
		return session.Stats{}, err
	}

	versions := versioning.New(filepath.Join(folder.Path, versionsDir), maxVersions)
	stats, err := Converge(ctx, conn, session.Folder{ID: folderID, Dir: folder.Path, Index: idx},
		session.WithBaseline(baseline),
		session.WithConflictNaming(e.ID()),
		session.WithVersioning(versions),
	)
	if err != nil {
		return stats, err
	}

	newIdx, err := e.scan(folder.Path)
	if err != nil {
		return stats, err
	}
	if err := e.store.SetSyncedBaseline(folderID, indexHashes(newIdx)); err != nil {
		return stats, err
	}
	return stats, nil
}

func (e *Engine) scan(dir string) (*scanner.Index, error) {
	sc, err := scanner.New(nil, scanner.WithSkip(func(rel string, _ fs.DirEntry) bool {
		return rel == versionsDir || strings.HasPrefix(rel, versionsDir+"/") || strings.HasSuffix(rel, ".syncy-tmp")
	}))
	if err != nil {
		return nil, err
	}
	return sc.Scan(dir)
}

func indexHashes(idx *scanner.Index) map[string]hashing.Hash {
	out := make(map[string]hashing.Hash, len(idx.Files))
	for path, fi := range idx.Files {
		out[path] = fi.Hash
	}
	return out
}

type Conflict struct {
	FolderID string `json:"folder_id"`
	Path     string `json:"path"`
}

func (e *Engine) Conflicts() ([]Conflict, error) {
	folders, err := e.store.ListFolders()
	if err != nil {
		return nil, err
	}
	var out []Conflict
	for _, f := range folders {
		err := filepath.WalkDir(f.Path, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if strings.Contains(d.Name(), ".sync-conflict-") {
				rel, err := filepath.Rel(f.Path, path)
				if err != nil {
					return err
				}
				out = append(out, Conflict{FolderID: f.ID, Path: filepath.ToSlash(rel)})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (e *Engine) FolderVersions(folderID, relPath string) ([]versioning.Version, error) {
	folder, err := e.store.GetFolder(folderID)
	if err != nil {
		return nil, err
	}
	store := versioning.New(filepath.Join(folder.Path, versionsDir), maxVersions)
	return store.Versions(relPath)
}
