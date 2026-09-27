package container

import (
	"bytes"
	"io"
	"net/http"
	"reflect"
	"strconv"

	"github.com/danceable/container/internal/graph"
	"github.com/danceable/container/internal/registerar"
	"github.com/danceable/container/visualize"
)

// Visualize writes the dependency graph of the container to w, in the Graphviz DOT
// format unless another renderer is given:
//
//	var buf bytes.Buffer
//	if err := c.Visualize(&buf); err != nil { ... }
//	os.WriteFile("container.dot", buf.Bytes(), 0o600)
//
//	// dot -Tsvg container.dot -o container.svg
//
//	c.Visualize(os.Stdout, visualize.WithRenderer(visualize.ASCII{Color: true}))
//
// It covers the scope it is called on, the ancestors it resolves from and its named
// descendants: a cluster per scope, a node per binding, an edge per dependency. A
// dependency no binding satisfies is drawn dashed, a cycle red.
func (c *Container) Visualize(w io.Writer, opts ...visualize.Option) error {
	o := visualize.DefaultOptions(visualize.DOT{})
	for _, opt := range opts {
		opt(o)
	}

	return o.Renderer.Render(w, newScopeGraph(c).drawing())
}

// VisualizeHandler returns an http.Handler serving the dependency graph of the
// container, as an HTML page unless another renderer is given:
//
//	http.Handle("/debug/container", c.VisualizeHandler())
//
// The graph is drawn anew on every request, so it shows the bindings of the moment.
func (c *Container) VisualizeHandler(opts ...visualize.Option) http.Handler {
	o := visualize.DefaultOptions(visualize.HTML{})
	for _, opt := range opts {
		opt(o)
	}

	renderer := o.Renderer

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		if err := renderer.Render(&buf, newScopeGraph(c).drawing()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", renderer.ContentType())
		_, _ = buf.WriteTo(w)
	})
}

// scopeSlot identifies a node: a binding by its scope and slot, a dependency no binding
// satisfies by its type alone.
type scopeSlot struct {
	scope *Container
	slot  registerar.Slot
}

type scopeNode struct {
	scopeSlot
	binding *registerar.Binding // nil when no binding satisfies the node
	edges   []int
}

// scopeGraph is a snapshot of the bindings visible from a scope, taken once and never
// looking at the container again. Implementing graph.Graph lets the cycle detection that
// guards every Bind find the cycles this graph draws in red.
type scopeGraph struct {
	scopes []*Container // outermost first
	nodes  []scopeNode  // indexed by node number
	ids    map[scopeSlot]int
}

var _ graph.Graph = (*scopeGraph)(nil)

func (g *scopeGraph) Order() int {
	return len(g.nodes)
}

func (g *scopeGraph) EdgesFrom(u int) []int {
	return g.nodes[u].edges
}

func newScopeGraph(c *Container) *scopeGraph {
	g := &scopeGraph{scopes: c.visibleScopes(), ids: make(map[scopeSlot]int)}

	// Every node first: an edge can point at a scope not walked yet.
	registrations := make([][]registerar.Registration, len(g.scopes))
	for i, scope := range g.scopes {
		registrations[i] = scope.reg.Registrations()

		for _, registration := range registrations[i] {
			g.add(scopeNode{
				scopeSlot: scopeSlot{scope: scope, slot: registration.Slot},
				binding:   registration.Binding,
			})
		}
	}

	for i, scope := range g.scopes {
		for _, registration := range registrations[i] {
			u := g.ids[scopeSlot{scope: scope, slot: registration.Slot}]

			for _, dependency := range registration.Binding.Dependencies() {
				// Resolved before it is appended, since resolving it can grow g.nodes.
				edge := g.dependencyNode(scope, dependency, registration.Binding)
				g.nodes[u].edges = append(g.nodes[u].edges, edge)
			}
		}
	}

	return g
}

