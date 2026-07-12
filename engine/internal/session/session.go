// Package session runs the DeltaSync Protocol over a transport connection to
// reconcile a folder: it serves a local folder's index and blocks to peers and
// pulls missing files from them, reusing blocks it already has.
package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/TheGuyDangerous/Syncy/engine/internal/hashing"
	"github.com/TheGuyDangerous/Syncy/engine/internal/protocol"
	"github.com/TheGuyDangerous/Syncy/engine/internal/scanner"
	"github.com/TheGuyDangerous/Syncy/engine/internal/transport"
	"github.com/TheGuyDangerous/Syncy/engine/internal/versioning"
)

const tmpSuffix = ".syncy-tmp"

type Folder struct {
	ID    string
	Dir   string
	Index *scanner.Index
}

type config struct {
	versions *versioning.Store
}

type Option func(*config)

func WithVersioning(store *versioning.Store) Option {
	return func(c *config) { c.versions = store }
}

type Stats struct {
	FilesUpdated  int
	BlocksFetched int
	BlocksReused  int
}

func Serve(ctx context.Context, conn *transport.Conn, folder Folder) error {
	for {
		s, err := conn.AcceptStream(ctx)
		if err != nil {
			return err
		}
		go serveStream(s, folder)
	}
}

func serveStream(s transport.Stream, folder Folder) {
	defer s.Close()
	frame, err := protocol.ReadFrame(s)
	if err != nil {
		return
	}
	switch frame.Type {
	case protocol.TypeFolderSummary:
		_ = protocol.WriteMessage(s, protocol.TypeIndexUpdate, protocol.IndexUpdate{
			FolderID: folder.ID,
			Files:    indexToFileMeta(folder.Index),
			Final:    true,
		})
	case protocol.TypeBlockRequest:
		var req protocol.BlockRequest
		if err := protocol.Decode(frame, &req); err != nil {
			return
		}
		serveBlocks(s, folder.Dir, req)
	}
}

func serveBlocks(s transport.Stream, dir string, req protocol.BlockRequest) {
	f, err := os.Open(filepath.Join(dir, filepath.FromSlash(req.Path)))
	if err != nil {
		return
	}
	defer f.Close()
	for _, ref := range req.Blocks {
		buf := make([]byte, ref.Length)
		if _, err := f.ReadAt(buf, ref.Offset); err != nil {
			return
		}
		if err := protocol.WriteBlockData(s, ref.Hash, buf); err != nil {
			return
		}
	}
}

func Pull(ctx context.Context, conn *transport.Conn, folderID, destDir string, local *scanner.Index, opts ...Option) (Stats, error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	var stats Stats
	remote, err := requestIndex(ctx, conn, folderID)
	if err != nil {
		return stats, err
	}
	localBlocks := buildBlockIndex(local)
	for _, rf := range remote {
		if rf.Deleted {
			continue
		}
		if lf, ok := local.Files[rf.Path]; ok && lf.Hash == rf.Hash {
			continue
		}
		if err := pullFile(ctx, conn, folderID, destDir, rf, localBlocks, &cfg, &stats); err != nil {
			return stats, err
		}
		stats.FilesUpdated++
	}
	return stats, nil
}

type blockLoc struct {
	path   string
	offset int64
	length int
}

func buildBlockIndex(idx *scanner.Index) map[hashing.Hash]blockLoc {
	m := make(map[hashing.Hash]blockLoc)
	for _, fi := range idx.Files {
		for _, b := range fi.Blocks {
			if _, ok := m[b.Hash]; !ok {
				m[b.Hash] = blockLoc{path: fi.Path, offset: b.Offset, length: b.Length}
			}
		}
	}
	return m
}

func requestIndex(ctx context.Context, conn *transport.Conn, folderID string) ([]protocol.FileMeta, error) {
	s, err := conn.OpenStream(ctx)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	if err := protocol.WriteMessage(s, protocol.TypeFolderSummary, protocol.FolderSummary{FolderID: folderID}); err != nil {
		return nil, err
	}

	var files []protocol.FileMeta
	for {
		frame, err := protocol.ReadFrame(s)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if frame.Type != protocol.TypeIndexUpdate {
			return nil, fmt.Errorf("session: unexpected %s while reading index", frame.Type)
		}
		var iu protocol.IndexUpdate
		if err := protocol.Decode(frame, &iu); err != nil {
			return nil, err
		}
		files = append(files, iu.Files...)
		if iu.Final {
			break
		}
	}
	return files, nil
}

