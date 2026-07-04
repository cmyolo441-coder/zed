// Package dist implements a distributed agent network. Multiple ZED instances
// can connect to each other, share a task queue, and steal work from each
// other — enabling cluster-scale parallelism for massive projects.
package dist

import (
	"fmt"
	"sync"
	"time"
)

// Node represents a ZED instance in the network.
type Node struct {
	ID        string
	Address   string
	Capacity  int // max concurrent tasks
	Active    int // current active tasks
	LastSeen  time.Time
}

// Task is a unit of work in the distributed queue.
type Task struct {
	ID          string
	Description string
	Status      TaskStatus
	AssignedTo  string // node ID
	CreatedAt   time.Time
	StartedAt   time.Time
	CompletedAt time.Time
	Result      string
}

type TaskStatus int

const (
	TaskQueued   TaskStatus = iota
	TaskAssigned
	TaskRunning
	TaskDone
	TaskFailed
)

func (s TaskStatus) String() string {
	switch s {
	case TaskQueued:
		return "queued"
	case TaskAssigned:
		return "assigned"
	case TaskRunning:
		return "running"
	case TaskDone:
		return "done"
	case TaskFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Network is the distributed agent coordinator.
type Network struct {
	mu    sync.RWMutex
	nodes map[string]*Node
	queue []*Task
}

// New creates an empty network.
func New() *Network {
	return &Network{
		nodes: map[string]*Node{},
	}
}

// Register adds a node to the network.
func (n *Network) Register(node *Node) {
	n.mu.Lock()
	defer n.mu.Unlock()
	node.LastSeen = time.Now()
	n.nodes[node.ID] = node
}

// Heartbeat updates a node's last-seen time.
func (n *Network) Heartbeat(nodeID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if node, ok := n.nodes[nodeID]; ok {
		node.LastSeen = time.Now()
	}
}

// Submit adds a task to the distributed queue.
func (n *Network) Submit(task *Task) {
	n.mu.Lock()
	defer n.mu.Unlock()
	task.Status = TaskQueued
	task.CreatedAt = time.Now()
	n.queue = append(n.queue, task)
}

// StealWork lets an idle node take a queued task.
func (n *Network) StealWork(nodeID string) *Task {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, t := range n.queue {
		if t.Status == TaskQueued {
			t.Status = TaskAssigned
			t.AssignedTo = nodeID
			t.StartedAt = time.Now()
			return t
		}
	}
	return nil
}

// CompleteTask marks a task as done with a result.
func (n *Network) CompleteTask(taskID, result string, success bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, t := range n.queue {
		if t.ID == taskID {
			t.CompletedAt = time.Now()
			t.Result = result
			if success {
				t.Status = TaskDone
			} else {
				t.Status = TaskFailed
			}
			return
		}
	}
}

// Status returns the network status summary.
func (n *Network) Status() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	var b []byte
	b = append(b, fmt.Sprintf("🌐 Distributed Network: %d nodes, %d tasks\n", len(n.nodes), len(n.queue))...)
	for _, node := range n.nodes {
		b = append(b, fmt.Sprintf("  Node %s: %d/%d active, last seen %v ago\n",
			node.ID, node.Active, node.Capacity, time.Since(node.LastSeen).Round(time.Second))...)
	}
	queued, running, done := 0, 0, 0
	for _, t := range n.queue {
		switch t.Status {
		case TaskQueued:
			queued++
		case TaskRunning, TaskAssigned:
			running++
		case TaskDone:
			done++
		}
	}
	b = append(b, fmt.Sprintf("\n  Tasks: %d queued, %d running, %d done\n", queued, running, done)...)
	return string(b)
}
