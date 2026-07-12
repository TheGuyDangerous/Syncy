// Package conflict provides version vectors for detecting concurrent edits and
// naming conflict copies.
package conflict

import (
	"path"
	"time"

	"github.com/TheGuyDangerous/Syncy/engine/internal/core"
)

type Ordering int

const (
	Equal Ordering = iota
	Greater
	Less
	Concurrent
)

func (o Ordering) String() string {
	switch o {
	case Equal:
		return "equal"
	case Greater:
		return "greater"
	case Less:
		return "less"
	case Concurrent:
		return "concurrent"
	default:
		return "unknown"
	}
}

type Vector map[core.DeviceID]uint64

func (v Vector) Clone() Vector {
	out := make(Vector, len(v))
	for k, n := range v {
		out[k] = n
	}
	return out
}

func (v Vector) Increment(id core.DeviceID) {
	v[id]++
}

func (v Vector) Merge(other Vector) {
	for k, n := range other {
		if n > v[k] {
			v[k] = n
		}
	}
}

func (v Vector) Compare(other Vector) Ordering {
	var greater, less bool
	for k, n := range v {
		switch {
		case n > other[k]:
			greater = true
		case n < other[k]:
			less = true
		}
	}
	for k, n := range other {
		if _, ok := v[k]; !ok && n > 0 {
			less = true
		}
	}
	switch {
	case greater && less:
		return Concurrent
	case greater:
		return Greater
	case less:
		return Less
	default:
		return Equal
	}
}

func (v Vector) ConcurrentWith(other Vector) bool {
	return v.Compare(other) == Concurrent
}

func ConflictName(relPath string, id core.DeviceID, when time.Time) string {
	ext := path.Ext(relPath)
	stem := relPath[:len(relPath)-len(ext)]
	short := string(id)
	if len(short) > 7 {
		short = short[:7]
	}
	return stem + ".sync-conflict-" + when.UTC().Format("20060102-150405") + "-" + short + ext
}
