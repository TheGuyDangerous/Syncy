// Package fsafe guards filesystem paths built from untrusted input, rejecting
// anything that would escape its root directory.
package fsafe

import (
	"errors"
	"path/filepath"
)

var ErrUnsafePath = errors.New("fsafe: path escapes its root")

// Local reports whether rel is a safe, relative, in-root path: not absolute and
// with no ".." components that would climb out of a base directory.
func Local(rel string) bool {
	return filepath.IsLocal(filepath.FromSlash(rel))
}

// Join joins base with a slash-separated relative path, returning ErrUnsafePath
// if rel is absolute or escapes base.
func Join(base, rel string) (string, error) {
	clean := filepath.FromSlash(rel)
	if !filepath.IsLocal(clean) {
		return "", ErrUnsafePath
	}
	return filepath.Join(base, clean), nil
}
