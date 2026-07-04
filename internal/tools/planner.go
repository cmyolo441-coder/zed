package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cmyolo441-coder/zed/internal/planner"
)

// DecomposeTask is a tool that helps the agent break complex goals into
// a structured task tree with dependencies and progress tracking.
type DecomposeTask struct{}

func (t *DecomposeTask) Name() string { return "decompose_task" }
func (t *DecomposeTask) Description() string {
	return "Break a complex goal into a structured task tree with dependencies. " +
		"Each task has an ID, title, description, dependencies, and files to touch. " +
		"Returns a visual plan with progress tracking. " +
		"Args: {\"goal\": \"the overall goal\", \"tasks\": [{\"id\": \"1\", \"title\": \"...\", \"description\": \"...\", \"dependencies\": [], \"files\": []}]}"
}
func (t *DecomposeTask) RequiresApproval() bool { return false }
func (t *DecomposeTask) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"goal":  map[string]any{"type": "string", "description": "The overall goal to decompose."},
			"tasks": map[string]any{
				"type": "array",
				"description": "Sub-tasks with dependencies.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":           map[string]any{"type": "string"},
						"title":        map[string]any{"type": "string"},
						"description":  map[string]any{"type": "string"},
						"dependencies": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"files":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					},
				},
			},
		},
		"required": []string{"goal", "tasks"},
	}
}

func (t *DecomposeTask) Execute(_ context.Context, args string) (string, error) {
	var a struct {
		Goal  string `json:"goal"`
		Tasks []struct {
			ID           string   `json:"id"`
			Title        string   `json:"title"`
			Description  string   `json:"description"`
			Dependencies []string `json:"dependencies"`
			Files        []string `json:"files"`
		} `json:"tasks"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.Goal == "" || len(a.Tasks) == 0 {
		return "", fmt.Errorf("goal and tasks are required")
	}

	// Build the task tree.
	root := &planner.Task{
		ID:          "0",
		Title:       a.Goal,
		Description: "Root goal",
	}
	for _, t := range a.Tasks {
		root.SubTasks = append(root.SubTasks, &planner.Task{
			ID:           t.ID,
			Title:        t.Title,
			Description:  t.Description,
			Dependencies: t.Dependencies,
			Files:        t.Files,
		})
	}

	plan := planner.NewPlan(a.Goal, root)
	return plan.Render(), nil
}

var _ Tool = (*DecomposeTask)(nil)

// formatTasksForPrompt returns a formatted string of tasks for the agent.
func formatTasksForPrompt(tasks []*planner.Task) string {
	var b strings.Builder
	for _, t := range tasks {
		fmt.Fprintf(&b, "  [%s] %s: %s\n", t.ID, t.Title, t.Description)
		if len(t.Dependencies) > 0 {
			fmt.Fprintf(&b, "    deps: %s\n", strings.Join(t.Dependencies, ", "))
		}
		if len(t.Files) > 0 {
			fmt.Fprintf(&b, "    files: %s\n", strings.Join(t.Files, ", "))
		}
	}
	return b.String()
}
