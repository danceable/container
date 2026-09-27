package visualize

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// DOT renders graphs in the Graphviz DOT language:
//
//	dot -Tsvg container.dot -o container.svg
type DOT struct{}

func (DOT) ContentType() string {
	return "text/vnd.graphviz; charset=utf-8"
}

// Render writes the graph to w.
func (DOT) Render(w io.Writer, g Graph) error {
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
