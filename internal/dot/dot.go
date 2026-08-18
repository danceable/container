// Package dot renders directed graphs in the Graphviz DOT language.
//
// It knows the syntax and nothing about what is being drawn: callers describe their
// graph with the model below and stay free of DOT quoting and attribute rules.
package dot

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Style is how a node or an edge is drawn.
type Style uint8

const (
	Solid Style = iota
	Dashed
	Highlighted
)

// Node is a single box, identified by a number unique within the Graph.
type Node struct {
	ID    int
	Label string
	Style Style
}

// Cluster is a group of nodes drawn in a labelled box of its own.
type Cluster struct {
	Label string
	Nodes []Node
}

// Edge points from one node to another.
type Edge struct {
	From  int
	To    int
	Style Style
}

// Graph is a directed graph ready to be rendered.
type Graph struct {
	Name     string
	Clusters []Cluster
	Nodes    []Node // nodes outside of any cluster
	Edges    []Edge
}

// Write renders the graph and writes it to w.
func Write(w io.Writer, g Graph) error {
	var dot strings.Builder

	fmt.Fprintf(&dot, "digraph %s {\n", g.Name)
	dot.WriteString("\trankdir = LR;\n")
	dot.WriteString("\tnode [shape = box, style = rounded, fontname = \"Helvetica\"];\n")
	dot.WriteString("\tedge [fontname = \"Helvetica\"];\n")

	for i, cluster := range g.Clusters {
		writeCluster(&dot, i, cluster)
	}

	if len(g.Nodes) > 0 {
		dot.WriteString("\n")
		for _, node := range g.Nodes {
			writeNode(&dot, "\t", node)
		}
	}

	if len(g.Edges) > 0 {
		dot.WriteString("\n")
		for _, edge := range g.Edges {
			writeEdge(&dot, edge)
		}
	}

	dot.WriteString("}\n")

	_, err := io.WriteString(w, dot.String())

	return err
}

func writeCluster(dot *strings.Builder, index int, c Cluster) {
	fmt.Fprintf(dot, "\n\tsubgraph cluster_%d {\n", index)
	fmt.Fprintf(dot, "\t\tlabel = %s;\n", strconv.Quote(c.Label))
	dot.WriteString("\t\tstyle = rounded;\n\t\tcolor = \"#b0b0b0\";\n\t\tfontcolor = \"#606060\";\n\n")

	for _, node := range c.Nodes {
		writeNode(dot, "\t\t", node)
	}

	dot.WriteString("\t}\n")
}

func writeNode(dot *strings.Builder, indent string, n Node) {
	fmt.Fprintf(dot, "%sn%d [label = %s", indent, n.ID, strconv.Quote(n.Label))

	switch n.Style {
	case Dashed:
		dot.WriteString(", style = dashed, color = \"#909090\", fontcolor = \"#606060\"")
	case Highlighted:
		dot.WriteString(", color = red, fontcolor = red")
	}

	dot.WriteString("];\n")
}

func writeEdge(dot *strings.Builder, e Edge) {
	fmt.Fprintf(dot, "\tn%d -> n%d", e.From, e.To)

	if e.Style == Highlighted {
		dot.WriteString(" [color = red, penwidth = 2]")
	}

	dot.WriteString(";\n")
}
