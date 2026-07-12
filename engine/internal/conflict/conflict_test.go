package conflict

import (
	"testing"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
)

const (
	devA core.DeviceID = "device-a"
	devB core.DeviceID = "device-b"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		name string
		a, b Vector
		want Ordering
	}{
		{"both empty", Vector{}, Vector{}, Equal},
		{"identical", Vector{devA: 2, devB: 1}, Vector{devA: 2, devB: 1}, Equal},
		{"a ahead", Vector{devA: 3, devB: 1}, Vector{devA: 2, devB: 1}, Greater},
		{"b ahead", Vector{devA: 2}, Vector{devA: 2, devB: 1}, Less},
		{"a superset", Vector{devA: 1, devB: 1}, Vector{devA: 1}, Greater},
		{"concurrent", Vector{devA: 2, devB: 1}, Vector{devA: 1, devB: 2}, Concurrent},
		{"empty vs nonempty", Vector{}, Vector{devA: 1}, Less},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.a.Compare(tc.b); got != tc.want {
				t.Errorf("Compare = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCompareIsAntisymmetric(t *testing.T) {
	a := Vector{devA: 3, devB: 1}
	b := Vector{devA: 1, devB: 2}
	if a.Compare(b) != Concurrent || b.Compare(a) != Concurrent {
		t.Error("concurrent vectors must be concurrent both ways")
	}
	x := Vector{devA: 5}
	y := Vector{devA: 4}
	if x.Compare(y) != Greater || y.Compare(x) != Less {
		t.Error("ordering should invert when arguments swap")
	}
}

func TestIncrement(t *testing.T) {
	v := Vector{}
	v.Increment(devA)
	v.Increment(devA)
	if v[devA] != 2 {
		t.Errorf("counter = %d, want 2", v[devA])
	}
}

func TestMergeTakesMax(t *testing.T) {
	a := Vector{devA: 3, devB: 1}
	b := Vector{devA: 2, devB: 5}
	a.Merge(b)
	if a[devA] != 3 || a[devB] != 5 {
		t.Errorf("merged = %v, want {a:3 b:5}", a)
	}
}

func TestCloneIsIndependent(t *testing.T) {
	original := Vector{devA: 1}
	clone := original.Clone()
	clone.Increment(devA)
	if original[devA] != 1 {
		t.Error("mutating a clone must not affect the original")
	}
}

func TestConcurrentWith(t *testing.T) {
	if !(Vector{devA: 1}).ConcurrentWith(Vector{devB: 1}) {
		t.Error("independent single-device edits should be concurrent")
	}
	if (Vector{devA: 2}).ConcurrentWith(Vector{devA: 1}) {
		t.Error("an ancestor relationship is not concurrent")
	}
}

func TestConflictName(t *testing.T) {
	when := time.Date(2026, 7, 13, 12, 30, 45, 0, time.UTC)
	got := ConflictName("docs/report.txt", "ABCDEFGHIJK", when)
	want := "docs/report.sync-conflict-20260713-123045-ABCDEFG.txt"
	if got != want {
		t.Errorf("ConflictName = %q, want %q", got, want)
	}
	if got := ConflictName("README", "XYZ", when); got != "README.sync-conflict-20260713-123045-XYZ" {
		t.Errorf("extensionless conflict name wrong: %q", got)
	}
}

func TestOrderingString(t *testing.T) {
	if Concurrent.String() != "concurrent" || Equal.String() != "equal" {
		t.Error("unexpected Ordering.String()")
	}
}
