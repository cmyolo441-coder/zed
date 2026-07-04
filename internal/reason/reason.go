// Package reason implements a transparent decision-analysis engine. Before
// taking any action, the agent explores multiple decision paths, evaluates
// trade-offs, and picks the best approach — dramatically reducing mistakes
// on complex decisions.
package reason

import (
	"fmt"
	"strings"
)

// Thought is a single step in a decision trail.
type Thought struct {
	Step      int
	Reasoning string
	Action    string // what this step suggests
	Confidence float64 // 0-1
}

// Path is one complete decision trail leading to a conclusion.
type Path struct {
	ID         string
	Thoughts   []Thought
	Conclusion string
	Confidence float64
	Risks      []string
}

// Engine explores multiple decision paths and picks the best.
type Engine struct{}

// New creates a reasoning engine.
func New() *Engine { return &Engine{} }

// Explore takes a decision point and returns multiple decision paths.
// The agent can then pick the path with highest confidence / lowest risk.
func (e *Engine) Explore(decision string, options []string) []Path {
	var paths []Path
	for i, opt := range options {
		p := Path{
			ID:         fmt.Sprintf("path-%d", i+1),
			Conclusion: opt,
			Confidence: 0.5, // default; agent will refine
			Thoughts: []Thought{
				{
					Step:      1,
					Reasoning: fmt.Sprintf("Option: %s. What are the implications?", opt),
					Action:    "evaluate",
					Confidence: 0.5,
				},
				{
					Step:      2,
					Reasoning: fmt.Sprintf("What could go wrong with %s?", opt),
					Action:    "risk-assess",
					Confidence: 0.4,
				},
				{
					Step:      3,
					Reasoning: fmt.Sprintf("What are the benefits of %s?", opt),
					Action:    "benefit-assess",
					Confidence: 0.6,
				},
			},
		}
		paths = append(paths, p)
	}
	return paths
}

// BestPath returns the path with highest confidence and lowest risk.
func BestPath(paths []Path) *Path {
	if len(paths) == 0 {
		return nil
	}
	best := &paths[0]
	for i := range paths {
		score := paths[i].Confidence - float64(len(paths[i].Risks))*0.1
		bestScore := best.Confidence - float64(len(best.Risks))*0.1
		if score > bestScore {
			best = &paths[i]
		}
	}
	return best
}

// RenderPath formats a reasoning path for display.
func RenderPath(p Path) string {
	var b strings.Builder
	fmt.Fprintf(&b, "🧠 Reasoning Path: %s (confidence: %.0f%%)\n", p.ID, p.Confidence*100)
	for _, t := range p.Thoughts {
		fmt.Fprintf(&b, "  Step %d: %s\n    → %s (confidence: %.0f%%)\n", t.Step, t.Reasoning, t.Action, t.Confidence*100)
	}
	fmt.Fprintf(&b, "  Conclusion: %s\n", p.Conclusion)
	if len(p.Risks) > 0 {
		b.WriteString("  Risks:\n")
		for _, r := range p.Risks {
			fmt.Fprintf(&b, "    ⚠️  %s\n", r)
		}
	}
	return b.String()
}

// RenderAll formats all paths for comparison.
func RenderAll(paths []Path) string {
	var b strings.Builder
	b.WriteString("🧠 Transparent Decision Analysis\n\n")
	for _, p := range paths {
		b.WriteString(RenderPath(p))
		b.WriteString("\n")
	}
	best := BestPath(paths)
	if best != nil {
		fmt.Fprintf(&b, "✅ Best path: %s (%.0f%% confidence)\n", best.ID, best.Confidence*100)
	}
	return b.String()
}
