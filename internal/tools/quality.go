package tools

import (
	"context"
	"fmt"

	"github.com/cmyolo441-coder/zed/internal/quality"
)

// CodeQuality is a tool that runs static analysis and security checks.
type CodeQuality struct {
	WorkDir string
}

func (t *CodeQuality) Name() string { return "code_quality" }
func (t *CodeQuality) Description() string {
	return "Run static analysis on the codebase. Detects: " +
		"security vulnerabilities (SQL injection, XSS, hardcoded secrets, path traversal, command injection), " +
		"code smells (TODOs, empty catches, debug prints, magic numbers), " +
		"code duplication, and complexity. " +
		"Returns a quality score (0-10) and actionable fix suggestions. " +
		"Args: {\"path\": \"file or directory to analyze\"}"
}
func (t *CodeQuality) RequiresApproval() bool { return false }
func (t *CodeQuality) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "File or directory to analyze."},
		},
		"required": []string{"path"},
	}
}

func (t *CodeQuality) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.Path == "" {
		a.Path = t.WorkDir
	}
	report, err := quality.Analyze(a.Path)
	if err != nil {
		return "", fmt.Errorf("quality analysis failed: %w", err)
	}
	return report.Summary(), nil
}

var _ Tool = (*CodeQuality)(nil)
