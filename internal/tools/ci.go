package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gjkjk/zed/internal/ci"
)

// CIPipeline is a tool that runs the full CI pipeline (lint → build → test → security).
type CIPipeline struct {
	WorkDir string
}

func (t *CIPipeline) Name() string { return "ci_pipeline" }
func (t *CIPipeline) Description() string {
	return "Run the full CI pipeline: lint → build → test → security scan. " +
		"Auto-detects project language(s) (Go, Node, Python, Rust) and runs appropriate commands. " +
		"Returns a pass/fail report for each stage. " +
		"Args: {\"stage\": \"optional: run only one stage (lint|build|test|security)\", \"path\": \"optional: project root\"}"
}
func (t *CIPipeline) RequiresApproval() bool { return false }
func (t *CIPipeline) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"stage": map[string]any{"type": "string", "description": "Optional: run only this stage (lint, build, test, security)."},
			"path":  map[string]any{"type": "string", "description": "Optional: project root directory (defaults to work dir)."},
		},
	}
}

func (t *CIPipeline) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Stage string `json:"stage"`
		Path  string `json:"path"`
	}
	if args != "" && args != "{}" {
		if err := parseArgs(args, &a); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}

	workDir := t.WorkDir
	if a.Path != "" {
		workDir = a.Path
	}

	project := ci.Detect(workDir)
	if len(project.Languages) == 0 {
		return "No supported project detected. Supported: Go, Node.js, Python, Rust, Java (Maven/Gradle), C# (.NET), Ruby.", nil
	}

	if a.Stage != "" {
		return t.runStage(project, a.Stage)
	}

	return t.runFullPipeline(project)
}

func (t *CIPipeline) runStage(project *ci.DetectedProject, stage string) (string, error) {
	stages := project.Pipeline()
	for _, s := range stages {
		if strings.Contains(s.Name, stage) {
			if s.Command == "" {
				return fmt.Sprintf("Stage %q has no command (use code_quality tool for security scan).", s.Name), nil
			}
			out, err := t.execCmd(s.Command)
			if err != nil {
				return fmt.Sprintf("❌ %s FAILED:\n%s", s.Name, out), nil
			}
			return fmt.Sprintf("✅ %s PASSED:\n%s", s.Name, out), nil
		}
	}
	return fmt.Sprintf("Stage %q not found. Available: lint, build, test, security.", stage), nil
}

func (t *CIPipeline) runFullPipeline(project *ci.DetectedProject) (string, error) {
	stages := project.Pipeline()
	var b strings.Builder
	b.WriteString(project.Summary())
	b.WriteString("\n\n🚀 Running CI Pipeline:\n\n")

	passed, failed := 0, 0
	for _, s := range stages {
		if s.Command == "" {
			b.WriteString(fmt.Sprintf("  ⏭️  %s (skipped — use code_quality tool)\n", s.Name))
			continue
		}
		out, err := t.execCmd(s.Command)
		if err != nil || strings.Contains(out, "error") || strings.Contains(out, "FAIL") {
			failed++
			b.WriteString(fmt.Sprintf("  ❌ %s FAILED\n", s.Name))
			b.WriteString(fmt.Sprintf("     %s\n", truncateOutput(out, 500)))
		} else {
			passed++
			b.WriteString(fmt.Sprintf("  ✅ %s PASSED\n", s.Name))
		}
	}

	b.WriteString(fmt.Sprintf("\n📊 Pipeline: %d passed, %d failed\n", passed, failed))
	if failed > 0 {
		b.WriteString("\n⚠️  Fix the failing stages before declaring the task complete.\n")
	}
	return b.String(), nil
}

// execCmd runs a shell command directly (PowerShell on Windows, sh elsewhere).
func (t *CIPipeline) execCmd(command string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	cmd.Dir = t.WorkDir

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()

	out := buf.String()
	if ctx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("command timed out after 5 minutes")
	}
	if err != nil {
		return fmt.Sprintf("[exit error: %v]\n%s", err, out), err
	}
	if out == "" {
		out = "(no output)"
	}
	return out, nil
}

func truncateOutput(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

var _ Tool = (*CIPipeline)(nil)
