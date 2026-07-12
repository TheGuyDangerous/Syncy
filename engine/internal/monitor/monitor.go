// Package monitor turns filesystem activity into a live stream of index changes.
package monitor

import (
	"sync"

	"github.com/TheGuyDangerous/Syncy/engine/internal/fswatch"
	"github.com/TheGuyDangerous/Syncy/engine/internal/scanner"
)

type Monitor struct {
	root      string
	sc        *scanner.Scanner
	watcher   *fswatch.Watcher
	baseline  *scanner.Index
	changes   chan []scanner.Change
	errs      chan error
	done      chan struct{}
	closeOnce sync.Once
}

// New scans root for a baseline index, starts watching it, and returns the
// monitor together with that baseline. Subsequent changes arrive on Changes.
func New(root string, sc *scanner.Scanner, watchOpts ...fswatch.Option) (*Monitor, *scanner.Index, error) {
	if sc == nil {
		var err error
		sc, err = scanner.New(nil)
		if err != nil {
			return nil, nil, err
		}
	}
	baseline, err := sc.Scan(root)
	if err != nil {
		return nil, nil, err
	}
	w, err := fswatch.New(root, watchOpts...)
	if err != nil {
		return nil, nil, err
	}
	m := &Monitor{
		root:     root,
		sc:       sc,
		watcher:  w,
		baseline: baseline,
		changes:  make(chan []scanner.Change),
		errs:     make(chan error, 1),
		done:     make(chan struct{}),
	}
	go m.run()
	return m, baseline, nil
}

func (m *Monitor) Changes() <-chan []scanner.Change { return m.changes }

func (m *Monitor) Errors() <-chan error { return m.errs }

func (m *Monitor) Close() error {
	var err error
	m.closeOnce.Do(func() {
		close(m.done)
		err = m.watcher.Close()
	})
	return err
}

func (m *Monitor) run() {
	defer close(m.changes)
	for {
		select {
		case <-m.done:
			return

		case _, ok := <-m.watcher.Events():
			if !ok {
				return
			}
			newIdx, err := m.sc.Scan(m.root)
			if err != nil {
				m.emitErr(err)
				continue
			}
			changes := scanner.Diff(m.baseline, newIdx)
			m.baseline = newIdx
			if len(changes) == 0 {
				continue
			}
			select {
			case m.changes <- changes:
			case <-m.done:
				return
			}

		case err := <-m.watcher.Errors():
			m.emitErr(err)
		}
	}
}

func (m *Monitor) emitErr(err error) {
	select {
	case m.errs <- err:
	default:
	}
}
