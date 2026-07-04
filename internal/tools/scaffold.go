package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cmyolo441-coder/zed/internal/scaffold"
)

// Scaffold is a tool that generates code from templates.
type Scaffold struct {
	WorkDir string
}

func (t *Scaffold) Name() string { return "scaffold" }
func (t *Scaffold) Description() string {
	return "Generate code from templates. Rapid scaffolding for: " +
		"REST APIs, CRUD models, test files, Python CLIs, Dockerfiles, GitHub Actions CI. " +
		"Args: {\"template\": \"template id\", \"params\": {\"name\": \"...\", \"model\": \"...\", \"language\": \"go\"}}"
}
func (t *Scaffold) RequiresApproval() bool { return false }
func (t *Scaffold) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"template": map[string]any{"type": "string", "description": "Template ID (use \"list\" to see all)."},
			"params":   map[string]any{"type": "object", "description": "Template parameters (name, model, language, etc.)"},
		},
		"required": []string{"template"},
	}
}

func (t *Scaffold) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Template string            `json:"template"`
		Params   map[string]string `json:"params"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.Template == "" {
		return "", fmt.Errorf("template is required")
	}

	registry := scaffold.New()

	if a.Template == "list" {
		var b strings.Builder
		b.WriteString("📋 Available Templates:\n\n")
		for _, id := range registry.List() {
			tmpl, _ := registry.Get(id)
			fmt.Fprintf(&b, "  %s: %s (%s)\n", id, tmpl.Name, tmpl.Language)
			fmt.Fprintf(&b, "    %s\n", tmpl.Description)
		}
		return b.String(), nil
	}

	code, err := registry.Generate(a.Template, a.Params)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("✅ Generated code from template %q:\n\n%s", a.Template, code), nil
}

var _ Tool = (*Scaffold)(nil)
