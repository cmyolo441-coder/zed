package snapshot

import (
	"fmt"
	"strings"
)

// Diff generates a unified diff between the before and after states of a snapshot.
func (s *Snapshot) Diff() string {
	if s.Before == nil && s.After == nil {
		return ""
	}
	beforeLines := splitLines(string(s.Before))
	afterLines := splitLines(string(s.After))

	var b strings.Builder
	fmt.Fprintf(&b, "--- %s (before)\n+++ %s (after)\n", s.Path, s.Path)

	// Simple line-by-line diff.
	maxLen := len(beforeLines)
	if len(afterLines) > maxLen {
		maxLen = len(afterLines)
	}
	for i := 0; i < maxLen; i++ {
		var before, after string
		if i < len(beforeLines) {
			before = beforeLines[i]
		}
		if i < len(afterLines) {
			after = afterLines[i]
		}
		if before == after {
			fmt.Fprintf(&b, " %s\n", before)
		} else {
			if before != "" {
				fmt.Fprintf(&b, "-%s\n", before)
			}
			if after != "" {
				fmt.Fprintf(&b, "+%s\n", after)
			}
		}
	}
	return b.String()
}

// DiffSummary returns a short summary of what changed.
func (s *Snapshot) DiffSummary() string {
	added, removed := 0, 0
	beforeLines := splitLines(string(s.Before))
	afterLines := splitLines(string(s.After))
	maxLen := len(beforeLines)
	if len(afterLines) > maxLen {
		maxLen = len(afterLines)
	}
	for i := 0; i < maxLen; i++ {
		var before, after string
		if i < len(beforeLines) {
			before = beforeLines[i]
		}
		if i < len(afterLines) {
			after = afterLines[i]
		}
		if before != after {
			if before != "" {
				removed++
			}
			if after != "" {
				added++
			}
		}
	}
	return fmt.Sprintf("%s: +%d -%d lines", s.Path, added, removed)
}

// HistoryDetailed returns detailed history with diff summaries.
func (m *Manager) HistoryDetailed() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.undo))
	for i, s := range m.undo {
		out = append(out, fmt.Sprintf("#%d [%s] %s — %s", i+1, s.Tool, s.Kind, s.DiffSummary()))
	}
	return out
}

// RollbackTo undoes all changes after the given index (1-based).
// Returns a description of what was rolled back.
func (m *Manager) RollbackTo(index int) (string, error) {
	m.mu.Lock()
	if index < 1 || index > len(m.undo) {
		m.mu.Unlock()
		return "", fmt.Errorf("invalid rollback index %d (valid: 1-%d)", index, len(m.undo))
	}
	toUndo := len(m.undo) - index
	snapshots := make([]*Snapshot, toUndo)
	copy(snapshots, m.undo[index:])
	m.undo = m.undo[:index]
	m.mu.Unlock()

	var undone []string
	for i := len(snapshots) - 1; i >= 0; i-- {
		s := snapshots[i]
		if err := restore(s, true); err != nil {
			return fmt.Sprintf("rolled back %d/%d changes (failed at %s: %v)", len(undone), toUndo, s.Path, err), err
		}
		undone = append(undone, s.Path)
	}
	return fmt.Sprintf("rolled back %d changes to snapshot #%d", len(undone), index), nil
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}