func requestBlocks(ctx context.Context, conn *transport.Conn, folderID, path string, refs []protocol.BlockRef) (map[hashing.Hash][]byte, error) {
	s, err := conn.OpenStream(ctx)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	if err := protocol.WriteMessage(s, protocol.TypeBlockRequest, protocol.BlockRequest{
		FolderID: folderID,
		Path:     path,
		Blocks:   refs,
	}); err != nil {
		return nil, err
	}

	out := make(map[hashing.Hash][]byte, len(refs))
	for {
		frame, err := protocol.ReadFrame(s)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if frame.Type != protocol.TypeBlockData {
			return nil, fmt.Errorf("session: unexpected %s while reading blocks", frame.Type)
		}
		h, data, err := protocol.ParseBlockData(frame.Payload)
		if err != nil {
			return nil, err
		}
		out[h] = data
		if len(out) == len(refs) {
			break
		}
	}
	return out, nil
}

func pullFile(ctx context.Context, conn *transport.Conn, folderID, destDir string, rf protocol.FileMeta, localBlocks map[hashing.Hash]blockLoc, cfg *config, stats *Stats) error {
	var need []protocol.BlockRef
	for _, b := range rf.Blocks {
		if _, ok := localBlocks[b.Hash]; !ok {
			need = append(need, protocol.BlockRef{Offset: b.Offset, Length: b.Length, Hash: b.Hash})
		}
	}

	fetched := map[hashing.Hash][]byte{}
	if len(need) > 0 {
		var err error
		fetched, err = requestBlocks(ctx, conn, folderID, rf.Path, need)
		if err != nil {
			return err
		}
	}

	destPath := filepath.Join(destDir, filepath.FromSlash(rf.Path))
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	tmpPath := destPath + tmpSuffix
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	hasher := hashing.NewHasher()
	writer := io.MultiWriter(out, hasher)

	fail := func(err error) error {
		out.Close()
		os.Remove(tmpPath)
		return err
	}

	for _, b := range rf.Blocks {
		var data []byte
		if loc, ok := localBlocks[b.Hash]; ok {
			data, err = readLocalBlock(destDir, loc)
			if err != nil {
				return fail(err)
			}
			stats.BlocksReused++
		} else {
			data = fetched[b.Hash]
			if data == nil {
				return fail(fmt.Errorf("session: peer did not deliver block %s", b.Hash.Short()))
			}
			stats.BlocksFetched++
		}
		if hashing.OfBytes(data) != b.Hash {
			return fail(fmt.Errorf("session: block hash mismatch in %s", rf.Path))
		}
		if _, err := writer.Write(data); err != nil {
			return fail(err)
		}
	}

	if err := out.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if hasher.Sum() != rf.Hash {
		os.Remove(tmpPath)
		return fmt.Errorf("session: file hash mismatch for %s", rf.Path)
	}

	if cfg.versions != nil {
		if _, err := os.Stat(destPath); err == nil {
			if err := cfg.versions.Archive(destDir, rf.Path); err != nil {
				os.Remove(tmpPath)
				return err
			}
		}
	}
	return os.Rename(tmpPath, destPath)
}

func readLocalBlock(destDir string, loc blockLoc) ([]byte, error) {
	f, err := os.Open(filepath.Join(destDir, filepath.FromSlash(loc.path)))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, loc.length)
	if _, err := f.ReadAt(buf, loc.offset); err != nil {
		return nil, err
	}
	return buf, nil
}

func indexToFileMeta(idx *scanner.Index) []protocol.FileMeta {
	files := make([]protocol.FileMeta, 0, len(idx.Files))
	for _, fi := range idx.Files {
		files = append(files, protocol.FileMeta{
			Path:    fi.Path,
			Size:    fi.Size,
			ModUnix: fi.ModTime.Unix(),
			Mode:    uint32(fi.Mode),
			Hash:    fi.Hash,
			Blocks:  fi.Blocks,
		})
	}
	return files
}
