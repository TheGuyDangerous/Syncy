// Package fswatch is a recursive filesystem watcher that emits debounced
// batches of changed paths, used to trigger rescans in the sync engine.
package fswatch

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const defaultDebounce = 200 * time.Millisecond

type Event struct {
	Paths []string
}

type Watcher struct {
	root      string
	debounce  time.Duration
	fsw       *fsnotify.Watcher
	events    chan Event
	errs      chan error
	done      chan struct{}
	closeOnce sync.Once
}

type Option func(*Watcher)

func WithDebounce(d time.Duration) Option {
	return func(w *Watcher) {
		if d > 0 {
			w.debounce = d
		}
	}
}

func New(dir string, opts ...Option) (*Watcher, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		root:     abs,
		debounce: defaultDebounce,
		fsw:      fsw,
		events:   make(chan Event),
		errs:     make(chan error, 1),
		done:     make(chan struct{}),
	}
	for _, opt := range opts {
		opt(w)
	}
	if err := w.addRecursive(abs); err != nil {
		_ = fsw.Close()
		return nil, err
	}
	go w.run()
	return w, nil
}

func (w *Watcher) Events() <-chan Event { return w.events }

func (w *Watcher) Errors() <-chan error { return w.errs }

func (w *Watcher) Close() error {
	var err error
	w.closeOnce.Do(func() {
		close(w.done)
		err = w.fsw.Close()
	})
	return err
}

func (w *Watcher) addRecursive(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return w.fsw.Add(path)
		}
		return nil
	})
}

func (w *Watcher) run() {
	defer close(w.events)

	pending := make(map[string]struct{})
	timer := time.NewTimer(w.debounce)
	if !timer.Stop() {
		<-timer.C
	}
	arm := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(w.debounce)
	}

	for {
		select {
		case <-w.done:
			timer.Stop()
			return

		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if ev.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					_ = w.addRecursive(ev.Name)
				}
			}
			pending[ev.Name] = struct{}{}
			arm()

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			select {
			case w.errs <- err:
			default:
			}

		case <-timer.C:
			if len(pending) == 0 {
				continue
			}
			paths := make([]string, 0, len(pending))
			for p := range pending {
				paths = append(paths, p)
			}
			sort.Strings(paths)
			pending = make(map[string]struct{})
			select {
			case w.events <- Event{Paths: paths}:
			case <-w.done:
				return
			}
		}
	}
}
