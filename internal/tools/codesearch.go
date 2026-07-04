package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/gjkjk/zed/internal/index"
)

// CodeSearch exposes the codebase index to the agent for relevance-ranked file
// discovery and symbol lookup. It complements the raw grep/find tools with
// TF-IDF ranking and symbol awareness.
type CodeSearch struct {
	Index *index.Index
}

func (t *CodeSearch) Name() string { return "code_search" }
func (t *CodeSearch) Description() string {
	return "Search the indexed codebase for the files most relevant to a query, ranked by relevance. Returns file paths with matching symbols. Use to quickly locate where functionality lives."
}
func (t *CodeSearch) RequiresApproval() bool { return false }
func (t *CodeSearch) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "Natural-language or keyword query."},
			"limit": map[string]any{"type": "integer", "description": "Max files to return (default 8)."},
		},
		"required": []string{"query"},
	}
}

func (t *CodeSearch) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.Query == "" {
		return "", fmt.Errorf("empty query")
	}
	if a.Limit <= 0 {
		a.Limit = 8
	}
	results := t.Index.Search(a.Query, a.Limit)
	if len(results) == 0 {
		return "No relevant files found. The index may be empty; the agent can still use grep/find.", nil
	}
	var b strings.Builder
	for _, r := range results {
		fmt.Fprintf(&b, "%s  (score %.2f)\n", r.Path, r.Score)
		shown := 0
		for _, sym := range r.Symbols {
			fmt.Fprintf(&b, "    %s %s:%d\n", sym.Kind, sym.Name, sym.Line)
			shown++
			if shown >= 5 {
				break
			}
		}
	}
	return b.String(), nil
}

// SymbolLookup finds the definition location of a named symbol.
type SymbolLookup struct {
	Index *index.Index
}

func (t *SymbolLookup) Name() string { return "find_symbol" }
func (t *SymbolLookup) Description() string {
	return "Find where a function, type, or class is defined by name. Returns file:line locations."
}
func (t *SymbolLookup) RequiresApproval() bool { return false }
func (t *SymbolLookup) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string", "description": "Symbol name to locate."},
		},
		"required": []string{"name"},
	}
}

func (t *SymbolLookup) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Name string `json:"name"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	syms := t.Index.FindSymbol(a.Name)
	if len(syms) == 0 {
		return fmt.Sprintf("No symbol named %q found in the index.", a.Name), nil
	}
	var b strings.Builder
	for _, s := range syms {
		fmt.Fprintf(&b, "%s %s  %s:%d\n", s.Kind, s.Name, s.File, s.Line)
	}
	return b.String(), nil
}
