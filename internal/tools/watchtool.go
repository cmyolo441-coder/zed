package tools

import (
	"context"
	"fmt"

	"github.com/cmyolo441-coder/zed/internal/watcher"
)

type WatcherStatus struct {
	Watcher *watcher.Watcher
}

func (t *WatcherStatus) Name() string { return "watcher_status" }
func (t *WatcherStatus) Description() string {
	return "Return the file watcher status: which directory is monitored, " +
		"which file extensions are watched, and current state. " +
		"Args: {} (no arguments)"
}
func (t *WatcherStatus) RequiresApproval() bool { return false }
func (t *WatcherStatus) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{},
	}
}
func (t *WatcherStatus) Execute(_ context.Context, args string) (string, error) {
	var a struct{}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if t.Watcher == nil {
		return "File watcher is not active.", nil
	}
	return t.Watcher.Summary(), nil
}

var _ Tool = (*WatcherStatus)(nil)
