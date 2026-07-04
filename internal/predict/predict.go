// Package predict implements intent prediction. It learns from the user's
// past interactions to predict what they'll ask next — enabling proactive
// suggestions and auto-complete.
package predict

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Pattern is a learned user behavior pattern.
type Pattern struct {
	Trigger    string // what the user did
	FollowUp   string // what they usually do next
	Count      int    // how many times observed
	LastSeen   time.Time
}

// Engine learns from user interactions and predicts intent.
type Engine struct {
	mu       sync.RWMutex
	patterns map[string]*Pattern // key: trigger → followUp
	history  []string             // recent user inputs
}

// New creates an intent prediction engine.
func New() *Engine {
	return &Engine{patterns: map[string]*Pattern{}}
}

// Observe records a user action and learns from it.
func (e *Engine) Observe(input string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return
	}
	// If we have a previous input, learn the pattern.
	if len(e.history) > 0 {
		prev := e.history[len(e.history)-1]
		key := prev + "→" + input
		if p, ok := e.patterns[key]; ok {
			p.Count++
			p.LastSeen = time.Now()
		} else {
			e.patterns[key] = &Pattern{
				Trigger:  prev,
				FollowUp: input,
				Count:    1,
				LastSeen: time.Now(),
			}
		}
	}
	e.history = append(e.history, input)
	if len(e.history) > 100 {
		e.history = e.history[len(e.history)-50:]
	}
}

// Predict returns the most likely next action based on patterns.
func (e *Engine) Predict(currentInput string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	current := strings.ToLower(strings.TrimSpace(currentInput))

	// Collect candidate follow-ups with their counts.
	type candidate struct {
		text  string
		count int
	}
	var candidates []candidate
	for key, p := range e.patterns {
		if p.Trigger == current && p.Count > 0 && strings.HasPrefix(key, current+"→") {
			candidates = append(candidates, candidate{text: p.FollowUp, count: p.Count})
		}
	}
	// Sort by frequency descending (simple insertion sort — candidates list is tiny).
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].count < candidates[j].count {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
	predictions := make([]string, len(candidates))
	for i, c := range candidates {
		predictions[i] = fmt.Sprintf("%s (seen %d times)", c.text, c.count)
	}
	return predictions
}

// Suggestions returns proactive suggestions based on common patterns.
func (e *Engine) Suggestions() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var suggestions []string
	for _, p := range e.patterns {
		if p.Count >= 3 {
			suggestions = append(suggestions, fmt.Sprintf("After %q, you usually ask: %q", p.Trigger, p.FollowUp))
		}
	}
	return suggestions
}
