package container_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/danceable/container"
	"github.com/danceable/container/bind"
	"github.com/danceable/container/visualize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nodePattern matches a node declaration of the DOT output, capturing its number and
// its label, e.g. `n1 [label = "container_test.Shape\nsingleton"];`.
var nodePattern = regexp.MustCompile(`\bn(\d+) \[label = "([^"]*)"`)

// nodes returns the node number of every node in the DOT output, keyed by the type and
// name on the first line of its label.
func nodes(t *testing.T, dot string) map[string]string {
	t.Helper()

	numbers := make(map[string]string)
	for _, match := range nodePattern.FindAllStringSubmatch(dot, -1) {
		label, _, _ := strings.Cut(match[2], `\n`)
		numbers[label] = "n" + match[1]
	}

	return numbers
}

// requireEdge asserts that the DOT output holds an edge between the two nodes and
// returns the line declaring it.
func requireEdge(t *testing.T, dot, from, to string) string {
	t.Helper()

	for line := range strings.Lines(dot) {
		if strings.HasPrefix(strings.TrimSpace(line), from+" -> "+to) {
			return strings.TrimSpace(line)
		}
	}

	require.Failf(t, "missing edge", "expected an edge %s -> %s in:\n%s", from, to, dot)

	return ""
}

// render renders the container and returns the DOT output.
func render(t *testing.T, c *container.Container) string {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, c.Visualize(&buf))

	return buf.String()
}

func TestContainer_Visualize(t *testing.T) {
	t.Parallel()

	t.Run("empty_container_renders_an_empty_graph", func(t *testing.T) {
		t.Parallel()

		dot := render(t, container.New())

		assert.True(t, strings.HasPrefix(dot, "digraph container {\n"))
		assert.True(t, strings.HasSuffix(dot, "}\n"))
		assert.NotContains(t, dot, "subgraph")
		assert.NotContains(t, dot, "->")
	})

	t.Run("renders_a_node_per_binding_in_a_scope_cluster", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Logger { return StdLogger{} }, bind.Singleton()))
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.Lazy()))

		dot := render(t, c)

		assert.Contains(t, dot, "subgraph cluster_0 {")
		assert.Contains(t, dot, `label = "root";`)
		// The eager singleton is already built, the lazy transient is not.
		assert.Contains(t, dot, `label = "container_test.Logger\nsingleton, resolved"`)
		assert.Contains(t, dot, `label = "container_test.Shape\ntransient"`)
	})

	t.Run("names_the_binding_it_draws", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.WithName("circle"), bind.Singleton(), bind.Lazy()))

		dot := render(t, c)

		assert.Contains(t, dot, `label = "container_test.Shape(\"circle\")\nsingleton"`)
	})

	t.Run("draws_an_edge_per_dependency", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Logger { return StdLogger{} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func(l Logger) Shape { return &Circle{a: 1} }, bind.Lazy()))
		require.NoError(t, c.Bind(func(l Logger, s Shape) Database { return MySQL{} }, bind.Lazy()))

		dot := render(t, c)
		n := nodes(t, dot)

		requireEdge(t, dot, n["container_test.Shape"], n["container_test.Logger"])
		requireEdge(t, dot, n["container_test.Database"], n["container_test.Logger"])
		requireEdge(t, dot, n["container_test.Database"], n["container_test.Shape"])
	})

	t.Run("draws_a_dependency_bound_to_a_parameter_as_unsatisfied", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func(area int) Shape { return &Circle{a: area} }, bind.Lazy(), bind.ResolveDepenenciesByParams(10)))
		require.NoError(t, c.Bind(func(missing Logger) Database { return MySQL{} }, bind.Lazy()))

		dot := render(t, c)
		n := nodes(t, dot)

		// The int is supplied at bind time, so it is not a dependency at all.
		assert.NotContains(t, n, "int")

		// The Logger is nowhere to be found: it gets a node of its own, drawn dashed.
		assert.Contains(t, dot, `label = "container_test.Logger\nunsatisfied", style = dashed`)
		requireEdge(t, dot, n["container_test.Database"], n["container_test.Logger"])
	})

	t.Run("shares_one_unsatisfied_node_between_the_bindings_needing_it", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func(l Logger) Shape { return &Circle{a: 1} }, bind.Lazy()))
		require.NoError(t, c.Bind(func(l Logger) Database { return MySQL{} }, bind.Lazy()))

		dot := render(t, c)
		n := nodes(t, dot)

		assert.Equal(t, 1, strings.Count(dot, "unsatisfied"))
		requireEdge(t, dot, n["container_test.Shape"], n["container_test.Logger"])
		requireEdge(t, dot, n["container_test.Database"], n["container_test.Logger"])
	})

	t.Run("draws_a_cluster_per_scope_and_edges_across_them", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Database { return MySQL{} }, bind.Singleton(), bind.Lazy()))

		request := c.Scope("request")
		require.NoError(t, request.Bind(func(d Database) Service { return AppService{} }, bind.Lazy()))

		dot := render(t, c)
		n := nodes(t, dot)

		assert.Contains(t, dot, `label = "root";`)
		assert.Contains(t, dot, `label = "scope \"request\"";`)
		// The scope resolves the Database from its parent, so the edge leaves the cluster.
		requireEdge(t, dot, n["container_test.Service"], n["container_test.Database"])
	})

	t.Run("includes_the_ancestors_of_the_scope_it_is_called_on", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Database { return MySQL{} }, bind.Singleton(), bind.Lazy()))

		request := c.Scope("request")
		require.NoError(t, request.Bind(func(d Database) Service { return AppService{} }, bind.Lazy()))

		dot := render(t, request)
		n := nodes(t, dot)

		// The parent is drawn as well, so the dependency has somewhere to point at.
		assert.Contains(t, dot, `label = "root";`)
		requireEdge(t, dot, n["container_test.Service"], n["container_test.Database"])
	})

	t.Run("leaves_out_the_scopes_holding_no_binding", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		c.Scope("empty")
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.Lazy()))

		dot := render(t, c)

		assert.NotContains(t, dot, `label = "scope \"empty\"";`)
		assert.Equal(t, 1, strings.Count(dot, "subgraph"))
	})

	t.Run("renders_the_same_container_the_same_way", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Logger { return StdLogger{} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func(l Logger) Shape { return &Circle{a: 1} }, bind.Lazy()))

		for _, name := range []string{"b", "a", "c"} {
			require.NoError(t, c.Scope(name).Bind(func(l Logger) Cache { return InMemoryCache{} }, bind.Lazy()))
		}

		dot := render(t, c)
		assert.Equal(t, dot, render(t, c), "the same container must render the same graph")

		// The scopes are ordered by name, not by the order they were created in.
		assert.Less(t, strings.Index(dot, `scope \"a\"`), strings.Index(dot, `scope \"b\"`))
		assert.Less(t, strings.Index(dot, `scope \"b\"`), strings.Index(dot, `scope \"c\"`))
	})

	t.Run("returns_the_error_of_the_writer", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.Lazy()))

		err := c.Visualize(failingWriter{})
		assert.ErrorIs(t, err, errWriteFailed)
	})
}

