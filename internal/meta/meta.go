// Package meta implements self-modification capabilities. The agent can
// analyze its own performance, identify bottlenecks in its tools, and
// suggest or apply optimizations to its own codebase — getting smarter
// over time.
package meta

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Metric records a single performance measurement of a tool or operation.
type Metric struct {
	Tool      string
	Duration  time.Duration
	Success   bool
	Timestamp time.Time
	Input     string // short description of input
}

// Profile holds accumulated performance metrics for analysis.
type Profile struct {
	mu      sync.RWMutex
	metrics []Metric
}

// New creates an empty performance profile.
func New() *Profile { return &Profile{} }

// Record adds a performance metric.
func (p *Profile) Record(tool string, duration time.Duration, success bool, input string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.metrics = append(p.metrics, Metric{
		Tool:      tool,
		Duration:  duration,
		Success:   success,
		Timestamp: time.Now(),
		Input:     input,
	})
}

// Bottleneck returns the slowest tools by average duration.
func (p *Profile) Bottleneck() []Bottleneck {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]*toolStat)
	for _, m := range p.metrics {
		s, ok := stats[m.Tool]
		if !ok {
			s = &toolStat{}
			stats[m.Tool] = s
		}
		s.total += m.Duration
		s.count++
		if !m.Success {
			s.failures++
		}
	}

	var bottlenecks []Bottleneck
	for tool, s := range stats {
		bottlenecks = append(bottlenecks, Bottleneck{
			Tool:        tool,
			AvgDuration: s.total / time.Duration(s.count),
			Calls:       s.count,
			Failures:    s.failures,
			FailRate:    float64(s.failures) / float64(s.count) * 100,
		})
	}
	sort.Slice(bottlenecks, func(i, j int) bool {
		return bottlenecks[i].AvgDuration > bottlenecks[j].AvgDuration
	})
	return bottlenecks
}

type toolStat struct {
	total    time.Duration
	count    int
	failures int
}

// Bottleneck is a performance hotspot summary.
type Bottleneck struct {
	Tool        string
	AvgDuration time.Duration
	Calls       int
	Failures    int
	FailRate   float64
}

// Suggestion is a self-improvement recommendation.
type Suggestion struct {
	Tool     string
	Issue    string // "slow", "failing", "redundant"
	Fix      string // suggested optimization
	Priority string // "high", "medium", "low"
}

// Analyze returns self-improvement suggestions based on performance data.
func (p *Profile) Analyze() []Suggestion {
	bottlenecks := p.Bottleneck()
	var suggestions []Suggestion
	for _, b := range bottlenecks {
		if b.AvgDuration > 5*time.Second {
			suggestions = append(suggestions, Suggestion{
				Tool:     b.Tool,
				Issue:    "slow",
				Fix:      fmt.Sprintf("%s averages %v — consider caching results or optimizing the implementation", b.Tool, b.AvgDuration),
				Priority: "high",
			})
		}
		if b.FailRate > 20 {
			suggestions = append(suggestions, Suggestion{
				Tool:     b.Tool,
				Issue:    "failing",
				Fix:      fmt.Sprintf("%s has %.1f%% failure rate — investigate error handling and edge cases", b.Tool, b.FailRate),
				Priority: "high",
			})
		}
		if b.Calls > 50 {
			suggestions = append(suggestions, Suggestion{
				Tool:     b.Tool,
				Issue:    "redundant",
				Fix:      fmt.Sprintf("%s called %d times — consider batching or caching to reduce calls", b.Tool, b.Calls),
				Priority: "medium",
			})
		}
	}
	return suggestions
}

// Report returns a human-readable performance analysis.
func (p *Profile) Report() string {
	bottlenecks := p.Bottleneck()
	suggestions := p.Analyze()
	var b strings.Builder
	b.WriteString("🤖 Self-Analysis Report\n\n")
	b.WriteString("Performance Bottlenecks:\n")
	for _, bn := range bottlenecks {
		fmt.Fprintf(&b, "  %s: avg %v, %d calls, %.1f%% fail\n", bn.Tool, bn.AvgDuration, bn.Calls, bn.FailRate)
	}
	if len(suggestions) > 0 {
		b.WriteString("\nSelf-Improvement Suggestions:\n")
		for _, s := range suggestions {
			fmt.Fprintf(&b, "  [%s] %s: %s\n    → %s\n", s.Priority, s.Tool, s.Issue, s.Fix)
		}
	}
	return b.String()
}

// SelfModifyPath returns the path to the agent's own source code.
func SelfModifyPath() string {
	_, file, _, _ := runtime.Caller(0)
	// Walk up to find go.mod.
	dir := filepath.Dir(file)
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return ""
}
