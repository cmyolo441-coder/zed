package tools

import (
	"context"
	"fmt"

	"github.com/gjkjk/zed/internal/profiler"
)

type ProfileCode struct {
	WorkDir string
}

func (t *ProfileCode) Name() string { return "profile_code" }
func (t *ProfileCode) Description() string {
	return "Detect the appropriate profiling command for the project language " +
		"(Go, Python, Node.js, Rust) and return it. " +
		"The agent can then run the suggested command via the shell tool and " +
		"optionally parse benchmark results with parse_benchmark. " +
		"Args: {\"path\": \"optional subdirectory\"}"
}
func (t *ProfileCode) RequiresApproval() bool { return false }
func (t *ProfileCode) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "Optional subdirectory to profile."},
		},
	}
}
func (t *ProfileCode) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	wd := t.WorkDir
	if a.Path != "" {
		wd = a.Path
	}
	result, err := profiler.Profile(nil, wd)
	if err != nil {
		return "", err
	}
	return result.Output, nil
}

var _ Tool = (*ProfileCode)(nil)
