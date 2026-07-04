package llm

// Role identifies who authored a message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is a single turn in a conversation. It can carry plain text, a tool
// call requested by the assistant, or the result of a tool execution.
type Message struct {
	Role      Role       `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// For RoleTool messages: which call this result answers.
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
}

// ToolCall is a request from the model to invoke a named tool with JSON args.
type ToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Args string `json:"arguments"` // raw JSON object
}

// ToolSchema describes a tool the model may call.
type ToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema object
}

// Request is a completion request sent to a provider.
type Request struct {
	Model     string
	System    string
	Messages  []Message
	Tools     []ToolSchema
	MaxTokens int
	Stream    bool
}

// StreamEvent is emitted while a response streams in.
type StreamEvent struct {
	// One of the following is set per event.
	TextDelta string    // incremental assistant text
	ToolCall  *ToolCall // a completed tool call
	Done      bool      // stream finished
	Err       error     // fatal error
	Usage     *Usage    // token usage, on Done
	// Truncated is set when the model stopped because it hit the output token
	// limit (finish_reason == "length"). When true on a ToolCall, its Args are
	// likely incomplete JSON.
	Truncated bool
}

// Usage reports token consumption for a turn.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// Response is a non-streaming completion response.
type Response struct {
	Content   string
	ToolCalls []ToolCall
	Usage     *Usage
}
