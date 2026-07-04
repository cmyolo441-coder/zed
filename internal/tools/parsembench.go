package tools

import (
	"context"
	"fmt"

	"github.com/gjkjk/zed/internal/profiler"
)

type ParseBenchmark struct{}

func (t *ParseBenchmark) Name() string { return "parse_benchmark" }
func (t *ParseBenchmark) Description() string {
	return "Parse Go benchmark output (`go test -bench`) and return a human-readable " +
		"summary with iterations, ns/op, MB/s, and allocs/op. " +
		"Args: {\"output\": \"the raw benchmark output text\"}"
}
func (t *ParseBenchmark) RequiresApproval() bool { return false }
func (t *ParseBenchmark) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"output": map[string]any{"type": "string", "description": "Raw Go benchmark output."},
		},
		"required": []string{"output"},
	}
}
func (t *ParseBenchmark) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Output string `json:"output"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	results := profiler.ParseGoBenchmark(a.Output)
	return profiler.FormatResults(results), nil
}

var _ Tool = (*ParseBenchmark)(nil)
