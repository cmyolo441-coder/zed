package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cmyolo441-coder/zed/internal/memory"
)

// MemoryRemember is a tool to store persistent memories.
type MemoryRemember struct {
	Store *memory.Store
}

func (t *MemoryRemember) Name() string { return "remember" }
func (t *MemoryRemember) Description() string {
	return "Store a persistent memory that survives across sessions. " +
		"Use for: codebase patterns, user preferences, project decisions, important facts. " +
		"Args: {\"category\": \"fact|preference|pattern|decision\", \"key\": \"short name\", \"value\": \"the memory\", \"tags\": [\"optional\"]}"
}
func (t *MemoryRemember) RequiresApproval() bool { return false }
func (t *MemoryRemember) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"category": map[string]any{"type": "string", "description": "fact, preference, pattern, or decision"},
			"key":      map[string]any{"type": "string", "description": "Short identifier for this memory."},
			"value":    map[string]any{"type": "string", "description": "The memory content to store."},
			"tags":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional searchable tags."},
		},
		"required": []string{"category", "key", "value"},
	}
}

func (t *MemoryRemember) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Category string   `json:"category"`
		Key      string   `json:"key"`
		Value    string   `json:"value"`
		Tags     []string `json:"tags"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.Category == "" || a.Key == "" || a.Value == "" {
		return "", fmt.Errorf("category, key, and value are required")
	}
	if err := t.Store.Remember(a.Category, a.Key, a.Value, a.Tags...); err != nil {
		return "", fmt.Errorf("failed to save memory: %w", err)
	}
	return fmt.Sprintf("✓ Remembered [%s] %s: %s", a.Category, a.Key, a.Value), nil
}

// MemoryRecall is a tool to retrieve persistent memories.
type MemoryRecall struct {
	Store *memory.Store
}

func (t *MemoryRecall) Name() string { return "recall" }
func (t *MemoryRecall) Description() string {
	return "Recall a persistent memory from past sessions. " +
		"Use to check if you already know something about the project or user. " +
		"Args: {\"query\": \"search terms\", \"category\": \"optional category filter\"}"
}
func (t *MemoryRecall) RequiresApproval() bool { return false }
func (t *MemoryRecall) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query":    map[string]any{"type": "string", "description": "Search query for memories."},
			"category": map[string]any{"type": "string", "description": "Optional: filter by category."},
		},
		"required": []string{"query"},
	}
}

func (t *MemoryRecall) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Query    string `json:"query"`
		Category string `json:"category"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.Query == "" {
		return "", fmt.Errorf("query is required")
	}
	var results []*memory.Entry
	if a.Category != "" {
		all := t.Store.RecallCategory(a.Category)
		// Filter by query within category.
		q := strings.ToLower(a.Query)
		for _, e := range all {
			if strings.Contains(strings.ToLower(e.Key), q) ||
				strings.Contains(strings.ToLower(e.Value), q) {
				results = append(results, e)
			}
		}
	} else {
		results = t.Store.Search(a.Query, 10)
	}
	if len(results) == 0 {
		return "No memories found for: " + a.Query, nil
	}
	var b strings.Builder
	b.WriteString("🧠 Memories:\n\n")
	for _, e := range results {
		fmt.Fprintf(&b, "  [%s] %s: %s\n", e.Category, e.Key, e.Value)
		if len(e.Tags) > 0 {
			fmt.Fprintf(&b, "    tags: %s\n", strings.Join(e.Tags, ", "))
		}
	}
	return b.String(), nil
}

var _ Tool = (*MemoryRemember)(nil)
var _ Tool = (*MemoryRecall)(nil)
