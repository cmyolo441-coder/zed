package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Shell runs shell commands cross-platform (PowerShell on Windows, sh elsewhere).
type Shell struct {
	WorkDir string
	Timeout time.Duration
}

func (t *Shell) Name() string { return "run_shell" }
func (t *Shell) Description() string {
	return "Execute a shell command in the project directory and return combined stdout/stderr. Supports optional cwd, timeout_seconds, env, and max_output_bytes. Always returns exit information instead of hiding failures."
}
func (t *Shell) RequiresApproval() bool { return true }
func (t *Shell) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command":         map[string]any{"type": "string", "description": "The shell command to run."},
			"cwd":             map[string]any{"type": "string", "description": "Optional working directory relative to project root."},
			"timeout_seconds": map[string]any{"type": "integer", "description": "Optional timeout in seconds."},
			"env":             map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
			"max_output_bytes": map[string]any{"type": "integer", "description": "Optional output truncation limit."},
		},
		"required": []string{"command"},
	}
}

func (t *Shell) Execute(ctx context.Context, args string) (string, error) {
	var a struct {
		Command        string            `json:"command"`
		CWD            string            `json:"cwd"`
		TimeoutSeconds int               `json:"timeout_seconds"`
		Env            map[string]string `json:"env"`
		MaxOutputBytes int               `json:"max_output_bytes"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	a.Command = strings.TrimSpace(a.Command)
	if a.Command == "" {
		return "", fmt.Errorf("empty command")
	}
	wd := t.WorkDir
	if a.CWD != "" {
		resolved, err := resolvePath(t.WorkDir, a.CWD)
		if err != nil { return "", err }
		wd = resolved
	}
	if info, err := os.Stat(wd); err != nil || !info.IsDir() {
		return "", fmt.Errorf("working directory does not exist or is not a directory: %s", wd)
	}
	timeout := t.Timeout
	if timeout == 0 { timeout = 5 * time.Minute }
	if a.TimeoutSeconds > 0 { timeout = time.Duration(a.TimeoutSeconds) * time.Second }
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(cctx, "powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", a.Command)
	} else {
		shell := os.Getenv("SHELL"); if shell == "" { shell = "/bin/sh" }
		cmd = exec.CommandContext(cctx, shell, "-c", a.Command)
	}
	cmd.Dir = wd
	cmd.Env = os.Environ()
	for k, v := range a.Env { cmd.Env = append(cmd.Env, k+"="+v) }

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	start := time.Now()
	err := cmd.Run()
	dur := time.Since(start).Round(time.Millisecond)
	out := buf.String()
	if a.MaxOutputBytes <= 0 { a.MaxOutputBytes = 20000 }
	truncated := false
	if len(out) > a.MaxOutputBytes { out = out[:a.MaxOutputBytes] + "\n…[shell output truncated]"; truncated = true }
	exitCode := 0
	if err != nil {
		exitCode = 1
		if ee, ok := err.(*exec.ExitError); ok { exitCode = ee.ExitCode() }
	}
	status := "success"
	if cctx.Err() == context.DeadlineExceeded { status = "timeout"; exitCode = -1 } else if err != nil { status = "failed" }
	if out == "" { out = "(no output)" }
	return fmt.Sprintf("command: %s\ncwd: %s\nstatus: %s\nexit_code: %d\nduration: %s\ntruncated: %v\n\n%s", a.Command, filepath.ToSlash(wd), status, exitCode, dur, truncated, out), nil
}
