// Package versioning keeps recoverable copies of files before they are
// overwritten or deleted, and restores them on request.
package versioning

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const stampLayout = "20060102-150405"

type Store struct {
	dir  string
	keep int
}

type Version struct {
	Stamp   string
	Path    string
	ModTime time.Time
	Size    int64
}

func New(dir string, keep int) *Store {
	return &Store{dir: dir, keep: keep}
}

func (s *Store) Archive(root, relPath string) error {
	src := filepath.Join(root, filepath.FromSlash(relPath))
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("versioning: cannot archive a directory")
	}

	stamp := info.ModTime().UTC().Format(stampLayout)
	dst := s.uniquePath(relPath, stamp)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := copyFile(src, dst); err != nil {
		return err
	}
	_ = os.Chtimes(dst, info.ModTime(), info.ModTime())
	return s.prune(relPath)
}

func (s *Store) Versions(relPath string) ([]Version, error) {
	rel := filepath.FromSlash(relPath)
	dir := filepath.Join(s.dir, filepath.Dir(rel))
	prefix, ext := splitName(filepath.Base(rel))

	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var versions []Version
	tagPrefix := prefix + "~"
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, tagPrefix) || !strings.HasSuffix(name, ext) {
			continue
		}
		stamp := name[len(tagPrefix) : len(name)-len(ext)]
		info, err := e.Info()
		if err != nil {
			return nil, err
		}
		versions = append(versions, Version{
			Stamp:   stamp,
			Path:    filepath.Join(dir, name),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i].Stamp > versions[j].Stamp })
	return versions, nil
}

func (s *Store) Restore(root, relPath, stamp string) error {
	rel := filepath.FromSlash(relPath)
	prefix, ext := splitName(filepath.Base(rel))
	archived := filepath.Join(s.dir, filepath.Dir(rel), prefix+"~"+stamp+ext)
	if _, err := os.Stat(archived); err != nil {
		return err
	}
	dst := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return copyFile(archived, dst)
}

func (s *Store) prune(relPath string) error {
	if s.keep <= 0 {
		return nil
	}
	versions, err := s.Versions(relPath)
	if err != nil {
		return err
	}
	for _, v := range versions[min(s.keep, len(versions)):] {
		if err := os.Remove(v.Path); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) uniquePath(relPath, stamp string) string {
	rel := filepath.FromSlash(relPath)
	dir := filepath.Join(s.dir, filepath.Dir(rel))
	prefix, ext := splitName(filepath.Base(rel))
	for i := 0; ; i++ {
		tag := stamp
		if i > 0 {
			tag = stamp + "-" + strconv.Itoa(i)
		}
		candidate := filepath.Join(dir, prefix+"~"+tag+ext)
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
}

func splitName(base string) (prefix, ext string) {
	ext = filepath.Ext(base)
	return base[:len(base)-len(ext)], ext
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
