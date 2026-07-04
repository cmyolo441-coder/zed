package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	appctx "github.com/cmyolo441-coder/zed/internal/context"
	"github.com/cmyolo441-coder/zed/internal/config"
	"github.com/cmyolo441-coder/zed/internal/llm"
	"github.com/cmyolo441-coder/zed/internal/logging"
	"github.com/cmyolo441-coder/zed/internal/retry"
	"github.com/cmyolo441-coder/zed/internal/security"
	"github.com/cmyolo441-coder/zed/internal/snapshot"
	"github.com/cmyolo441-coder/zed/internal/tools"
)

// EventKind classifies an event streamed out of the agent.
type EventKind int

const (
	EventText     EventKind = iota // incremental assistant text
	EventToolCall                  // agent is about to run a tool
	EventToolDone                  // a tool finished, carries its result
	EventDone                      // the whole turn is complete
	EventError                     // fatal error
	EventNotice                    // transient status (e.g. rate-limit retry) — not fatal
)

// Event is emitted by the agent so the UI can render progress live.
type Event struct {
	Kind     EventKind
	Text     string       // for EventText / EventError
	ToolName string       // for EventToolCall / EventToolDone
	ToolArgs string       // for EventToolCall
	Result   string       // for EventToolDone
	IsError  bool         // for EventToolDone
	Usage    *llm.Usage   // for EventDone
}

// Agent runs the ReAct loop: it calls the LLM, executes any requested tools,
// feeds results back, and repeats until the model stops calling tools.
type Agent struct {
	cfg       *config.Config
	client    llm.Client
	registry  *tools.Registry
	history   []llm.Message
	system    string
	snapshots *snapshot.Manager
	policy    *security.Policy
	logger    *logging.Logger
	ctxMgr    *appctx.Manager
	retryPol  retry.Policy
	effort      config.Effort
	cyberMode   bool // cybersecurity mode flag
	debugFails  int  // consecutive auto-debug failures (resets on pass)
}

// New builds an Agent with a fresh conversation history.
func New(cfg *config.Config, client llm.Client, registry *tools.Registry) *Agent {
	effort := config.EffortProfile(cfg.Effort)
	system := SystemPrompt(cfg.WorkDir, effort)
	return &Agent{
		cfg:      cfg,
		client:   client,
		registry: registry,
		system:   system,
		logger:   logging.Nop(),
		ctxMgr:   appctx.NewManager(system, effortBudget(cfg.Model, effort), nil),
		retryPol: retry.DefaultPolicy(),
		effort:   effort,
	}
}

// SetEffort switches the agent's effort level, rebuilding the system prompt so
// subsequent turns use the new reasoning depth, planning, and verification
// behavior. The change takes effect on the next Run.
func (a *Agent) SetEffort(e config.Effort) {
	a.effort = e
	a.rebuildSystem()
}

// Effort returns the agent's current effort profile.
func (a *Agent) Effort() config.Effort { return a.effort }

// SetCyberMode toggles cybersecurity mode. When enabled, the agent uses a
// security-focused system prompt that makes it think like a penetration
// tester and defensive security engineer.
func (a *Agent) SetCyberMode(enabled bool) {
	a.cyberMode = enabled
	a.rebuildSystem()
}

// CyberMode returns whether cybersecurity mode is active.
func (a *Agent) CyberMode() bool { return a.cyberMode }

// rebuildSystem regenerates the system prompt based on current mode + effort.
func (a *Agent) rebuildSystem() {
	if a.cyberMode {
		a.system = CyberSecurityPrompt(a.cfg.WorkDir, a.effort)
	} else {
		a.system = SystemPrompt(a.cfg.WorkDir, a.effort)
	}
	a.ctxMgr = appctx.NewManager(a.system, effortBudget(a.cfg.Model, a.effort), nil)
}

// SetSnapshots attaches an undo/redo manager.
func (a *Agent) SetSnapshots(m *snapshot.Manager) { a.snapshots = m }

// SetPolicy attaches the security policy.
func (a *Agent) SetPolicy(p *security.Policy) { a.policy = p }

