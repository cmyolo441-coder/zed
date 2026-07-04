package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/gjkjk/zed/internal/diagram"
	"github.com/gjkjk/zed/internal/fuzz"
	"github.com/gjkjk/zed/internal/meta"
	"github.com/gjkjk/zed/internal/reason"
	"github.com/gjkjk/zed/internal/selftest"
	"github.com/gjkjk/zed/internal/vcs"
)

// --- Self-Analysis Tool ---

type SelfAnalyze struct{ Profile *meta.Profile }

func (t *SelfAnalyze) Name() string { return "self_analyze" }
func (t *SelfAnalyze) Description() string {
	return "Analyze the agent's own performance — find bottlenecks, failures, and improvement opportunities. " +
		"Returns suggestions for self-optimization."
}
func (t *SelfAnalyze) RequiresApproval() bool { return false }
func (t *SelfAnalyze) Schema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
func (t *SelfAnalyze) Execute(_ context.Context, _ string) (string, error) {
	return t.Profile.Report(), nil
}

// --- Transparent Decision Analysis Tool ---

type ReasonTool struct{}

func (t *ReasonTool) Name() string { return "reason" }
func (t *ReasonTool) Description() string {
	return "Explore multiple transparent decision paths before making a decision. " +
		"Generates alternative approaches, evaluates risks, and picks the best path. " +
		"Args: {\"decision\": \"what to decide\", \"options\": [\"option1\", \"option2\"]}"
}
func (t *ReasonTool) RequiresApproval() bool { return false }
func (t *ReasonTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"decision": map[string]any{"type": "string", "description": "The decision to reason about."},
			"options":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Options to evaluate."},
		},
		"required": []string{"decision", "options"},
	}
}
func (t *ReasonTool) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Decision string   `json:"decision"`
		Options  []string `json:"options"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", err
	}
	if a.Decision == "" || len(a.Options) == 0 {
		return "", fmt.Errorf("decision and options required")
	}
	engine := reason.New()
	paths := engine.Explore(a.Decision, a.Options)
	return reason.RenderAll(paths), nil
}

// --- Git VCS Tool ---

type GitTool struct{ WorkDir string }

func (t *GitTool) Name() string { return "git" }
func (t *GitTool) Description() string {
	return "Git version control automation: status, auto-commit with semantic messages, branch creation, diff, log. " +
		"Args: {\"action\": \"status|commit|branch|diff|log\", \"message\": \"commit message\", \"branch\": \"branch name\"}"
}
func (t *GitTool) RequiresApproval() bool { return true }
func (t *GitTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action":  map[string]any{"type": "string", "description": "status, commit, branch, diff, log"},
			"message": map[string]any{"type": "string", "description": "Commit message (for commit action)."},
			"branch":  map[string]any{"type": "string", "description": "Branch name (for branch action)."},
		},
		"required": []string{"action"},
	}
}
func (t *GitTool) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Action  string `json:"action"`
		Message string `json:"message"`
		Branch  string `json:"branch"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", err
	}
	switch a.Action {
	case "status":
		info, err := vcs.Status(t.WorkDir)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		fmt.Fprintf(&b, "🌿 Git Status\n  Branch: %s\n", info.Branch)
		if len(info.Staged) > 0 {
			b.WriteString("  Staged:\n")
			for _, f := range info.Staged {
				fmt.Fprintf(&b, "    + %s\n", f)
			}
		}
		if len(info.Modified) > 0 {
			b.WriteString("  Modified:\n")
			for _, f := range info.Modified {
				fmt.Fprintf(&b, "    M %s\n", f)
			}
		}
		if len(info.Untracked) > 0 {
			b.WriteString("  Untracked:\n")
			for _, f := range info.Untracked {
				fmt.Fprintf(&b, "    ? %s\n", f)
			}
		}
		return b.String(), nil
	case "commit":
		msg := a.Message
		if msg == "" {
			info, _ := vcs.Status(t.WorkDir)
			msg = vcs.SemanticMessage(info)
		}
		return vcs.AutoCommit(t.WorkDir, msg)
	case "branch":
		if a.Branch == "" {
			return "", fmt.Errorf("branch name required")
		}
		return vcs.CreateBranch(t.WorkDir, a.Branch)
	case "diff":
		return vcs.Diff(t.WorkDir)
	case "log":
		return vcs.Log(t.WorkDir, 10)
	default:
		return "", fmt.Errorf("unknown action: %s", a.Action)
	}
}

// --- Fuzz Testing Tool ---

type FuzzTool struct{}

