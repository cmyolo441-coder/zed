package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cmyolo441-coder/zed/internal/index"
)

// SemanticSearch is a concept-aware code search tool.
// It expands queries with synonyms (e.g. "authentication" → "login", "auth",
// "session", "token") and ranks files by semantic relevance.
type SemanticSearch struct {
	Index *index.Index
}

func (t *SemanticSearch) Name() string { return "semantic_search" }
func (t *SemanticSearch) Description() string {
	return "Semantic code search — finds files by concept, not just keywords. " +
		"Understands synonyms: 'authentication' matches 'login', 'auth', 'session', 'token'. " +
		"Use for: 'find authentication logic', 'where is the database layer', 'show me error handling'. " +
		"Args: {\"query\": \"natural language concept\", \"limit\": 8}"
}
func (t *SemanticSearch) RequiresApproval() bool { return false }
func (t *SemanticSearch) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "Natural-language concept to search for."},
			"limit": map[string]any{"type": "integer", "description": "Max files to return (default 8)."},
		},
		"required": []string{"query"},
	}
}

func (t *SemanticSearch) Execute(_ context.Context, args string) (string, error) {
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
	results := t.Index.SemanticSearch(a.Query, a.Limit)
	if len(results) == 0 {
		return "No semantically relevant files found. Try a different concept or use grep for exact text.", nil
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("🧠 Semantic search: %q\n\n", a.Query))
	for _, r := range results {
		fmt.Fprintf(&b, "  %s  (relevance: %.2f)\n", r.Path, r.Score)
		shown := 0
		for _, sym := range r.Symbols {
			fmt.Fprintf(&b, "      %s %s (line %d)\n", sym.Kind, sym.Name, sym.Line)
			shown++
			if shown >= 5 {
				break
			}
		}
		b.WriteString("\n")
	}
	return b.String(), nil
}

var _ Tool = (*SemanticSearch)(nil)
