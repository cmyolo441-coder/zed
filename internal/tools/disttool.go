package tools

import (
	"context"
	"fmt"

	"github.com/cmyolo441-coder/zed/internal/dist"
)

type DistStatus struct {
	Network *dist.Network
}

func (t *DistStatus) Name() string { return "dist_status" }
func (t *DistStatus) Description() string {
	return "Return the distributed agent network status: registered nodes, " +
		"task queue, and work distribution. " +
		"Args: {} (no arguments)"
}
func (t *DistStatus) RequiresApproval() bool { return false }
func (t *DistStatus) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{},
	}
}
func (t *DistStatus) Execute(_ context.Context, args string) (string, error) {
	var a struct{}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	return t.Network.Status(), nil
}

type DistSubmit struct {
	Network *dist.Network
}

func (t *DistSubmit) Name() string { return "dist_submit" }
func (t *DistSubmit) Description() string {
	return "Submit a task to the distributed agent network for parallel execution. " +
		"Args: {\"description\": \"what to do\"}"
}
func (t *DistSubmit) RequiresApproval() bool { return true }
func (t *DistSubmit) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"description": map[string]any{"type": "string", "description": "Task description."},
		},
		"required": []string{"description"},
	}
}
func (t *DistSubmit) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Description string `json:"description"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	task := &dist.Task{
		ID:          fmt.Sprintf("task-%d", len(t.Network.Status())),
		Description: a.Description,
	}
	t.Network.Submit(task)
	return fmt.Sprintf("Task %q submitted to distributed network.", task.ID), nil
}

var (
	_ Tool = (*DistStatus)(nil)
	_ Tool = (*DistSubmit)(nil)
)
