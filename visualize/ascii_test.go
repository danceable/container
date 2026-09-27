package visualize_test

import (
	"bytes"
	"testing"

	"github.com/danceable/container/visualize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderASCII(t *testing.T, r visualize.ASCII, g visualize.Graph) string {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, g))

	return buf.String()
}

func TestASCII(t *testing.T) {
	t.Parallel()

	t.Run("renders_an_empty_graph", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "(empty graph)\n", renderASCII(t, visualize.ASCII{}, visualize.Graph{Name: "empty"}))
	})

	t.Run("renders_a_tree_per_cluster_with_the_edges_under_their_node", func(t *testing.T) {
		t.Parallel()

		out := renderASCII(t, visualize.ASCII{}, visualize.Graph{
			Clusters: []visualize.Cluster{
				{Label: "root", Nodes: []visualize.Node{
					{ID: 0, Label: "main.Database\nsingleton, resolved"},
					{ID: 1, Label: "main.Shape\ntransient"},
				}},
				{Label: `scope "request"`, Nodes: []visualize.Node{{ID: 2, Label: "main.Logger\nsingleton"}}},
			},
			Nodes: []visualize.Node{{ID: 3, Label: "main.Cache\nunsatisfied", Style: visualize.Dashed}},
			Edges: []visualize.Edge{{From: 1, To: 0}, {From: 2, To: 0}, {From: 2, To: 3}},
		})

		assert.Equal(t, "root\n"+
			"|-- main.Database (singleton, resolved)\n"+
			"`-- main.Shape (transient)\n"+
			"    `-> main.Database\n"+
			"\n"+
			"scope \"request\"\n"+
			"`-- main.Logger (singleton)\n"+
			"    |-> main.Database [root]\n"+
			"    `-> main.Cache\n"+
			"\n"+
			"main.Cache (unsatisfied)\n", out)
	})

	t.Run("marks_what_is_highlighted", func(t *testing.T) {
		t.Parallel()

		out := renderASCII(t, visualize.ASCII{}, visualize.Graph{
			Clusters: []visualize.Cluster{{Label: "root", Nodes: []visualize.Node{
				{ID: 0, Label: "a", Style: visualize.Highlighted},
				{ID: 1, Label: "b", Style: visualize.Highlighted},
			}}},
			Edges: []visualize.Edge{{From: 0, To: 1, Style: visualize.Highlighted}, {From: 1, To: 0, Style: visualize.Highlighted}},
		})

		assert.Equal(t, "root\n"+
			"|-- a (!)\n"+
			"|   `=> b (!)\n"+
			"`-- b (!)\n"+
			"    `=> a (!)\n", out)
	})

	t.Run("colors_only_when_asked_to", func(t *testing.T) {
		t.Parallel()

		g := visualize.Graph{Nodes: []visualize.Node{
			{ID: 0, Label: "cycle", Style: visualize.Highlighted},
			{ID: 1, Label: "missing", Style: visualize.Dashed},
			{ID: 2, Label: "plain"},
		}}

		assert.NotContains(t, renderASCII(t, visualize.ASCII{}, g), "\x1b[")

		out := renderASCII(t, visualize.ASCII{Color: true}, g)
		assert.Contains(t, out, "\x1b[31mcycle (!)\x1b[0m\n")
		assert.Contains(t, out, "\x1b[2mmissing\x1b[0m\n")
		assert.Contains(t, out, "\nplain\n")
	})

	t.Run("returns_the_error_of_the_writer", func(t *testing.T) {
		t.Parallel()

		err := visualize.ASCII{}.Render(failingWriter{}, visualize.Graph{})

		assert.ErrorIs(t, err, errWriteFailed)
	})
}
