package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/gjkjk/zed/internal/snapshot"
)

// SnapshotHistory is a tool to view file change history and diffs.
type SnapshotHistory struct {
	Snapshots *snapshot.Manager
}

func (t *SnapshotHistory) Name() string { return "snapshot_history" }
func (t *SnapshotHistory) Description() string {
	return "View the history of file changes made by the agent, with diff summaries. " +
		"Shows what was changed, when, and by which tool. " +
		"Args: {\"diff\": \"optional index to see full diff of a specific change\"}"
}
func (t *SnapshotHistory) RequiresApproval() bool { return false }
func (t *SnapshotHistory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"diff": map[string]any{"type": "integer", "description": "Optional: show full diff for this change index (1-based)."},
		},
	}
}

func (t *SnapshotHistory) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Diff int `json:"diff"`
	}
	if args != "" && args != "{}" {
		if err := parseArgs(args, &a); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}

	if a.Diff > 0 {
		// Show full diff for a specific snapshot.
		hist := t.Snapshots.HistoryDetailed()
		if a.Diff > len(hist) {
			return "", fmt.Errorf("snapshot #%d not found (max: %d)", a.Diff, len(hist))
		}
		// We need to access the snapshot directly — use History for now.
		return fmt.Sprintf("Snapshot #%d:\n%s", a.Diff, hist[a.Diff-1]), nil
	}

	hist := t.Snapshots.HistoryDetailed()
	if len(hist) == 0 {
		return "No file changes recorded yet.", nil
	}
	var b strings.Builder
	b.WriteString("📸 File Change History:\n\n")
	for _, h := range hist {
		b.WriteString("  " + h + "\n")
	}
	b.WriteString("\nUse /undo to revert the last change, or snapshot_history with \"diff\" index for details.")
	return b.String(), nil
}

var _ Tool = (*SnapshotHistory)(nil)
