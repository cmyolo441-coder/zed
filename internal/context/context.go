// Package context manages the conversation window sent to the LLM. It estimates
// token usage, trims or compacts old history when the budget is exceeded, and
// preserves the most relevant recent turns so long sessions stay coherent
// without exceeding model context limits.
package context

import (
	"strings"

	"github.com/cmyolo441-coder/zed/internal/llm"
)

// Budget describes the token limits for a conversation window.
type Budget struct {
	MaxContextTokens int // hard ceiling for the whole request
	ReserveOutput    int // tokens to leave free for the model's reply
	CompactThreshold float64 // fraction (0..1) of budget that triggers compaction
}

// DefaultBudget returns a conservative budget suitable for mid-size models.
func DefaultBudget() Budget {
	return Budget{
		MaxContextTokens: 128000,
		ReserveOutput:    8000,
		CompactThreshold: 0.75,
	}
}

// BudgetForModel returns a budget sized to the model's context window.
func BudgetForModel(maxTokens int) Budget {
	if maxTokens <= 0 {
		maxTokens = 128000
	}
	reserve := 8000
	if maxTokens >= 500000 {
		reserve = 32000
	} else if maxTokens >= 100000 {
		reserve = 16000
	}
	return Budget{
		MaxContextTokens: maxTokens,
		ReserveOutput:    reserve,
		CompactThreshold: 0.75,
	}
}

// EstimateTokens approximates the token count of a string. Real tokenizers are
// model-specific; this uses the well-known ~4 chars/token heuristic plus a
// per-word correction, which is accurate enough for budgeting.
func EstimateTokens(s string) int {
	if s == "" {
		return 0
	}
	chars := len(s)
	words := len(strings.Fields(s))
	// Blend the two estimates.
	byChars := chars / 4
	byWords := (words * 4) / 3
	est := (byChars + byWords) / 2
	if est < 1 {
		est = 1
	}
	return est
}

// EstimateMessageTokens estimates the cost of a single message including its
// tool calls and results, plus a small per-message overhead.
func EstimateMessageTokens(m llm.Message) int {
	total := EstimateTokens(m.Content) + 4 // role/formatting overhead
	for _, tc := range m.ToolCalls {
		total += EstimateTokens(tc.Name) + EstimateTokens(tc.Args) + 8
	}
	return total
}

// EstimateConversation sums the estimated tokens for a slice of messages.
func EstimateConversation(system string, msgs []llm.Message) int {
	total := EstimateTokens(system)
	for _, m := range msgs {
		total += EstimateMessageTokens(m)
	}
	return total
}

// Summarizer produces a compact text summary of a set of messages. The agent
// supplies an implementation backed by the LLM; a trivial fallback is provided.
type Summarizer interface {
	Summarize(messages []llm.Message) (string, error)
}

// Manager applies budgeting and compaction to a conversation.
type Manager struct {
	budget     Budget
	summarizer Summarizer
	system     string
}

// NewManager builds a context manager.
func NewManager(system string, budget Budget, s Summarizer) *Manager {
	return &Manager{budget: budget, summarizer: s, system: system}
}

// Available returns the number of tokens left for input after reserving output.
func (m *Manager) Available() int {
	return m.budget.MaxContextTokens - m.budget.ReserveOutput
}

// NeedsCompaction reports whether the conversation exceeds the compaction
// threshold and should be shrunk before the next request.
func (m *Manager) NeedsCompaction(msgs []llm.Message) bool {
	used := EstimateConversation(m.system, msgs)
	limit := float64(m.Available()) * m.budget.CompactThreshold
	return float64(used) > limit
}

// Fit ensures the conversation fits within the available budget. It first tries
// to compact older turns into a summary; if that is unavailable, it falls back
// to dropping the oldest turns while always keeping the latest ones intact.
func (m *Manager) Fit(msgs []llm.Message) ([]llm.Message, bool, error) {
	if EstimateConversation(m.system, msgs) <= m.Available() {
		return msgs, false, nil
	}

	// Keep the most recent turns verbatim; summarize the rest.
	keep := m.recentToKeep(msgs)
	head := msgs[:len(msgs)-keep]
	tail := msgs[len(msgs)-keep:]

	if len(head) == 0 {
		// Even the tail is too big; hard-trim from the front of the tail.
		return m.hardTrim(tail), true, nil
	}

	if m.summarizer != nil {
		summary, err := m.summarizer.Summarize(head)
		if err == nil && strings.TrimSpace(summary) != "" {
			compacted := []llm.Message{{
				Role:    llm.RoleUser,
				Content: "[Earlier conversation summary]\n" + summary,
			}}
			compacted = append(compacted, tail...)
			if EstimateConversation(m.system, compacted) <= m.Available() {
				return compacted, true, nil
			}
			return m.hardTrim(compacted), true, nil
		}
	}

	// Fallback: drop oldest head entirely.
	return m.hardTrim(tail), true, nil
}

// recentToKeep decides how many trailing messages to preserve verbatim.
// Smarter: keeps more messages for higher budgets, preserves tool results
// (they're expensive to regenerate), and always keeps the last exchange.
func (m *Manager) recentToKeep(msgs []llm.Message) int {
	keep := 6
	// Scale with budget — larger contexts keep more history.
	if m.budget.MaxContextTokens >= 500000 {
		keep = 20
	} else if m.budget.MaxContextTokens >= 100000 {
		keep = 10
	}
	// Never drop tool results in the tail — they're expensive to regenerate.
	for i := len(msgs) - 1; i >= 0 && i >= len(msgs)-keep-4; i-- {
		if msgs[i].Role == llm.RoleTool {
			keep = max(keep, len(msgs)-i)
		}
	}
	if keep > len(msgs) {
		keep = len(msgs)
	}
	return keep
}

// hardTrim removes oldest messages until the conversation fits.
func (m *Manager) hardTrim(msgs []llm.Message) []llm.Message {
	for len(msgs) > 1 && EstimateConversation(m.system, msgs) > m.Available() {
		msgs = msgs[1:]
	}
	// If a single giant message remains, truncate its content.
	if len(msgs) == 1 && EstimateConversation(m.system, msgs) > m.Available() {
		msgs[0].Content = truncateToTokens(msgs[0].Content, m.Available()-1000)
	}
	return msgs
}

func truncateToTokens(s string, maxTokens int) string {
	if maxTokens < 1 {
		maxTokens = 1
	}
	maxChars := maxTokens * 4
	if len(s) <= maxChars {
		return s
	}
	return s[:maxChars] + "\n…[truncated]"
}

// Stats reports current usage against the budget.
type Stats struct {
	UsedTokens      int
	AvailableTokens int
	Percent         float64
}

// Analyze returns usage statistics for the given conversation.
func (m *Manager) Analyze(msgs []llm.Message) Stats {
	used := EstimateConversation(m.system, msgs)
	avail := m.Available()
	pct := 0.0
	if avail > 0 {
		pct = float64(used) / float64(avail) * 100
	}
	return Stats{UsedTokens: used, AvailableTokens: avail, Percent: pct}
}
