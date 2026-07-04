package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gjkjk/zed/internal/ast"
)

// AnalyzeCode is a tool that runs AST-based code analysis on a file or directory.
type AnalyzeCode struct {
	WorkDir string
}

func (t *AnalyzeCode) Name() string { return "analyze_code" }
func (t *AnalyzeCode) Description() string {
	return "Run AST-based code intelligence on a file or directory. " +
		"Detects: symbols (functions, types, classes), unused imports, dead code, " +
		"circular dependencies, and cyclomatic complexity. " +
		"Supports Go (real go/ast parser), Python, JavaScript, TypeScript. " +
		"Args: {\"path\": \"file or dir to analyze\", \"check_circular\": true}"
}
func (t *AnalyzeCode) RequiresApproval() bool { return false }
func (t *AnalyzeCode) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":           map[string]any{"type": "string", "description": "File or directory to analyze."},
			"check_circular": map[string]any{"type": "boolean", "description": "Also check for circular dependencies."},
		},
		"required": []string{"path"},
	}
}

func (t *AnalyzeCode) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Path          string `json:"path"`
		CheckCircular bool   `json:"check_circular"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	target := a.Path
	if !strings.HasPrefix(target, "/") && !strings.Contains(target, ":") {
		target = t.WorkDir + "/" + a.Path
	}

	// Check if it's a single file or directory.
	var analyses []*ast.Analysis
	var err error
	if isDir(target) {
		analyses, err = ast.AnalyzeDir(target)
	} else {
		var an *ast.Analysis
		an, err = ast.AnalyzeFile(target)
		if an != nil {
			analyses = []*ast.Analysis{an}
		}
	}
	if err != nil {
		return "", fmt.Errorf("analysis failed: %w", err)
	}
	if len(analyses) == 0 {
		return "No analyzable source files found.", nil
	}

	var b strings.Builder
	b.WriteString("🔍 AST Code Analysis\n\n")
	totalSyms := 0
	totalIssues := 0
	totalDead := 0
	totalUnused := 0
	for _, an := range analyses {
		b.WriteString(an.Summary())
		b.WriteString("\n")
		totalSyms += len(an.Symbols)
		totalIssues += len(an.Issues)
		totalDead += len(an.DeadCode)
		totalUnused += len(an.UnusedImports)
	}
	b.WriteString(fmt.Sprintf("\n📈 Total: %d files | %d symbols | %d issues | %d dead code | %d unused imports\n",
		len(analyses), totalSyms, totalIssues, totalDead, totalUnused))

	if a.CheckCircular {
		cycles := ast.DetectCircularDependencies(analyses)
		if len(cycles) > 0 {
			b.WriteString("\n🔄 Circular Dependencies Detected:\n")
			for _, cycle := range cycles {
				b.WriteString("  " + strings.Join(cycle, " → ") + "\n")
			}
		} else {
			b.WriteString("\n✅ No circular dependencies found.\n")
		}
	}
	return b.String(), nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
