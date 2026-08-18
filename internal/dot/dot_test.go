package dot_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/danceable/container/internal/dot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func render(t *testing.T, g dot.Graph) string {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, dot.Write(&buf, g))

	return buf.String()
}

func TestWrite(t *testing.T) {
	t.Parallel()

	t.Run("renders_an_empty_graph", func(t *testing.T) {
		t.Parallel()

		out := render(t, dot.Graph{Name: "empty"})

		assert.Equal(t, "digraph empty {\n"+
			"\trankdir = LR;\n"+
			"\tnode [shape = box, style = rounded, fontname = \"Helvetica\"];\n"+
			"\tedge [fontname = \"Helvetica\"];\n"+
			"}\n", out)
	})

	t.Run("renders_clusters_with_their_nodes", func(t *testing.T) {
		t.Parallel()

		out := render(t, dot.Graph{
			Name: "container",
			Clusters: []dot.Cluster{
				{Label: "root", Nodes: []dot.Node{{ID: 0, Label: "main.Shape"}}},
				{Label: `scope "request"`, Nodes: []dot.Node{{ID: 1, Label: "main.Logger"}}},
			},
		})

		assert.Contains(t, out, "\tsubgraph cluster_0 {\n\t\tlabel = \"root\";\n")
		assert.Contains(t, out, "\t\tn0 [label = \"main.Shape\"];\n")
		assert.Contains(t, out, "\tsubgraph cluster_1 {\n\t\tlabel = \"scope \\\"request\\\"\";\n")
		assert.Contains(t, out, "\t\tn1 [label = \"main.Logger\"];\n")
	})

	t.Run("renders_the_style_of_a_node", func(t *testing.T) {
		t.Parallel()

		out := render(t, dot.Graph{
			Nodes: []dot.Node{
				{ID: 0, Label: "solid"},
				{ID: 1, Label: "dashed", Style: dot.Dashed},
				{ID: 2, Label: "highlighted", Style: dot.Highlighted},
			},
		})

		assert.Contains(t, out, "\tn0 [label = \"solid\"];\n")
		assert.Contains(t, out, "\tn1 [label = \"dashed\", style = dashed, color = \"#909090\", fontcolor = \"#606060\"];\n")
		assert.Contains(t, out, "\tn2 [label = \"highlighted\", color = red, fontcolor = red];\n")
	})

	t.Run("renders_the_style_of_an_edge", func(t *testing.T) {
		t.Parallel()

		out := render(t, dot.Graph{
			Edges: []dot.Edge{
				{From: 0, To: 1},
				{From: 1, To: 0, Style: dot.Highlighted},
			},
		})

		assert.Contains(t, out, "\tn0 -> n1;\n")
		assert.Contains(t, out, "\tn1 -> n0 [color = red, penwidth = 2];\n")
	})

	t.Run("quotes_what_a_label_holds", func(t *testing.T) {
		t.Parallel()

		out := render(t, dot.Graph{Nodes: []dot.Node{{ID: 0, Label: "main.Shape(\"a\")\nsingleton"}}})

		assert.Contains(t, out, `n0 [label = "main.Shape(\"a\")\nsingleton"];`)
		assert.Equal(t, 1, strings.Count(out, "\n\tn0"), "a newline in a label must not break the line")
	})

	t.Run("returns_the_error_of_the_writer", func(t *testing.T) {
		t.Parallel()

		err := dot.Write(failingWriter{}, dot.Graph{Name: "container"})

		assert.ErrorIs(t, err, errWriteFailed)
	})
}

var errWriteFailed = errors.New("write failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}
