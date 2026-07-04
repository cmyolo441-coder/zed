package tools

import "context"

// Tool is any capability the agent can invoke. Each tool exposes a JSON schema
// so the LLM knows how to call it, and an Execute method that runs it.
type Tool interface {
	// Name is the unique identifier the model uses to call this tool.
	Name() string
	// Description tells the model what the tool does and when to use it.
	Description() string
	// Schema returns the JSON Schema for the tool's arguments.
	Schema() map[string]any
	// RequiresApproval reports whether execution must be confirmed by the user.
	RequiresApproval() bool
	// Execute runs the tool with raw JSON args and returns a text result.
	Execute(ctx context.Context, args string) (string, error)
}

// Result is the outcome of a tool execution, ready to feed back to the model.
type Result struct {
	CallID  string
	Name    string
	Content string
	IsError bool
}