// dependencyNode returns the node satisfying the dependency of b when resolved from the
// given scope, following the same names and ancestors the container does. A dependency
// nothing satisfies gets a node of its own, shared by every binding needing that type.
func (g *scopeGraph) dependencyNode(from *Container, dependency reflect.Type, b *registerar.Binding) int {
	for name := range b.DependencyNames() {
		for scope := from; scope != nil; scope = scope.parent {
			slot, exist := scope.reg.FindSlot(dependency, name)
			if !exist {
				continue
			}

			if id, tracked := g.ids[scopeSlot{scope: scope, slot: slot}]; tracked {
				return id
			}
		}
	}

	return g.add(scopeNode{scopeSlot: scopeSlot{slot: registerar.Slot{Type: dependency}}})
}

// add numbers the node, or returns the number it already has.
func (g *scopeGraph) add(n scopeNode) int {
	if id, exist := g.ids[n.scopeSlot]; exist {
		return id
	}

	id := len(g.nodes)
	g.nodes = append(g.nodes, n)
	g.ids[n.scopeSlot] = id

	return id
}

// drawing turns the graph into the model a renderer draws, one cluster per scope
// holding bindings.
func (g *scopeGraph) drawing() visualize.Graph {
	cycle := g.cycleEdges()
	drawing := visualize.Graph{Name: "container"}

	for _, scope := range g.scopes {
		nodes := g.drawNodes(scope, cycle)
		if len(nodes) == 0 {
			continue
		}

		drawing.Clusters = append(drawing.Clusters, visualize.Cluster{Label: scopeLabel(scope), Nodes: nodes})
	}

	drawing.Nodes = g.drawNodes(nil, cycle)

	for u := range g.nodes {
		for _, v := range g.nodes[u].edges {
			drawing.Edges = append(drawing.Edges, visualize.Edge{From: u, To: v, Style: edgeStyle(cycle, u, v)})
		}
	}

	return drawing
}

// drawNodes returns the nodes of the given scope, or the scopeless ones when it is nil.
func (g *scopeGraph) drawNodes(scope *Container, cycle map[[2]int]bool) []visualize.Node {
	var nodes []visualize.Node

	for u := range g.nodes {
		if g.nodes[u].scope != scope {
			continue
		}

		nodes = append(nodes, visualize.Node{ID: u, Label: g.label(u), Style: g.nodeStyle(u, cycle)})
	}

	return nodes
}

// label returns what a node provides, and how.
func (g *scopeGraph) label(u int) string {
	n := g.nodes[u]

	if n.binding == nil {
		return n.slot.String() + "\nunsatisfied"
	}

	kind := "transient"
	if n.binding.IsSingleton() {
		kind = "singleton"
	}
	if n.binding.HasConcrete() {
		kind += ", resolved"
	}

	return n.slot.String() + "\n" + kind
}

func (g *scopeGraph) nodeStyle(u int, cycle map[[2]int]bool) visualize.Style {
	for edge := range cycle {
		if edge[0] == u || edge[1] == u {
			return visualize.Highlighted
		}
	}

	if g.nodes[u].binding == nil {
		return visualize.Dashed
	}

	return visualize.Solid
}

func edgeStyle(cycle map[[2]int]bool, u, v int) visualize.Style {
	if cycle[[2]int{u, v}] {
		return visualize.Highlighted
	}

	return visualize.Solid
}

// cycleEdges returns the edges of a cycle, if the graph has one at all.
func (g *scopeGraph) cycleEdges() map[[2]int]bool {
	acyclic, nodes := graph.IsAcyclic(g)
	if acyclic {
		return nil
	}

	edges := make(map[[2]int]bool, len(nodes))
	for i := 1; i < len(nodes); i++ {
		edges[[2]int{nodes[i-1], nodes[i]}] = true
	}

	return edges
}

// scopeLabel returns the name of the scope as it is drawn.
func scopeLabel(c *Container) string {
	switch {
	case c.parent == nil:
		return "root"
	case c.name == "":
		return "derived scope"
	default:
		return "scope " + strconv.Quote(c.name)
	}
}
