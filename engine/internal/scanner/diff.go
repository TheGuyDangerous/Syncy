package scanner

import (
	"sort"

	"github.com/TheGuyDangerous/Syncy/engine/internal/hashing"
)

type ChangeKind int

const (
	Added ChangeKind = iota
	Modified
	Deleted
	Renamed
)

func (k ChangeKind) String() string {
	switch k {
	case Added:
		return "added"
	case Modified:
		return "modified"
	case Deleted:
		return "deleted"
	case Renamed:
		return "renamed"
	default:
		return "unknown"
	}
}

type Change struct {
	Kind    ChangeKind
	Path    string
	OldPath string
	File    FileInfo
}

// Diff returns the changes that turn oldIdx into newIdx, pairing identical
// non-empty content across deletes and adds as renames.
func Diff(oldIdx, newIdx *Index) []Change {
	var changes []Change
	var addedPaths, deletedPaths []string

	for path, nf := range newIdx.Files {
		of, ok := oldIdx.Files[path]
		if !ok {
			addedPaths = append(addedPaths, path)
			continue
		}
		if of.Hash != nf.Hash {
			changes = append(changes, Change{Kind: Modified, Path: path, File: nf})
		}
	}
	for path := range oldIdx.Files {
		if _, ok := newIdx.Files[path]; !ok {
			deletedPaths = append(deletedPaths, path)
		}
	}

	sort.Strings(addedPaths)
	sort.Strings(deletedPaths)

	addedByHash := make(map[hashing.Hash][]string)
	for _, p := range addedPaths {
		f := newIdx.Files[p]
		if f.Size == 0 {
			continue
		}
		addedByHash[f.Hash] = append(addedByHash[f.Hash], p)
	}

	matchedAdded := make(map[string]bool)
	matchedDeleted := make(map[string]bool)
	for _, dp := range deletedPaths {
		df := oldIdx.Files[dp]
		if df.Size == 0 {
			continue
		}
		for _, ap := range addedByHash[df.Hash] {
			if matchedAdded[ap] || newIdx.Files[ap].Size != df.Size {
				continue
			}
			matchedAdded[ap] = true
			matchedDeleted[dp] = true
			changes = append(changes, Change{
				Kind:    Renamed,
				Path:    ap,
				OldPath: dp,
				File:    newIdx.Files[ap],
			})
			break
		}
	}

	for _, p := range addedPaths {
		if !matchedAdded[p] {
			changes = append(changes, Change{Kind: Added, Path: p, File: newIdx.Files[p]})
		}
	}
	for _, p := range deletedPaths {
		if !matchedDeleted[p] {
			changes = append(changes, Change{Kind: Deleted, Path: p, File: oldIdx.Files[p]})
		}
	}

	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Path != changes[j].Path {
			return changes[i].Path < changes[j].Path
		}
		return changes[i].Kind < changes[j].Kind
	})
	return changes
}
