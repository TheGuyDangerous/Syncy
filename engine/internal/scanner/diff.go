package scanner

import (
	"sort"

	"github.com/TheGuyDangerous/Syncy/engine/internal/hashing"
)

// ChangeKind classifies one difference between two indexes.
type ChangeKind int

const (
	// Added means the file exists in the new index but not the old one.
	Added ChangeKind = iota
	// Modified means the file exists in both but its content changed.
	Modified
	// Deleted means the file existed in the old index but not the new one.
	Deleted
	// Renamed means identical content moved from OldPath to Path.
	Renamed
)

// String returns the lowercase name of the change kind.
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

// Change describes a single difference that transforms an old index into a new
// one.
type Change struct {
	Kind ChangeKind
	// Path is the current path: the new path for Added/Modified/Renamed, and
	// the removed path for Deleted.
	Path string
	// OldPath is only set for Renamed: the file's previous path.
	OldPath string
	// File is the relevant FileInfo: the new info for Added/Modified/Renamed,
	// and the old info for Deleted.
	File FileInfo
}

// Diff computes the set of changes that turn oldIdx into newIdx. Renames and
// moves are detected by matching deleted and added files with identical
// non-empty content, so a moved file transfers no data. The result is sorted by
// path for deterministic output.
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

	// Sort so both rename matching and final output are deterministic.
	sort.Strings(addedPaths)
	sort.Strings(deletedPaths)

	// Index the additions by content hash for rename matching. Empty files are
	// excluded: many unrelated empty files share one hash and would pair
	// spuriously.
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