func (t *FuzzTool) Name() string { return "fuzz_test" }
func (t *FuzzTool) Description() string {
	return "Generate fuzz tests for a function. Creates property-based tests that " +
		"discover edge cases humans miss. " +
		"Args: {\"function\": \"function name\", \"language\": \"go|python\", \"param_type\": \"int|string|float\"}"
}
func (t *FuzzTool) RequiresApproval() bool { return false }
func (t *FuzzTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"function":   map[string]any{"type": "string", "description": "Function to fuzz test."},
			"language":   map[string]any{"type": "string", "description": "go or python"},
			"param_type": map[string]any{"type": "string", "description": "int, string, float, bool"},
		},
		"required": []string{"function", "language", "param_type"},
	}
}
func (t *FuzzTool) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Function  string `json:"function"`
		Language  string `json:"language"`
		ParamType string `json:"param_type"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", err
	}
	var test *fuzz.FuzzTest
	switch a.Language {
	case "go":
		test = fuzz.GenerateGoFuzz(a.Function, a.ParamType)
	case "python":
		test = fuzz.GeneratePythonFuzz(a.Function, a.ParamType)
	default:
		return "", fmt.Errorf("unsupported language: %s", a.Language)
	}
	return fmt.Sprintf("🧬 Fuzz test for %s:\n\nProperty: %s\n\n%s", a.Function, test.Property, test.Code), nil
}

// --- Architecture Diagram Tool ---

type DiagramTool struct{ WorkDir string }

func (t *DiagramTool) Name() string { return "diagram" }
func (t *DiagramTool) Description() string {
	return "Generate architecture diagrams (Mermaid format) from code. " +
		"Creates dependency graphs, call graphs, and class relationship diagrams. " +
		"Args: {\"type\": \"dependency|call_graph|class\", \"data\": {\"file\": [\"imports\"]}}"
}
func (t *DiagramTool) RequiresApproval() bool { return false }
func (t *DiagramTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"type": map[string]any{"type": "string", "description": "dependency, call_graph, or class"},
			"data": map[string]any{"type": "object", "description": "Map of node → [related nodes]"},
		},
		"required": []string{"type", "data"},
	}
}
func (t *DiagramTool) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Type string              `json:"type"`
		Data map[string][]string `json:"data"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", err
	}
	var d *diagram.Diagram
	switch a.Type {
	case "dependency":
		d = diagram.BuildDependencyDiagram(a.Data)
	case "call_graph":
		d = diagram.BuildCallGraph(a.Data)
	case "class":
		d = diagram.BuildClassDiagram(a.Data)
	default:
		return "", fmt.Errorf("unknown diagram type: %s", a.Type)
	}
	return fmt.Sprintf("```mermaid\n%s```\n", d.RenderMermaid()), nil
}

// --- Self-Testing Code Gen Tool ---

type SelfTestTool struct{}

func (t *SelfTestTool) Name() string { return "generate_tests" }
func (t *SelfTestTool) Description() string {
	return "Generate an honest comprehensive test plan and, when concrete signature/cases are supplied, runnable tests with real assertions. " +
		"It never emits skipped empty tests. " +
		"Args: {\"function\": \"function name\", \"language\": \"go|python\", \"coverage_goal\": 90, \"package\": \"pkg\", \"param_type\": \"string\", \"return_type\": \"string\", \"cases\": [{\"name\":\"basic\",\"input\":\"\\\"a\\\"\",\"expected\":\"\\\"A\\\"\"}]}"
}
func (t *SelfTestTool) RequiresApproval() bool { return false }
func (t *SelfTestTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"function":      map[string]any{"type": "string"},
			"language":      map[string]any{"type": "string"},
			"coverage_goal": map[string]any{"type": "number"},
			"package":       map[string]any{"type": "string", "description": "Go package name for runnable generated tests."},
			"param_type":    map[string]any{"type": "string", "description": "Single-argument parameter type for generated tests."},
			"return_type":   map[string]any{"type": "string", "description": "Return type for generated tests."},
			"cases": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{
				"name":        map[string]any{"type": "string"},
				"description": map[string]any{"type": "string"},
				"input":       map[string]any{"type": "string"},
				"expected":    map[string]any{"type": "string"},
				"type":        map[string]any{"type": "string"},
			}}},
		},
		"required": []string{"function", "language"},
	}
}
func (t *SelfTestTool) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Function     string              `json:"function"`
		Language     string              `json:"language"`
		CoverageGoal float64             `json:"coverage_goal"`
		Package      string              `json:"package"`
		ParamType    string              `json:"param_type"`
		ReturnType   string              `json:"return_type"`
		Cases        []selftest.TestCase `json:"cases"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", err
	}
	if a.CoverageGoal == 0 {
		a.CoverageGoal = 90
	}
	plan := selftest.Generate(a.Function, a.Language, a.CoverageGoal).WithExecutableDetails(a.Package, a.ParamType, a.ReturnType, a.Cases)
	return plan.Summary() + "\n\n" + plan.Render(), nil
}

var (
	_ Tool = (*SelfAnalyze)(nil)
	_ Tool = (*ReasonTool)(nil)
	_ Tool = (*GitTool)(nil)
	_ Tool = (*FuzzTool)(nil)
	_ Tool = (*DiagramTool)(nil)
	_ Tool = (*SelfTestTool)(nil)
)
