package scanner

import (
	"reflect"
	"testing"

	"github.com/TheGuyDangerous/Syncy/engine/internal/hashing"
)

// fi builds a FileInfo from a path and content (only the fields Diff uses).
func fi(path string, content []byte) FileInfo {
	return FileInfo{
		Path: path,
		Size: int64(len(content)),
		Hash: hashing.OfBytes(content),
	}
}

func index(files ...FileInfo) *Index {
	m := make(map[string]FileInfo, len(files))
	for _, f := range files {
		m[f.Path] = f
	}
	return &Index{Files: m}
}

// kindByPath collapses a change list into a path -> kind map for easy asserts.
func kindByPath(changes []Change) map[string]ChangeKind {
	out := make(map[string]ChangeKind, len(changes))
	for _, c := range changes {
		out[c.Path] = c.Kind
	}
	return out
}

func TestDiffAddedModifiedDeletedUnchanged(t *testing.T) {
	old := index(
		fi("keep.txt", []byte("unchanged")),
		fi("edit.txt", []byte("before")),
		fi("gone.txt", []byte("delete me")),
	)
	newer := index(
		fi("keep.txt", []byte("unchanged")),
		fi("edit.txt", []byte("after")),
		fi("fresh.txt", []byte("brand new")),
	)

	got := kindByPath(Diff(old, newer))
	want := map[string]ChangeKind{
		"edit.txt":  Modified,
		"gone.txt":  Deleted,
		"fresh.txt": Added,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("changes = %v, want %v", got, want)
	}
	if _, ok := got["keep.txt"]; ok {
		t.Error("unchanged file should produce no change")
	}
}

func TestDiffRename(t *testing.T) {
	content := []byte("this content simply moved to a new path")
	old := index(fi("docs/old-name.md", content))
	newer := index(fi("docs/new-name.md", content))

	changes := Diff(old, newer)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %+v", len(changes), changes)
	}
	c := changes[0]
	if c.Kind != Renamed || c.Path != "docs/new-name.md" || c.OldPath != "docs/old-name.md" {
		t.Errorf("unexpected rename change: %+v", c)
	}
}

func TestDiffRenameMultipleIdentical(t *testing.T) {
	content := []byte("duplicated content")
	old := index(fi("a", content), fi("b", content))
	newer := index(fi("a", content), fi("c", content))

	changes := Diff(old, newer)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %+v", len(changes), changes)
	}
	if changes[0].Kind != Renamed || changes[0].Path != "c" || changes[0].OldPath != "b" {
		t.Errorf("unexpected change: %+v", changes[0])
	}
}

func TestDiffEmptyFilesNotRenamed(t *testing.T) {
	old := index(fi("e1", nil))
	newer := index(fi("e2", nil))

	got := kindByPath(Diff(old, newer))
	want := map[string]ChangeKind{"e1": Deleted, "e2": Added}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("empty files should not be treated as a rename: got %v, want %v", got, want)
	}
}

func TestDiffDeterministic(t *testing.T) {
	old := index(
		fi("a", []byte("aaa")),
		fi("b", []byte("bbb")),
		fi("moved", []byte("unique payload")),
	)
	newer := index(
		fi("a", []byte("aaa-edited")),
		fi("c", []byte("ccc")),
		fi("relocated", []byte("unique payload")),
	)

	first := Diff(old, newer)
	second := Diff(old, newer)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("Diff is not deterministic:\n first  = %+v\n second = %+v", first, second)
	}
}

func TestChangeKindString(t *testing.T) {
	cases := map[ChangeKind]string{
		Added:            "added",
		Modified:         "modified",
		Deleted:          "deleted",
		Renamed:          "renamed",
		ChangeKind(9999): "unknown",
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("ChangeKind(%d).String() = %q, want %q", k, got, want)
		}
	}
}