// SetLogger attaches a structured logger.
func (a *Agent) SetLogger(l *logging.Logger) {
	if l != nil {
		a.logger = l
	}
}

// LoadHistory replaces the conversation (used when resuming a session).
func (a *Agent) LoadHistory(msgs []llm.Message) { a.history = msgs }

// History exposes the current conversation (read-only use).
func (a *Agent) History() []llm.Message { return a.history }

// Reset clears the conversation.
func (a *Agent) Reset() { a.history = nil }

// Run processes one user message, driving the ReAct loop and emitting events.
// It blocks until the turn completes; callers typically run it in a goroutine.
func (a *Agent) Run(ctx context.Context, userInput string, emit func(Event)) {
	a.history = append(a.history, llm.Message{Role: llm.RoleUser, Content: userInput})
	a.runEffortPreflight(ctx, userInput, emit)
	a.runGoalBootstrap(ctx, userInput, emit)

	// Effort level governs how hard the agent works this turn.
	maxSteps := a.effort.MaxSteps
	if maxSteps <= 0 {
		maxSteps = a.cfg.MaxSteps
	}
	maxTokens := a.effort.MaxTokens
	if maxTokens <= 0 {
		maxTokens = a.cfg.MaxTokens
	}
	// Cap to the current model's actual limit to avoid 400 errors.
	maxTokens = config.CappedMaxTokens(a.cfg, maxTokens)

	a.logger.Info("turn started",
		logging.F("input_len", len(userInput)),
		logging.F("effort", a.effort.Name),
		logging.F("max_steps", maxSteps))
	turnStart := time.Now()

	for step := 0; step < maxSteps; step++ {
		// Keep the conversation within the model's context budget.
		if a.ctxMgr != nil {
			if fitted, compacted, err := a.ctxMgr.Fit(a.history); err == nil {
				a.history = fitted
				if compacted {
					a.logger.Info("context compacted", logging.F("messages", len(fitted)))
				}
			}
		}

		req := llm.Request{
			Model:     config.EffectiveModel(a.cfg, a.effort.Name),
			System:    a.system,
			Messages:  a.history,
			Tools:     a.registry.Schemas(),
			MaxTokens: maxTokens,
			Stream:    true,
		}

		// Open the stream with retry/backoff on transient failures.
		var stream <-chan llm.StreamEvent
		err := retry.Do(ctx, a.retryPol, func(attempt int, delay time.Duration, e error) {
			a.logger.Warn("llm retry",
				logging.F("attempt", attempt),
				logging.F("delay_ms", delay.Milliseconds()),
				logging.F("err", e.Error()))
			// Let the user see we're waiting out a transient error (429/5xx)
			// instead of staring at a frozen screen.
			emit(Event{Kind: EventNotice, Text: fmt.Sprintf(
				"%s — retry %d/%d in %s", e.Error(), attempt, a.retryPol.MaxAttempts, delay.Round(time.Second))})
		}, func() error {
			s, e := a.client.Stream(ctx, req)
			if e != nil {
				return e
			}
			stream = s
			return nil
		})
		if err != nil {
			a.logger.Error("llm stream failed", logging.F("err", err.Error()))
			emit(Event{Kind: EventError, Text: err.Error()})
			return
		}

		var (
			assistantText string
			toolCalls     []llm.ToolCall
			usage         *llm.Usage
			truncated     bool
		)

		for ev := range stream {
			switch {
			case ev.Err != nil:
				a.logger.Error("stream error", logging.F("err", ev.Err.Error()))
				emit(Event{Kind: EventError, Text: ev.Err.Error()})
				return
			case ev.TextDelta != "":
				assistantText += ev.TextDelta
				emit(Event{Kind: EventText, Text: ev.TextDelta})
			case ev.ToolCall != nil:
				toolCalls = append(toolCalls, *ev.ToolCall)
				if ev.Truncated {
					truncated = true
				}
			case ev.Done:
				usage = ev.Usage
				if ev.Truncated {
					truncated = true
				}
			}
		}

		// Record the assistant turn (text + any tool calls).
		a.history = append(a.history, llm.Message{
			Role:      llm.RoleAssistant,
			Content:   assistantText,
			ToolCalls: toolCalls,
		})

		// No tool calls -> the model is done answering.
		if len(toolCalls) == 0 {
			a.logger.Info("turn complete",
				logging.F("steps", step+1),
				logging.F("ms", time.Since(turnStart).Milliseconds()))
			emit(Event{Kind: EventDone, Usage: usage})
			return
		}

		// Execute tool calls in parallel for speed, then feed results back.
		type toolResult struct {
			call    llm.ToolCall
			content string
			isError bool
		}
		results := make([]toolResult, len(toolCalls))
		var wg sync.WaitGroup
		executeOne := func(idx int, c llm.ToolCall) {
			res := a.registry.Execute(ctx, c)
			if res.IsError {
				a.logger.Warn("tool error", logging.F("tool", c.Name), logging.F("result", res.Content))
			}
			results[idx] = toolResult{call: c, content: res.Content, isError: res.IsError}
		}
		for i, call := range toolCalls {
			emit(Event{Kind: EventToolCall, ToolName: call.Name, ToolArgs: call.Args})
			a.logger.Info("tool call", logging.F("tool", call.Name))

			if truncated && !json.Valid([]byte(call.Args)) {
				msg := "The previous tool call was cut off because it exceeded the output token limit, " +
					"so its arguments are incomplete. Do NOT resend the whole thing. Instead, for large files: " +
					"create the file with write_file using only the FIRST portion of the content, then append the " +
					"rest in multiple follow-up steps using append_file (or edit_file). Keep each single tool call small."
				a.logger.Warn("truncated tool args", logging.F("tool", call.Name), logging.F("args_len", len(call.Args)))
				results[i] = toolResult{call: call, content: "Error: " + msg, isError: true}
				continue
			}

			if a.effort.ParallelTools {
				wg.Add(1)
				go func(idx int, c llm.ToolCall) { defer wg.Done(); executeOne(idx, c) }(i, call)
			} else {
				executeOne(i, call)
			}
		}
		wg.Wait()

		for _, tr := range results {
			emit(Event{
				Kind:     EventToolDone,
				ToolName: tr.call.Name,
				Result:   tr.content,
				IsError:  tr.isError,
			})
			a.history = append(a.history, llm.Message{
				Role:       llm.RoleTool,
				ToolCallID: tr.call.ID,
				ToolName:   tr.call.Name,
				Content:    tr.content,
			})
		}

		// Self-Healing Auto-Debug: after file-modifying tools, if AutoDebug
		// is enabled for this effort level, automatically run a build/test check
		// and inject any errors back into the conversation so the model can fix them.
		if a.effort.AutoDebug && a.detectFileChanges(toolCalls) {
			a.autoDebug(ctx, emit)
		}

		// Post-change quality scan: if the effort level enables it, run code
		// quality analysis after file changes and inject findings so the model
		// can address them in the next iteration.
		if a.effort.RunQualityCheck && a.detectFileChanges(toolCalls) {
			a.runQualityScan(ctx, emit)
		}

		// Post-change security scan: if enabled, scan for vulnerabilities after
		// file changes and inject findings for the model to fix.
		if a.effort.RunSecurityScan && a.detectFileChanges(toolCalls) {
			a.runSecurityScan(ctx, emit)
		}

		// Post-change CI pipeline: if enabled, run the full CI flow after changes.
		if a.effort.EnableCI && a.detectFileChanges(toolCalls) {
			a.runCIPipeline(ctx, emit)
		}
		if a.detectFileChanges(toolCalls) {
			a.runEnterpriseVerification(ctx, emit)
		}
		// Loop again so the model can react to the tool results.
	}

	a.logger.Warn("max steps reached", logging.F("steps", maxSteps))
	emit(Event{Kind: EventError, Text: "reached max steps without completion"})
}

