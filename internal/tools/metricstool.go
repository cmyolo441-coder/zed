package tools

import (
	"context"
	"fmt"

	"github.com/cmyolo441-coder/zed/internal/metrics"
)

type CodeMetrics struct {
	Dashboard *metrics.Dashboard
}

func (t *CodeMetrics) Name() string { return "code_metrics" }
func (t *CodeMetrics) Description() string {
	return "Return the current code metrics dashboard: project health score (0-10), " +
		"technical debt, and trends for complexity, coverage, LOC, files, functions, " +
		"and issues over time. " +
		"Args: {} (no arguments)"
}
func (t *CodeMetrics) RequiresApproval() bool { return false }
func (t *CodeMetrics) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{},
	}
}
func (t *CodeMetrics) Execute(_ context.Context, args string) (string, error) {
	var a struct{}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	return t.Dashboard.Render(), nil
}

var _ Tool = (*CodeMetrics)(nil)
