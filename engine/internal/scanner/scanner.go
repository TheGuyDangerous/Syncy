// Package scanner walks a folder into a content-addressed index of its files.
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

type FileInfo struct {
	Path    string
	Size    int64
	ModTime time.Time
	Mode    fs.FileMode
	Hash    hashing.Hash
	Blocks  []chunker.Chunk
}

type Index struct {
	Files map[string]FileInfo
}

type Scanner struct {
	chunker *chunker.Chunker
	skip    func(relPath string, d fs.DirEntry) bool
}

type Option func(*Scanner)

func WithSkip(skip func(relPath string, d fs.DirEntry) bool) Option {
	return func(s *Scanner) { s.skip = skip }
}

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
			return nil
		}
		if s.skip != nil && s.skip(rel, d) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
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

func relPath(root, target string) (string, error) {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}
