package visualize

import (
	"io"
	"strings"
)

// ASCII renders graphs as plain text trees for the terminal, one per cluster, listing
// under every node the nodes it points to:
//
//	root
//	|-- main.Database (singleton, resolved)
//	`-- main.Shape (transient)
//	    `-> main.Database
//
//	scope "request"
//	`-- main.Logger (singleton)
//	    `-> main.Database [root]
//
// A target outside the cluster of its node is followed by the label of its own cluster.
// A highlighted node or edge is marked with (!) and drawn as =>, a dashed one is left as
// its label describes it.
type ASCII struct {
	// Color draws highlighted items in red and dashed ones dimmed, using ANSI escape
	// codes. Leave it off when the output does not go to a terminal.
	Color bool
}

const (
	ansiRed   = "\x1b[31m"
	ansiDim   = "\x1b[2m"
	ansiReset = "\x1b[0m"
)

func (ASCII) ContentType() string {
	return "text/plain; charset=utf-8"
}

// Render writes the graph to w.
func (a ASCII) Render(w io.Writer, g Graph) error {
	var out strings.Builder

	// Where every node is, so an edge can name its target.
	names := make(map[int]string)
	clusterOf := make(map[int]int)
	for i, cluster := range g.Clusters {
		for _, n := range cluster.Nodes {
			names[n.ID], _, _ = strings.Cut(n.Label, "\n")
			clusterOf[n.ID] = i
		}
	}
	for _, n := range g.Nodes {
		names[n.ID], _, _ = strings.Cut(n.Label, "\n")
		clusterOf[n.ID] = -1
	}

	edges := make(map[int][]Edge)
	for _, e := range g.Edges {
		edges[e.From] = append(edges[e.From], e)
	}

	// writeNodes draws the nodes of a cluster as the branches of a tree, and the nodes
	// outside any cluster, numbered -1, as roots of their own.
	writeNodes := func(nodes []Node, cluster int) {
		for i, n := range nodes {
			branch, next := "|-- ", "|   "
			switch {
			case cluster < 0:
				branch, next = "", "    "
			case i == len(nodes)-1:
				branch, next = "`-- ", "    "
			}

			out.WriteString(branch + a.paint(nodeLabel(n), n.Style) + "\n")

			for j, e := range edges[n.ID] {
				connector, arrow := "|", "-> "
				if j == len(edges[n.ID])-1 {
					connector = "`"
				}

				target := names[e.To]
				if c, exist := clusterOf[e.To]; exist && c != cluster && c >= 0 {
					target += " [" + g.Clusters[c].Label + "]"
				}
				if e.Style == Highlighted {
					arrow = "=> "
					target += " (!)"
				}

				out.WriteString(next + a.paint(connector+arrow+target, e.Style) + "\n")
			}
		}
	}

	for i, cluster := range g.Clusters {
		if i > 0 {
			out.WriteString("\n")
		}

		out.WriteString(cluster.Label + "\n")
		writeNodes(cluster.Nodes, i)
	}

	if len(g.Nodes) > 0 {
		if len(g.Clusters) > 0 {
			out.WriteString("\n")
		}

		writeNodes(g.Nodes, -1)
	}

	if out.Len() == 0 {
		out.WriteString("(empty graph)\n")
	}

	_, err := io.WriteString(w, out.String())

	return err
}

// nodeLabel returns the label of a node on a single line, the lines describing it in
// parentheses.
func nodeLabel(n Node) string {
	lines := strings.Split(n.Label, "\n")

	label := lines[0]
	if len(lines) > 1 {
		label += " (" + strings.Join(lines[1:], "; ") + ")"
	}
	if n.Style == Highlighted {
		label += " (!)"
	}

	return label
}

// paint colors text as the style asks, when colors are on.
func (a ASCII) paint(text string, s Style) string {
	if !a.Color {
		return text
	}

	switch s {
	case Highlighted:
		return ansiRed + text + ansiReset
	case Dashed:
		return ansiDim + text + ansiReset
	default:
		return text
	}
}
