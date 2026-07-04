package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cmyolo441-coder/zed/internal/plugin"
)

type PluginExecute struct {
	Manager *plugin.Manager
}

func (t *PluginExecute) Name() string { return "plugin_exec" }
func (t *PluginExecute) Description() string {
	return "Execute a registered plugin by name. " +
		"Use plugin_list first to see available plugins. " +
		"Args: {\"name\": \"plugin name\", \"args\": \"arguments for the plugin\"}"
}
func (t *PluginExecute) RequiresApproval() bool { return true }
func (t *PluginExecute) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string", "description": "Plugin name."},
			"args": map[string]any{"type": "string", "description": "Arguments for the plugin."},
		},
		"required": []string{"name"},
	}
}
func (t *PluginExecute) Execute(ctx context.Context, args string) (string, error) {
	var a struct {
		Name string `json:"name"`
		Args string `json:"args,omitempty"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	return t.Manager.Execute(ctx, a.Name, a.Args)
}

type PluginList struct {
	Manager *plugin.Manager
}

func (t *PluginList) Name() string { return "plugin_list" }
func (t *PluginList) Description() string {
	return "List all registered plugins. " +
		"Args: {} (no arguments)"
}
func (t *PluginList) RequiresApproval() bool { return false }
func (t *PluginList) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{},
	}
}
func (t *PluginList) Execute(_ context.Context, args string) (string, error) {
	var a struct{}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	plugins := t.Manager.List()
	if len(plugins) == 0 {
		return "No plugins registered.", nil
	}
	var b strings.Builder
	b.WriteString("Registered plugins:\n")
	for _, name := range plugins {
		fmt.Fprintf(&b, "  • %s\n", name)
	}
	return b.String(), nil
}

var (
	_ Tool = (*PluginExecute)(nil)
	_ Tool = (*PluginList)(nil)
)
