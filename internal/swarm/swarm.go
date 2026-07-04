// Package swarm implements multi-agent orchestration. A main agent can spawn
// sub-agents that work in parallel on sub-tasks (research, coding, testing),
// then merge their results. This enables complex, multi-faceted work to be
// decomposed and executed concurrently for dramatic speedups.
package swarm

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/gjkjk/zed/internal/llm"
)

// SubTask is a unit of work assigned to a sub-agent.
type SubTask struct {
	ID          string
	Description string   // what the sub-agent should do
	Tools       []string // tool names this sub-agent is allowed to use (empty = all)
	Model       string   // model override (empty = use parent's model)
}

// SubResult is the outcome of a sub-agent's work.
type SubResult struct {
	TaskID    string
	Success   bool
	Output    string // the sub-agent's final answer
	Error     string
	ToolCalls int    // how many tool calls the sub-agent made
}

// Swarm coordinates parallel sub-agents. Each sub-agent runs its own ReAct
// loop independently, and results are collected and merged.
type Swarm struct {
	client   llm.Client
	registry ToolExecutor
	logger   func(string)
}

// ToolExecutor is a minimal interface for executing tool calls (the agent's
// registry satisfies this via an adapter).
type ToolExecutor interface {
	ExecTool(ctx context.Context, call llm.ToolCall) (string, error)
	HasTool(name string) bool
}

// New creates a swarm that can spawn sub-agents.
func New(client llm.Client, registry ToolExecutor, logger func(string)) *Swarm {
	if logger == nil {
		logger = func(string) {}
	}
	return &Swarm{client: client, registry: registry, logger: logger}
}

// Run executes sub-tasks in parallel and returns all results.
// Each sub-agent runs a mini ReAct loop: it gets its task, can call tools,
// and produces a final answer. Results are collected in order.
func (s *Swarm) Run(ctx context.Context, tasks []SubTask, systemPrompt string) []SubResult {
	results := make([]SubResult, len(tasks))
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t SubTask) {
			defer wg.Done()
			results[idx] = s.runSubAgent(ctx, t, systemPrompt)
		}(i, task)
	}
	wg.Wait()
	return results
}

// runSubAgent runs a single sub-agent's ReAct loop.
func (s *Swarm) runSubAgent(ctx context.Context, task SubTask, baseSystem string) SubResult {
	result := SubResult{TaskID: task.ID}

	// Build the sub-agent's system prompt.
	system := baseSystem
	if system == "" {
		system = "You are a focused sub-agent. Complete your assigned task efficiently."
	}
	system += fmt.Sprintf("\n\nYour specific task: %s\nWork autonomously. Use tools as needed. When done, provide a clear summary of your findings/work.", task.Description)

	// Build the conversation.
	history := []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: task.Description},
	}

	model := task.Model
	if model == "" {
		model = "mimo-v2.5-free" // fast default for sub-agents
	}

	// Mini ReAct loop — max 15 steps per sub-agent.
	for step := 0; step < 15; step++ {
		req := llm.Request{
			Model:    model,
			Messages: history,
			MaxTokens: 16000,
			Stream:   false,
		}

		resp, err := s.client.Complete(ctx, req)
		if err != nil {
			result.Error = fmt.Sprintf("LLM error: %v", err)
			return result
		}

		history = append(history, llm.Message{
			Role:    llm.RoleAssistant,
			Content: resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// No tool calls → sub-agent is done.
		if len(resp.ToolCalls) == 0 {
			result.Success = true
			result.Output = resp.Content
			result.ToolCalls = step
			return result
		}

		// Execute tool calls.
		for _, call := range resp.ToolCalls {
			// Check if this sub-agent is allowed to use this tool.
			if len(task.Tools) > 0 && !contains(task.Tools, call.Name) {
				history = append(history, llm.Message{
					Role:       llm.RoleTool,
					ToolCallID: call.ID,
					ToolName:   call.Name,
					Content:    "Error: this tool is not available for this sub-task.",
				})
				continue
			}
			if !s.registry.HasTool(call.Name) {
				history = append(history, llm.Message{
					Role:       llm.RoleTool,
					ToolCallID: call.ID,
					ToolName:   call.Name,
					Content:    "Error: unknown tool.",
				})
				continue
			}
			out, err := s.registry.ExecTool(ctx, call)
			if err != nil {
				out = "Error: " + err.Error()
			}
			result.ToolCalls++
			history = append(history, llm.Message{
				Role:       llm.RoleTool,
				ToolCallID: call.ID,
				ToolName:   call.Name,
				Content:    out,
			})
		}
	}

	// Max steps reached — return what we have.
	result.Success = true
	result.Output = "Sub-agent reached max steps. Partial work:\n" + lastAssistantText(history)
	result.ToolCalls = 15
	return result
}

// MergeResults combines sub-agent results into a single summary.
func MergeResults(results []SubResult) string {
	if len(results) == 0 {
		return "No sub-tasks were run."
	}
	var b strings.Builder
	b.WriteString("=== Multi-Agent Results ===\n\n")
	successCount := 0
	for _, r := range results {
		status := "✓"
		if !r.Success {
			status = "✗"
		} else {
			successCount++
		}
		fmt.Fprintf(&b, "%s Task %s (%d tool calls):\n", status, r.TaskID, r.ToolCalls)
		if r.Error != "" {
			fmt.Fprintf(&b, "   Error: %s\n", r.Error)
		}
		if r.Output != "" {
			// Indent the output.
			for _, line := range strings.Split(r.Output, "\n") {
				fmt.Fprintf(&b, "   %s\n", line)
			}
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "Summary: %d/%d tasks completed successfully.\n", successCount, len(results))
	return b.String()
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func lastAssistantText(msgs []llm.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == llm.RoleAssistant && msgs[i].Content != "" {
			return msgs[i].Content
		}
	}
	return ""
}
