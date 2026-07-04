// Package diagram generates architecture diagrams from source code. It creates
// Mermaid diagrams for dependency graphs, call graphs, data flow, and class
// relationships — giving instant visual understanding of the codebase.
package diagram

import (
	"fmt"
	"strings"
)

// DiagramType specifies what kind of diagram to generate.
type DiagramType int

const (
	DiagramDependency DiagramType = iota
	DiagramCallGraph
	DiagramClassRel
	DiagramDataFlow
	DiagramSequence
)

func (d DiagramType) String() string {
	switch d {
	case DiagramDependency:
		return "dependency"
	case DiagramCallGraph:
		return "call_graph"
	case DiagramClassRel:
		return "class_relationship"
	case DiagramDataFlow:
		return "data_flow"
	case DiagramSequence:
		return "sequence"
	default:
		return "unknown"
	}
}

// Node is a node in a diagram (file, function, class, etc.).
type Node struct {
	ID    string
	Label string
	Type  string // "file", "function", "class", "module"
}

// Edge is a connection between nodes.
type Edge struct {
	From  string
	To    string
	Label string // "calls", "imports", "depends on", "uses"
}

// Diagram holds the nodes and edges of a visualization.
type Diagram struct {
	Type  DiagramType
	Nodes []Node
	Edges []Edge
}

// RenderMermaid returns a Mermaid diagram string.
func (d *Diagram) RenderMermaid() string {
	var b strings.Builder
	switch d.Type {
	case DiagramDependency:
		b.WriteString("graph TD\n")
	case DiagramCallGraph:
		b.WriteString("graph LR\n")
	case DiagramClassRel:
		b.WriteString("classDiagram\n")
	case DiagramDataFlow:
		b.WriteString("graph TD\n")
	case DiagramSequence:
		b.WriteString("sequenceDiagram\n")
	}

	// Nodes.
	for _, n := range d.Nodes {
		switch n.Type {
		case "file":
			fmt.Fprintf(&b, "  %s[\"%s\"]\n", n.ID, n.Label)
		case "function":
			fmt.Fprintf(&b, "  %s(\"%s\")\n", n.ID, n.Label)
		case "class":
			fmt.Fprintf(&b, "  %s[%s]\n", n.ID, n.Label)
		default:
			fmt.Fprintf(&b, "  %s[%s]\n", n.ID, n.Label)
		}
	}

	// Edges.
	for _, e := range d.Edges {
		if e.Label != "" {
			fmt.Fprintf(&b, "  %s -->|%s| %s\n", e.From, e.Label, e.To)
		} else {
			fmt.Fprintf(&b, "  %s --> %s\n", e.From, e.To)
		}
	}
	return b.String()
}

// BuildDependencyDiagram creates a dependency graph from file imports.
func BuildDependencyDiagram(files map[string][]string) *Diagram {
	d := &Diagram{Type: DiagramDependency}
	seen := make(map[string]bool)
	for file, imports := range files {
		if !seen[file] {
			nodeID := sanitizeID(file)
			d.Nodes = append(d.Nodes, Node{ID: nodeID, Label: file, Type: "file"})
			seen[file] = true
		}
		for _, imp := range imports {
			if !seen[imp] {
				d.Nodes = append(d.Nodes, Node{ID: sanitizeID(imp), Label: imp, Type: "file"})
				seen[imp] = true
			}
			d.Edges = append(d.Edges, Edge{
				From: sanitizeID(file),
				To:   sanitizeID(imp),
				Label: "imports",
			})
		}
	}
	return d
}

// BuildCallGraph creates a function call graph.
func BuildCallGraph(calls map[string][]string) *Diagram {
	d := &Diagram{Type: DiagramCallGraph}
	seen := make(map[string]bool)
	for caller, callees := range calls {
		if !seen[caller] {
			d.Nodes = append(d.Nodes, Node{ID: sanitizeID(caller), Label: caller, Type: "function"})
			seen[caller] = true
		}
		for _, callee := range callees {
			if !seen[callee] {
				d.Nodes = append(d.Nodes, Node{ID: sanitizeID(callee), Label: callee, Type: "function"})
				seen[callee] = true
			}
			d.Edges = append(d.Edges, Edge{
				From: sanitizeID(caller),
				To:   sanitizeID(callee),
				Label: "calls",
			})
		}
	}
	return d
}

// BuildClassDiagram creates a class relationship diagram.
func BuildClassDiagram(classes map[string][]string) *Diagram {
	d := &Diagram{Type: DiagramClassRel}
	seen := make(map[string]bool)
	for class, deps := range classes {
		if !seen[class] {
			d.Nodes = append(d.Nodes, Node{ID: sanitizeID(class), Label: class, Type: "class"})
			seen[class] = true
		}
		for _, dep := range deps {
			if !seen[dep] {
				d.Nodes = append(d.Nodes, Node{ID: sanitizeID(dep), Label: dep, Type: "class"})
				seen[dep] = true
			}
			d.Edges = append(d.Edges, Edge{
				From: sanitizeID(class),
				To:   sanitizeID(dep),
				Label: "uses",
			})
		}
	}
	return d
}

func sanitizeID(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, s)
}
