package visualize_test

import (
	"bytes"
	"testing"

	"github.com/danceable/container/visualize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTML(t *testing.T) {
	t.Parallel()

	g := visualize.Graph{
		Name:     "container",
		Clusters: []visualize.Cluster{{Label: "root", Nodes: []visualize.Node{{ID: 0, Label: "main.Shape<T>\ntransient"}}}},
	}

	t.Run("renders_a_page_drawing_the_graph", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		require.NoError(t, visualize.HTML{}.Render(&buf, g))
		out := buf.String()

		assert.Contains(t, out, "<title>Dependency graph</title>")
		assert.Contains(t, out, "viz-standalone.js")
		// The DOT source goes to the script as a string literal, the ASCII tree to the
		// fallback, both escaped.
		assert.Contains(t, out, `viz.renderSVGElement("digraph container {\n`)
		assert.Contains(t, out, "`-- main.Shape&lt;T&gt; (transient)")
		assert.NotContains(t, out, "main.Shape<T>")
	})

	t.Run("uses_the_title_given", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		require.NoError(t, visualize.HTML{Title: "My app"}.Render(&buf, g))

		assert.Contains(t, buf.String(), "<title>My app</title>")
	})

	t.Run("returns_the_error_of_the_writer", func(t *testing.T) {
		t.Parallel()

		err := visualize.HTML{}.Render(failingWriter{}, g)

		assert.ErrorIs(t, err, errWriteFailed)
	})
}

func TestContentType(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "text/vnd.graphviz; charset=utf-8", visualize.DOT{}.ContentType())
	assert.Equal(t, "text/plain; charset=utf-8", visualize.ASCII{}.ContentType())
	assert.Equal(t, "text/html; charset=utf-8", visualize.HTML{}.ContentType())
}