// detectFileChanges returns true if any of the tool calls modify files.
func (a *Agent) detectFileChanges(calls []llm.ToolCall) bool {
	for _, c := range calls {
		switch c.Name {
		case "write_file", "edit_file", "append_file":
			return true
		}
	}
	return false
}

// autoDebug runs a build/test check and injects results into the conversation.
// If the build fails, the model will see the errors and fix them in the next
// ReAct iteration — creating a self-healing loop: build → diagnose → fix → rebuild.
func (a *Agent) autoDebug(ctx context.Context, emit func(Event)) {
	cmd := a.detectBuildCommand()
	if cmd == "" {
		return
	}

	shellTool, ok := a.registry.Get("run_shell")
	if !ok {
		return
	}

	maxFails := a.effort.MaxDebugRetries
	if maxFails <= 0 {
		maxFails = 3
	}

	// If we've already exhausted the debug budget, stop injecting to avoid
	// an infinite build→fix→build loop.
	if a.debugFails >= maxFails {
		a.logger.Warn("auto-debug: budget exhausted", logging.F("fails", a.debugFails), logging.F("max", maxFails))
		a.history = append(a.history, llm.Message{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("[SELF-HEALING] Build/test has failed %d consecutive times. Report the remaining issues to the user and stop retrying.", a.debugFails),
		})
		return
	}

	emit(Event{Kind: EventToolCall, ToolName: "auto_debug", ToolArgs: cmd})
	args := fmt.Sprintf(`{"command": %q}`, cmd)
	result, err := shellTool.Execute(ctx, args)
	if err != nil {
		result = "Error: " + err.Error()
	}

	failed := err != nil || strings.Contains(result, "error") ||
		strings.Contains(result, "FAIL") ||
		strings.Contains(result, "Error") ||
		strings.Contains(strings.ToLower(result), "failed")

	emit(Event{
		Kind:     EventToolDone,
		ToolName: "auto_debug",
		Result:   result,
		IsError:  failed,
	})

	if !failed {
		a.debugFails = 0
		a.logger.Info("auto-debug: build/test passed")
		return
	}

	a.debugFails++
	remaining := maxFails - a.debugFails
	hint := fmt.Sprintf("\n\n(auto-debug: attempt %d/%d failed — %d retries remaining)", a.debugFails, maxFails, remaining)
	if remaining <= 0 {
		hint = "\n\n(auto-debug: FINAL attempt — if this fails, report to user)"
	}

	a.history = append(a.history, llm.Message{
		Role:    llm.RoleUser,
		Content: "[SELF-HEALING] Build/test check failed. Fix these errors:\n\n" + result + hint,
	})
	a.logger.Info("auto-debug: errors injected", logging.F("attempt", a.debugFails), logging.F("remaining", remaining))
}
// detectBuildCommand returns the appropriate build/test command for the project.
func (a *Agent) detectBuildCommand() string {
	wd := a.cfg.WorkDir
	// Go project.
	if fileExists(filepath.Join(wd, "go.mod")) {
		return "go build ./... 2>&1 && go vet ./... 2>&1"
	}
	// Node.js project — prefer `npm test`, fall back to `npm run build`.
	if fileExists(filepath.Join(wd, "package.json")) {
		return "npm test 2>&1 || npm run build 2>&1"
	}
	// Python project — compile-check first, then pytest.
	if fileExists(filepath.Join(wd, "pyproject.toml")) || fileExists(filepath.Join(wd, "setup.py")) || fileExists(filepath.Join(wd, "requirements.txt")) {
		return "python -m pytest --tb=short 2>&1 || python -m py_compile . 2>&1"
	}
	// Rust project.
	if fileExists(filepath.Join(wd, "Cargo.toml")) {
		return "cargo check 2>&1 && cargo test 2>&1"
	}
	// Java/Maven project.
	if fileExists(filepath.Join(wd, "pom.xml")) {
		return "mvn compile 2>&1 && mvn test 2>&1"
	}
	// Java/Gradle project.
	if fileExists(filepath.Join(wd, "build.gradle")) || fileExists(filepath.Join(wd, "build.gradle.kts")) {
		return "gradle build 2>&1"
	}
	// C# / .NET project — use Glob since *.csproj/*.sln are patterns, not literal files.
	if hasGlobMatch(wd, "*.csproj") || hasGlobMatch(wd, "*.sln") {
		return "dotnet build 2>&1 && dotnet test 2>&1"
	}
	// Ruby project.
	if fileExists(filepath.Join(wd, "Gemfile")) {
		return "bundle exec rake 2>&1"
	}
	// Makefile.
	if fileExists(filepath.Join(wd, "Makefile")) {
		return "make 2>&1"
	}
	return ""
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// hasGlobMatch checks if any file in the directory matches a glob pattern.
func hasGlobMatch(dir, pattern string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		matched, _ := filepath.Match(pattern, e.Name())
		if matched {
			return true
		}
	}
	return false
}

