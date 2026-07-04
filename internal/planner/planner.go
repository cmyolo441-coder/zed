// Package planner implements intelligent task decomposition. It breaks a
// complex goal into a tree of sub-tasks with dependencies, tracks progress,
// and identifies the critical path — so the agent can work systematically
// instead of chaotically on large multi-file projects.
package planner

import (
	"fmt"
	"strings"
	"sync"
)

// Status represents the state of a task.
type Status int

const (
	StatusPending   Status = iota // not started
	StatusInProgress              // currently being worked on
	StatusBlocked                 // waiting on a dependency
	StatusDone                    // completed
	StatusFailed                  // attempted but failed
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "⏳ pending"
	case StatusInProgress:
		return "🔄 in progress"
	case StatusBlocked:
		return "🚫 blocked"
	case StatusDone:
		return "✅ done"
	case StatusFailed:
		return "❌ failed"
	default:
		return "unknown"
	}
}

// Task is a single unit of work in a decomposition plan.
type Task struct {
	ID           string   // unique identifier (e.g. "1", "1.2")
	Title        string   // short description
	Description  string   // detailed instructions
	Dependencies []string  // task IDs that must complete before this one
	Files        []string  // files this task will touch
	Status       Status
	SubTasks     []*Task // child tasks (for hierarchical decomposition)
	parent       *Task
}

// Plan is a complete task decomposition.
type Plan struct {
	Goal    string
	Root    *Task
	tasks   map[string]*Task // flat index by ID
	mu      sync.RWMutex
}

// NewPlan creates a plan from a goal and a root task tree.
func NewPlan(goal string, root *Task) *Plan {
	p := &Plan{
		Goal:  goal,
		Root:  root,
		tasks: map[string]*Task{},
	}
	p.indexTask(root, nil)
	return p
}

func (p *Plan) indexTask(t *Task, parent *Task) {
	t.parent = parent
	p.tasks[t.ID] = t
	for _, st := range t.SubTasks {
		p.indexTask(st, t)
	}
}

// Get returns a task by ID.
func (p *Plan) Get(id string) (*Task, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	t, ok := p.tasks[id]
	return t, ok
}

// UpdateStatus sets the status of a task and updates blocked tasks.
func (p *Plan) UpdateStatus(id string, status Status) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if t, ok := p.tasks[id]; ok {
		t.Status = status
		// Check if any blocked tasks can now proceed.
		for _, t2 := range p.tasks {
			if t2.Status == StatusBlocked {
				if p.depsMet(t2) {
					t2.Status = StatusPending
				}
			}
		}
	}
}

// depsMet checks if all dependencies of a task are done.
func (p *Plan) depsMet(t *Task) bool {
	for _, depID := range t.Dependencies {
		if dep, ok := p.tasks[depID]; ok {
			if dep.Status != StatusDone {
				return false
			}
		}
	}
	return true
}

// Ready returns all tasks that are ready to start (pending + deps met).
func (p *Plan) Ready() []*Task {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var ready []*Task
	for _, t := range p.tasks {
		if t.Status == StatusPending && p.depsMet(t) {
			ready = append(ready, t)
		}
	}
	return ready
}

// Blocked returns tasks that are blocked by incomplete dependencies.
func (p *Plan) Blocked() []*Task {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var blocked []*Task
	for _, t := range p.tasks {
		if t.Status == StatusPending && !p.depsMet(t) {
			blocked = append(blocked, t)
		}
	}
	return blocked
}

// Progress returns completion statistics.
func (p *Plan) Progress() (done, total int, pct float64) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, t := range p.tasks {
		if len(t.SubTasks) > 0 {
			continue // don't count parent tasks
		}
		total++
		if t.Status == StatusDone {
			done++
		}
	}
	if total > 0 {
		pct = float64(done) / float64(total) * 100
	}
	return
}

// CriticalPath returns the longest dependency chain (bottleneck).
func (p *Plan) CriticalPath() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var longest []string
	for _, t := range p.tasks {
		if len(t.SubTasks) > 0 {
			continue
		}
		path := p.chainTo(t)
		if len(path) > len(longest) {
			longest = path
		}
	}
	return longest
}

func (p *Plan) chainTo(t *Task) []string {
	return p.chainToVisited(t, map[string]bool{})
}

func (p *Plan) chainToVisited(t *Task, visited map[string]bool) []string {
	if t == nil || visited[t.ID] {
		return nil
	}
	visited[t.ID] = true
	chain := []string{t.ID}
	for _, depID := range t.Dependencies {
		if dep, ok := p.tasks[depID]; ok {
			chain = append(p.chainToVisited(dep, visited), chain...)
		}
	}
	return chain
}

// Render returns a tree visualization of the plan.
func (p *Plan) Render() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var b strings.Builder
	fmt.Fprintf(&b, "🎯 Goal: %s\n\n", p.Goal)
	p.renderTask(&b, p.Root, 0)
	done, total, pct := p.Progress()
	fmt.Fprintf(&b, "\n📊 Progress: %d/%d tasks (%.0f%%)\n", done, total, pct)
	return b.String()
}

func (p *Plan) renderTask(b *strings.Builder, t *Task, depth int) {
	indent := strings.Repeat("  ", depth)
	marker := "▸"
	if t.Status == StatusDone {
		marker = "✓"
	} else if t.Status == StatusInProgress {
		marker = "🔄"
	} else if t.Status == StatusFailed {
		marker = "✗"
	}
	fmt.Fprintf(b, "%s%s [%s] %s\n", indent, marker, t.ID, t.Title)
	if t.Description != "" && depth == 0 {
		fmt.Fprintf(b, "%s   %s\n", indent, t.Description)
	}
	for _, st := range t.SubTasks {
		p.renderTask(b, st, depth+1)
	}
}

// AllTasks returns all leaf tasks (tasks without subtasks) in dependency order.
func (p *Plan) AllTasks() []*Task {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var tasks []*Task
	for _, t := range p.tasks {
		if len(t.SubTasks) == 0 {
			tasks = append(tasks, t)
		}
	}
	return tasks
}
