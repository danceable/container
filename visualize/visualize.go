// Package visualize draws the dependency graph of a container.
//
// The container describes its graph with the model below and hands it to a Renderer,
// which decides what the drawing looks like: Graphviz DOT, a text tree for the
// terminal, an HTML page, or anything else implementing the interface.
//
//	c.Visualize(os.Stdout, visualize.WithRenderer(visualize.ASCII{}))
package visualize

import "io"

// Renderer draws a graph and writes it to w.
type Renderer interface {
	Render(w io.Writer, g Graph) error

	// ContentType returns the media type of what Render writes, for serving it over HTTP.
	ContentType() string
}

var (
	_ Renderer = DOT{}
	_ Renderer = ASCII{}
	_ Renderer = HTML{}
)

// Style is how a node or an edge is drawn.
type Style uint8

const (
	Solid Style = iota
	Dashed
	Highlighted
)

// Node is a single box, identified by a number unique within the Graph. The first line
// of its label names it, the others describe it.
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

// options holds the configuration of a drawing.
type options struct {
	Renderer Renderer
}

// DefaultOptions returns the options a drawing starts from, fallback being the renderer
// used when none is given.
func DefaultOptions(fallback Renderer) *options {
	return &options{Renderer: fallback}
}

// Option is a functional option for configuring a drawing.
type Option func(*options)

// WithRenderer sets the renderer drawing the graph.
func WithRenderer(r Renderer) Option {
	return func(o *options) {
		if r != nil {
			o.Renderer = r
		}
	}
}