func TestContainer_VisualizeWithRenderer(t *testing.T) {
	t.Parallel()

	c := container.New()
	require.NoError(t, c.Bind(func() Logger { return StdLogger{} }, bind.Singleton(), bind.Lazy()))
	require.NoError(t, c.Bind(func(l Logger) Shape { return &Circle{a: 1} }, bind.Lazy()))

	var buf bytes.Buffer
	require.NoError(t, c.Visualize(&buf, visualize.WithRenderer(visualize.ASCII{})))

	assert.Equal(t, "root\n"+
		"|-- container_test.Logger (singleton)\n"+
		"`-- container_test.Shape (transient)\n"+
		"    `-> container_test.Logger\n", buf.String())

	// A nil renderer keeps the default one.
	buf.Reset()
	require.NoError(t, c.Visualize(&buf, visualize.WithRenderer(nil)))
	assert.True(t, strings.HasPrefix(buf.String(), "digraph container {\n"))
}

func TestContainer_VisualizeHandler(t *testing.T) {
	t.Parallel()

	serve := func(h http.Handler) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/debug/container", nil))

		return rec
	}

	c := container.New()
	require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.Lazy()))

	t.Run("serves_an_html_page_by_default", func(t *testing.T) {
		t.Parallel()

		rec := serve(c.VisualizeHandler())

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Body.String(), "<!doctype html>")
		assert.Contains(t, rec.Body.String(), "container_test.Shape (transient)")
	})

	t.Run("serves_what_the_renderer_given_draws", func(t *testing.T) {
		t.Parallel()

		rec := serve(c.VisualizeHandler(visualize.WithRenderer(visualize.DOT{})))

		assert.Equal(t, "text/vnd.graphviz; charset=utf-8", rec.Header().Get("Content-Type"))
		assert.True(t, strings.HasPrefix(rec.Body.String(), "digraph container {\n"))
	})

	t.Run("draws_the_bindings_of_the_moment", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		h := c.VisualizeHandler(visualize.WithRenderer(visualize.ASCII{}))
		assert.Equal(t, "(empty graph)\n", serve(h).Body.String())

		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.Lazy()))
		assert.Contains(t, serve(h).Body.String(), "container_test.Shape")
	})

	t.Run("answers_with_an_error_when_the_renderer_fails", func(t *testing.T) {
		t.Parallel()

		rec := serve(c.VisualizeHandler(visualize.WithRenderer(failingRenderer{})))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), errWriteFailed.Error())
	})
}

func TestVisualize(t *testing.T) {
	container.Reset()
	defer container.Reset()

	require.NoError(t, container.Bind(func() Shape { return &Circle{a: 1} }, bind.Lazy()))

	var buf bytes.Buffer
	require.NoError(t, container.Visualize(&buf))

	assert.Contains(t, buf.String(), `label = "container_test.Shape\ntransient"`)

	rec := httptest.NewRecorder()
	container.VisualizeHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Contains(t, rec.Body.String(), "container_test.Shape (transient)")
}

// errWriteFailed is the error the failingWriter fails with.
var errWriteFailed = errors.New("write failed")

// failingWriter is an io.Writer that fails on every write.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

// failingRenderer is a visualize.Renderer that fails on every graph.
type failingRenderer struct{}

func (failingRenderer) Render(io.Writer, visualize.Graph) error {
	return errWriteFailed
}

func (failingRenderer) ContentType() string {
	return "text/plain"
}