// runQualityScan runs code_quality after file changes and injects findings.
func (a *Agent) runQualityScan(ctx context.Context, emit func(Event)) {
	qt, ok := a.registry.Get("code_quality")
	if !ok {
		return
	}
	emit(Event{Kind: EventToolCall, ToolName: "auto_quality", ToolArgs: a.cfg.WorkDir})
	args := fmt.Sprintf(`{"path": %q}`, a.cfg.WorkDir)
	result, err := qt.Execute(ctx, args)
	if err != nil {
		result = "Error: " + err.Error()
	}
	// Only inject if there are findings worth addressing.
	hasIssues := err != nil ||
		strings.Contains(result, "issue") ||
		strings.Contains(result, "smell") ||
		strings.Contains(result, "vulnerability") ||
		strings.Contains(result, "TODO")
	emit(Event{Kind: EventToolDone, ToolName: "auto_quality", Result: result, IsError: err != nil})
	if hasIssues {
		a.history = append(a.history, llm.Message{
			Role:    llm.RoleUser,
			Content: "[QUALITY CHECK] Code quality scan after your changes produced findings. Address any issues your changes introduced:\n\n" + result,
		})
		a.logger.Info("quality scan: findings injected")
	}
}

// runSecurityScan runs code_quality (security mode) after file changes.
func (a *Agent) runSecurityScan(ctx context.Context, emit func(Event)) {
	qt, ok := a.registry.Get("code_quality")
	if !ok {
		return
	}
	emit(Event{Kind: EventToolCall, ToolName: "auto_security", ToolArgs: a.cfg.WorkDir})
	args := fmt.Sprintf(`{"path": %q}`, a.cfg.WorkDir)
	result, err := qt.Execute(ctx, args)
	if err != nil {
		result = "Error: " + err.Error()
	}
	hasVulns := err != nil ||
		strings.Contains(strings.ToLower(result), "vulnerability") ||
		strings.Contains(strings.ToLower(result), "injection") ||
		strings.Contains(strings.ToLower(result), "xss") ||
		strings.Contains(strings.ToLower(result), "secret")
	emit(Event{Kind: EventToolDone, ToolName: "auto_security", Result: result, IsError: err != nil})
	if hasVulns {
		a.history = append(a.history, llm.Message{
			Role:    llm.RoleUser,
			Content: "[SECURITY SCAN] Security scan after your changes found potential vulnerabilities. Review and fix any issues your changes introduced:\n\n" + result,
		})
		a.logger.Info("security scan: findings injected")
	}
}

// runCIPipeline runs the full CI flow after file changes.
func (a *Agent) runCIPipeline(ctx context.Context, emit func(Event)) {
	ci, ok := a.registry.Get("ci_pipeline")
	if !ok {
		return
	}
	emit(Event{Kind: EventToolCall, ToolName: "auto_ci", ToolArgs: a.cfg.WorkDir})
	args := fmt.Sprintf(`{"path": %q}`, a.cfg.WorkDir)
	result, err := ci.Execute(ctx, args)
	if err != nil {
		result = "Error: " + err.Error()
	}
	failed := err != nil ||
		strings.Contains(strings.ToLower(result), "fail") ||
		strings.Contains(strings.ToLower(result), "error")
	emit(Event{Kind: EventToolDone, ToolName: "auto_ci", Result: result, IsError: failed})
	if failed {
		a.history = append(a.history, llm.Message{
			Role:    llm.RoleUser,
			Content: "[CI PIPELINE] CI check after your changes reported failures. Fix them before declaring the task complete:\n\n" + result,
		})
		a.logger.Info("ci pipeline: failures injected")
	}
}
