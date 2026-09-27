package visualize_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/danceable/container/visualize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderDOT(t *testing.T, g visualize.Graph) string {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, visualize.DOT{}.Render(&buf, g))

	return buf.String()
}

func TestDOT(t *testing.T) {
	t.Parallel()

	t.Run("renders_an_empty_graph", func(t *testing.T) {
		t.Parallel()

		out := renderDOT(t, visualize.Graph{Name: "empty"})

		assert.Equal(t, "digraph empty {\n"+
			"\trankdir = LR;\n"+
			"\tnode [shape = box, style = rounded, fontname = \"Helvetica\"];\n"+
			"\tedge [fontname = \"Helvetica\"];\n"+
			"}\n", out)
	})

	t.Run("renders_clusters_with_their_nodes", func(t *testing.T) {
		t.Parallel()

		out := renderDOT(t, visualize.Graph{
			Name: "container",
			Clusters: []visualize.Cluster{
				{Label: "root", Nodes: []visualize.Node{{ID: 0, Label: "main.Shape"}}},
				{Label: `scope "request"`, Nodes: []visualize.Node{{ID: 1, Label: "main.Logger"}}},
			},
		})

		assert.Contains(t, out, "\tsubgraph cluster_0 {\n\t\tlabel = \"root\";\n")
		assert.Contains(t, out, "\t\tn0 [label = \"main.Shape\"];\n")
		assert.Contains(t, out, "\tsubgraph cluster_1 {\n\t\tlabel = \"scope \\\"request\\\"\";\n")
		assert.Contains(t, out, "\t\tn1 [label = \"main.Logger\"];\n")
	})

	t.Run("renders_the_style_of_a_node", func(t *testing.T) {
		t.Parallel()

		out := renderDOT(t, visualize.Graph{
			Nodes: []visualize.Node{
				{ID: 0, Label: "solid"},
				{ID: 1, Label: "dashed", Style: visualize.Dashed},
				{ID: 2, Label: "highlighted", Style: visualize.Highlighted},
			},
		})

		assert.Contains(t, out, "\tn0 [label = \"solid\"];\n")
		assert.Contains(t, out, "\tn1 [label = \"dashed\", style = dashed, color = \"#909090\", fontcolor = \"#606060\"];\n")
		assert.Contains(t, out, "\tn2 [label = \"highlighted\", color = red, fontcolor = red];\n")
	})

	t.Run("renders_the_style_of_an_edge", func(t *testing.T) {
		t.Parallel()

		out := renderDOT(t, visualize.Graph{
			Edges: []visualize.Edge{
				{From: 0, To: 1},
				{From: 1, To: 0, Style: visualize.Highlighted},
			},
		})

		assert.Contains(t, out, "\tn0 -> n1;\n")
		assert.Contains(t, out, "\tn1 -> n0 [color = red, penwidth = 2];\n")
	})

	t.Run("quotes_what_a_label_holds", func(t *testing.T) {
		t.Parallel()

		out := renderDOT(t, visualize.Graph{Nodes: []visualize.Node{{ID: 0, Label: "main.Shape(\"a\")\nsingleton"}}})

		assert.Contains(t, out, `n0 [label = "main.Shape(\"a\")\nsingleton"];`)
		assert.Equal(t, 1, strings.Count(out, "\n\tn0"), "a newline in a label must not break the line")
	})

	t.Run("returns_the_error_of_the_writer", func(t *testing.T) {
		t.Parallel()

		err := visualize.DOT{}.Render(failingWriter{}, visualize.Graph{Name: "container"})

		assert.ErrorIs(t, err, errWriteFailed)
	})
}

var errWriteFailed = errors.New("write failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}
