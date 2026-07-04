// Package watcher provides file system watching for hot-reload style
// development. When the agent (or the user) changes a file, the watcher
// triggers relevant tests to run automatically — creating a fast feedback loop.
package watcher

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"os"
)

// EventType describes what happened to a file.
type EventType int

const (
	EventCreate EventType = iota
	EventModify
	EventDelete
)

func (e EventType) String() string {
	switch e {
	case EventCreate:
		return "created"
	case EventModify:
		return "modified"
	case EventDelete:
		return "deleted"
	default:
		return "unknown"
	}
}

// Event represents a file system change.
type Event struct {
	Type EventType
	Path  string
	Time  time.Time
}

// Callback is called when a file changes.
type Callback func(Event)

// Watcher monitors files for changes and invokes callbacks.
type Watcher struct {
	root      string
	patterns  []string // file extensions to watch
	callbacks []Callback
	stop      chan struct{}
	mu        sync.Mutex
	running   bool
	lastMod   map[string]time.Time // debounce
}

// New creates a file watcher for the given root directory.
func New(root string, extensions []string) *Watcher {
	return &Watcher{
		root:     root,
		patterns: extensions,
		stop:     make(chan struct{}),
		lastMod:  make(map[string]time.Time),
	}
}

// OnChange registers a callback for file changes.
func (w *Watcher) OnChange(cb Callback) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, cb)
}

// Start begins polling for file changes. It runs in a goroutine.
func (w *Watcher) Start(ctx context.Context) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	go w.poll(ctx)
}

// Stop stops the watcher.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		close(w.stop)
		w.running = false
	}
}

func (w *Watcher) poll(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		case <-ticker.C:
			w.scan()
		}
	}
}

func (w *Watcher) scan() {
	filepath.Walk(w.root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		// Skip common ignore dirs.
		for _, skip := range []string{".git", "node_modules", "vendor", "dist", "build"} {
			if strings.Contains(path, skip) {
				return nil
			}
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !w.matchesExtension(ext) {
			return nil
		}
		modTime := info.ModTime()
		if last, ok := w.lastMod[path]; ok {
			if modTime.Sub(last) > 100*time.Millisecond {
				w.fire(Event{Type: EventModify, Path: path, Time: modTime})
			}
		} else {
			w.fire(Event{Type: EventCreate, Path: path, Time: modTime})
		}
		w.lastMod[path] = modTime
		return nil
	})
}

func (w *Watcher) matchesExtension(ext string) bool {
	for _, p := range w.patterns {
		if ext == p {
			return true
		}
	}
	return len(w.patterns) == 0
}

func (w *Watcher) fire(e Event) {
	w.mu.Lock()
	cbs := make([]Callback, len(w.callbacks))
	copy(cbs, w.callbacks)
	w.mu.Unlock()
	for _, cb := range cbs {
		cb(e)
	}
}

// RelevantTests returns the test files that should run when a source file changes.
// Convention: foo.go → foo_test.go, foo.py → test_foo.py, foo.js → foo.test.js
func RelevantTests(srcPath string) []string {
	ext := filepath.Ext(srcPath)
	base := strings.TrimSuffix(srcPath, ext)
	switch ext {
	case ".go":
		return []string{base + "_test.go"}
	case ".py":
		dir := filepath.Dir(srcPath)
		name := filepath.Base(srcPath)
		return []string{filepath.Join(dir, "test_"+name), filepath.Join(dir, "test_"+strings.TrimSuffix(name, ext)+".py")}
	case ".js", ".jsx", ".ts", ".tsx":
		return []string{base + ".test" + ext, base + ".spec" + ext}
	default:
		return nil
	}
}

// Summary returns a description of the watcher state.
func (w *Watcher) Summary() string {
	return fmt.Sprintf("File watcher: monitoring %s for %v changes", w.root, w.patterns)
}
