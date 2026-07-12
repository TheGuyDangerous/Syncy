// Package scanner walks a folder and produces a content-addressed [Index] of
// its files: each file's size, modification time, mode, content-defined blocks
// and whole-file hash. The index is the input the sync engine reconciles
// against a peer's index to decide what to transfer.
//
// Paths in the index are always relative to the folder root and use forward
// slashes, so an index means the same thing on every operating system.
package scanner

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/chunker"
	"github.com/TheGuyDangerous/Syncy/engine/internal/hashing"
)

// FileInfo describes a single regular file within a scanned folder.
type FileInfo struct {
	Path    string          // relative, slash-separated path from the folder root
	Size    int64           // size in bytes
	ModTime time.Time       // last modification time (UTC)
	Mode    fs.FileMode     // permission bits
	Hash    hashing.Hash    // SHA-256 of the whole file
	Blocks  []chunker.Chunk // content-defined blocks, in order
}

// Index is a snapshot of a folder's files keyed by relative path.
type Index struct {
	// Files maps each relative path to its info. Directories and non-regular
	// files (symlinks, devices, sockets) are not included.
	Files map[string]FileInfo
}

// Scanner builds an [Index] from a folder using a configured chunker.
type Scanner struct {
	chunker *chunker.Chunker
	// Skip, if set, is called for each entry; returning true omits the entry
	// (and, for a directory, everything beneath it) from the scan.
	skip func(relPath string, d fs.DirEntry) bool
}

// Option configures a Scanner.
type Option func(*Scanner)

// WithSkip sets a predicate used to ignore entries during a scan.
func WithSkip(skip func(relPath string, d fs.DirEntry) bool) Option {
	return func(s *Scanner) { s.skip = skip }
}

// New returns a Scanner. If ch is nil, a chunker with the default config is
// used.
func New(ch *chunker.Chunker, opts ...Option) (*Scanner, error) {
	if ch == nil {
		var err error
		if ch, err = chunker.New(chunker.DefaultConfig()); err != nil {
			return nil, err
		}
	}
	s := &Scanner{chunker: ch}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// Scan walks root and returns an Index of its regular files.
func (s *Scanner) Scan(root string) (*Index, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("scanner: root is not a directory")
	}

	idx := &Index{Files: make(map[string]FileInfo)}
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := relPath(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil // the root itself
		}
		if s.skip != nil && s.skip(rel, d) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		// Only regular files are indexed; skip directories, symlinks, etc.
		if !d.Type().IsRegular() {
			return nil
		}
		fi, err := s.scanFile(path, rel)
		if err != nil {
			return err
		}
		idx.Files[rel] = fi
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return idx, nil
}

func (s *Scanner) scanFile(absPath, rel string) (FileInfo, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return FileInfo{}, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return FileInfo{}, err
	}

	// Hash the whole file in the same pass used to chunk it.
	hasher := hashing.NewHasher()
	tee := io.TeeReader(f, hasher)

	var blocks []chunker.Chunk
	if err := s.chunker.Split(tee, func(c chunker.Chunk) error {
		blocks = append(blocks, c)
		return nil
	}); err != nil {
		return FileInfo{}, err
	}

	return FileInfo{
		Path:    rel,
		Size:    stat.Size(),
		ModTime: stat.ModTime().UTC(),
		Mode:    stat.Mode().Perm(),
		Hash:    hasher.Sum(),
		Blocks:  blocks,
	}, nil
}

// relPath returns the slash-separated path of target relative to root.
func relPath(root, target string) (string, error) {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}
