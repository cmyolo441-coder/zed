// Package metrics provides real-time code metrics tracking. It monitors
// complexity, coverage, technical debt, and project health over time —
// giving a live dashboard of codebase quality.
package metrics

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// MetricType categorizes a tracked metric.
type MetricType int

const (
	MetricComplexity MetricType = iota
	MetricCoverage
	MetricTechDebt
	MetricHealth
	MetricLOC
	MetricFiles
	MetricFunctions
	MetricIssues
)

func (m MetricType) String() string {
	switch m {
	case MetricComplexity:
		return "complexity"
	case MetricCoverage:
		return "coverage"
	case MetricTechDebt:
		return "tech_debt"
	case MetricHealth:
		return "health"
	case MetricLOC:
		return "lines_of_code"
	case MetricFiles:
		return "files"
	case MetricFunctions:
		return "functions"
	case MetricIssues:
		return "issues"
	default:
		return "unknown"
	}
}

// Sample is a single metric measurement at a point in time.
type Sample struct {
	Type      MetricType
	Value     float64
	Timestamp time.Time
	Detail    string
}

// Dashboard tracks metrics over time.
type Dashboard struct {
	mu      sync.RWMutex
	samples []Sample
	health  float64 // current health score 0-10
	debt    float64 // accumulated technical debt
}

// New creates an empty dashboard.
func New() *Dashboard {
	return &Dashboard{health: 10.0}
}

// Record adds a metric sample.
func (d *Dashboard) Record(t MetricType, value float64, detail string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.samples = append(d.samples, Sample{
		Type:      t,
		Value:     value,
		Timestamp: time.Now(),
		Detail:    detail,
	})
	// Update health score.
	switch t {
	case MetricIssues:
		d.health -= value * 0.1
	case MetricCoverage:
		if value > 80 {
			d.health += 0.1
		} else if value < 50 {
			d.health -= 0.2
		}
	case MetricTechDebt:
		d.debt += value
		d.health -= value * 0.05
	}
	if d.health < 0 {
		d.health = 0
	}
	if d.health > 10 {
		d.health = 10
	}
}

// Health returns the current project health score (0-10).
func (d *Dashboard) Health() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.health
}

// Trend returns how a metric has changed over time.
func (d *Dashboard) Trend(t MetricType, window time.Duration) (start, end float64, trend string) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	cutoff := time.Now().Add(-window)
	var relevant []Sample
	for _, s := range d.samples {
		if s.Type == t && s.Timestamp.After(cutoff) {
			relevant = append(relevant, s)
		}
	}
	if len(relevant) == 0 {
		return 0, 0, "no data"
	}
	start = relevant[0].Value
	end = relevant[len(relevant)-1].Value
	if end > start {
		trend = "📈 increasing"
	} else if end < start {
		trend = "📉 decreasing"
	} else {
		trend = "➡️ stable"
	}
	return
}

// Render returns a dashboard summary.
func (d *Dashboard) Render() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var b []byte
	b = append(b, "📊 Code Metrics Dashboard\n\n"...)
	b = append(b, fmt.Sprintf("  Project Health: %.1f/10 %s\n", d.health, healthEmoji(d.health))...)
	b = append(b, fmt.Sprintf("  Technical Debt: %.1f\n", d.debt)...)

	// Latest values per metric type.
	latest := make(map[MetricType]Sample)
	for _, s := range d.samples {
		latest[s.Type] = s
	}
	types := []MetricType{MetricLOC, MetricFiles, MetricFunctions, MetricComplexity, MetricCoverage, MetricIssues}
	for _, t := range types {
		if s, ok := latest[t]; ok {
			b = append(b, fmt.Sprintf("  %s: %.1f (%s)\n", t, s.Value, s.Detail)...)
		}
	}
	return string(b)
}

func healthEmoji(h float64) string {
	if h >= 8 {
		return "💚"
	} else if h >= 5 {
		return "💛"
	} else if h >= 3 {
		return "🧡"
	}
	return "❤️"
}

// History returns all samples sorted by time.
func (d *Dashboard) History() []Sample {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]Sample, len(d.samples))
	copy(out, d.samples)
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out
}
