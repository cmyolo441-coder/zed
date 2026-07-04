package tools

import (
	"context"
	"fmt"

	"github.com/gjkjk/zed/internal/llm"
)

// Registry holds all available tools and dispatches calls to them.
type Registry struct {
	tools map[string]Tool
	order []string
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

// Register adds a tool. Later registration of the same name overwrites.
func (r *Registry) Register(t Tool) {
	if _, exists := r.tools[t.Name()]; !exists {
		r.order = append(r.order, t.Name())
	}
	r.tools[t.Name()] = t
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// HasTool reports whether a tool with the given name is registered.
func (r *Registry) HasTool(name string) bool {
	_, ok := r.tools[name]
	return ok
}

// Schemas returns the LLM-facing schema for every registered tool.
func (r *Registry) Schemas() []llm.ToolSchema {
	out := make([]llm.ToolSchema, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		out = append(out, llm.ToolSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Schema(),
		})
	}
	return out
}

// Execute dispatches a tool call and returns a Result.
func (r *Registry) Execute(ctx context.Context, call llm.ToolCall) Result {
	t, ok := r.tools[call.Name]
	if !ok {
		return Result{CallID: call.ID, Name: call.Name, IsError: true,
			Content: fmt.Sprintf("unknown tool: %s", call.Name)}
	}
	content, err := t.Execute(ctx, call.Args)
	if err != nil {
		return Result{CallID: call.ID, Name: call.Name, IsError: true,
			Content: "Error: " + err.Error()}
	}
	return Result{CallID: call.ID, Name: call.Name, Content: content}
}
