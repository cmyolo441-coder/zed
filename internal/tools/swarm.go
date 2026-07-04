package tools

import (
	"context"
	"fmt"

	"github.com/cmyolo441-coder/zed/internal/llm"
	"github.com/cmyolo441-coder/zed/internal/swarm"
)

// SpawnSwarm is a tool that lets the agent spawn parallel sub-agents.
type SpawnSwarm struct {
	Client   llm.Client
	Registry *Registry
}

func (t *SpawnSwarm) Name() string { return "spawn_swarm" }
func (t *SpawnSwarm) Description() string {
	return "Spawn multiple sub-agents to work in PARALLEL on sub-tasks. " +
		"Each sub-agent runs its own ReAct loop independently. " +
		"Use for: large tasks that can be decomposed (e.g. research file A + research file B + write tests). " +
		"Args: {\"tasks\": [{\"id\": \"task1\", \"description\": \"what to do\", \"tools\": [\"read_file\",\"grep\"]}, ...]}"
}
func (t *SpawnSwarm) RequiresApproval() bool { return false }
func (t *SpawnSwarm) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tasks": map[string]any{
				"type": "array",
				"description": "Sub-tasks to run in parallel.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":          map[string]any{"type": "string", "description": "Unique task identifier."},
						"description": map[string]any{"type": "string", "description": "What the sub-agent should do."},
						"tools":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Allowed tool names (empty = all)."},
					},
				},
			},
		},
		"required": []string{"tasks"},
	}
}

func (t *SpawnSwarm) Execute(ctx context.Context, args string) (string, error) {
	var a struct {
		Tasks []struct {
			ID          string   `json:"id"`
			Description string   `json:"description"`
			Tools       []string `json:"tools"`
		} `json:"tasks"`
	}
	if err := parseArgs(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if len(a.Tasks) == 0 {
		return "", fmt.Errorf("no tasks provided")
	}

	// Convert to swarm.SubTask.
	tasks := make([]swarm.SubTask, len(a.Tasks))
	for i, t := range a.Tasks {
		tasks[i] = swarm.SubTask{
			ID:          t.ID,
			Description: t.Description,
			Tools:       t.Tools,
		}
	}

	// Create and run the swarm.
	adapter := &registryAdapter{r: t.Registry}
	s := swarm.New(t.Client, adapter, nil)
	results := s.Run(ctx, tasks, "You are a focused sub-agent working on part of a larger task.")
	return swarm.MergeResults(results), nil
}

var _ Tool = (*SpawnSwarm)(nil)

// registryAdapter wraps the Registry to satisfy swarm.ToolExecutor.
type registryAdapter struct {
	r *Registry
}

func (a *registryAdapter) ExecTool(ctx context.Context, call llm.ToolCall) (string, error) {
	res := a.r.Execute(ctx, call)
	if res.IsError {
		return res.Content, fmt.Errorf("%s", res.Content)
	}
	return res.Content, nil
}

func (a *registryAdapter) HasTool(name string) bool {
	return a.r.HasTool(name)
}
