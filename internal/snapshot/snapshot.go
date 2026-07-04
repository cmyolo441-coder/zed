// Package snapshot records the state of files before the agent modifies them,
// enabling multi-level undo and redo of file operations. Each mutating tool
// captures a snapshot before applying its change; the user can then revert.
package snapshot

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Kind describes the type of file operation captured.
type Kind int

const (
	KindCreate Kind = iota // file did not exist before
	KindModify             // file existed and was changed
	KindDelete             // file existed and was removed
)

func (k Kind) String() string {
	switch k {
	case KindCreate:
		return "create"
	case KindModify:
		return "modify"
	case KindDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// Snapshot captures the before/after state of a single file for one operation.
type Snapshot struct {
	Path        string
	Kind        Kind
	Before      []byte // file content before the change (nil if it didn't exist)
	After       []byte // file content after the change (nil if deleted)
	ExistedPre  bool
	ExistedPost bool
	Time        time.Time
	Tool        string // which tool produced this change
}

// Manager maintains undo/redo stacks of snapshots. It is safe for concurrent use.
type Manager struct {
	mu   sync.Mutex
	undo []*Snapshot
	redo []*Snapshot
	max  int
}

// NewManager creates a snapshot manager keeping up to maxDepth entries.
func NewManager(maxDepth int) *Manager {
	if maxDepth <= 0 {
		maxDepth = 100
	}
	return &Manager{max: maxDepth}
}

// Capture reads the current on-disk state of path and returns a partially
// filled Snapshot. Call Commit with the post-change content to finalize it.
func Capture(path, tool string) (*Snapshot, error) {
	s := &Snapshot{Path: path, Tool: tool, Time: time.Now()}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s.ExistedPre = false
			return s, nil
		}
		return nil, err
	}
	s.ExistedPre = true
	s.Before = data
	return s, nil
}

// Commit finalizes a snapshot with the post-change state and pushes it onto the
// undo stack, clearing the redo stack (standard undo semantics).
func (m *Manager) Commit(s *Snapshot) {
	// Determine kind based on before/after existence.
	after, err := os.ReadFile(s.Path)
	if err == nil {
		s.ExistedPost = true
		s.After = after
	} else {
		s.ExistedPost = false
	}
	switch {
	case !s.ExistedPre && s.ExistedPost:
		s.Kind = KindCreate
	case s.ExistedPre && !s.ExistedPost:
		s.Kind = KindDelete
	default:
		s.Kind = KindModify
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.undo = append(m.undo, s)
	if len(m.undo) > m.max {
		m.undo = m.undo[len(m.undo)-m.max:]
	}
	m.redo = nil
}

// CanUndo reports whether there is anything to undo.
func (m *Manager) CanUndo() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.undo) > 0
}

// CanRedo reports whether there is anything to redo.
func (m *Manager) CanRedo() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.redo) > 0
}

// Undo reverts the most recent change and moves it to the redo stack.
// It returns a human-readable description of what was reverted.
func (m *Manager) Undo() (string, error) {
	m.mu.Lock()
	if len(m.undo) == 0 {
		m.mu.Unlock()
		return "", fmt.Errorf("nothing to undo")
	}
	s := m.undo[len(m.undo)-1]
	m.undo = m.undo[:len(m.undo)-1]
	m.mu.Unlock()

	if err := restore(s, true); err != nil {
		return "", err
	}

	m.mu.Lock()
	m.redo = append(m.redo, s)
	m.mu.Unlock()

	return fmt.Sprintf("undid %s on %s", s.Kind, s.Path), nil
}

// Redo re-applies the most recently undone change.
func (m *Manager) Redo() (string, error) {
	m.mu.Lock()
	if len(m.redo) == 0 {
		m.mu.Unlock()
		return "", fmt.Errorf("nothing to redo")
	}
	s := m.redo[len(m.redo)-1]
	m.redo = m.redo[:len(m.redo)-1]
	m.mu.Unlock()

	if err := restore(s, false); err != nil {
		return "", err
	}

	m.mu.Lock()
	m.undo = append(m.undo, s)
	m.mu.Unlock()

	return fmt.Sprintf("redid %s on %s", s.Kind, s.Path), nil
}

// restore writes the appropriate content back to disk. When undo is true it
// restores the "before" state; otherwise it restores the "after" state.
func restore(s *Snapshot, undo bool) error {
	if undo {
		// Restore pre-change state.
		if !s.ExistedPre {
			// File was created; undo means delete it.
			return os.Remove(s.Path)
		}
		return os.WriteFile(s.Path, s.Before, 0o644)
	}
	// Redo: restore post-change state.
	if !s.ExistedPost {
		return os.Remove(s.Path)
	}
	return os.WriteFile(s.Path, s.After, 0o644)
}

// History returns a copy of the undo stack descriptions, newest last.
func (m *Manager) History() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.undo))
	for _, s := range m.undo {
		out = append(out, fmt.Sprintf("%s %s (%s)", s.Kind, s.Path, s.Tool))
	}
	return out
}

// Depth returns the number of undoable operations.
func (m *Manager) Depth() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.undo)
}
