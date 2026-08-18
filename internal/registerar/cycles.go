package registerar

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/danceable/container/errors"
	"github.com/danceable/container/internal/graph"
)

// acyclic guards a store against registrations that would close a cycle.
type acyclic struct {
	store *store
}

// set registers the binding and rolls it back when it closes a cycle. Registering comes
// first because the binding is the node closing that cycle. Walking from the new node
// covers every cycle that can be new, every earlier registration having passed here too.
func (a acyclic) set(slot Slot, b *Binding) error {
	node, previous, replaced := a.store.put(slot, b)

	g := dependencyGraph{store: a.store}
	if acyclic, cycle := graph.IsAcyclicFrom(g, node); !acyclic {
		path := g.pathOf(cycle) // while the nodes of the cycle are still there
		a.store.restore(slot, previous, replaced)

		return fmt.Errorf("%w: %s", errors.ErrCircularDependency, path)
	}

	return nil
}

// dependencyGraph views the registrations of a store as a directed graph, where an edge
// from u to v means the binding in slot u needs the one in slot v. Edges are derived on
// demand, so it may only be used while the store is held still.
type dependencyGraph struct {
	store *store
}

var _ graph.Graph = dependencyGraph{}

func (g dependencyGraph) Order() int {
	return len(g.store.nodes)
}

// EdgesFrom returns the nodes holding the dependencies of the binding in node u. A
// dependency no registration satisfies is resolved outside the graph, so it is no edge.
func (g dependencyGraph) EdgesFrom(u int) []int {
	reg, exist := g.store.registration(g.store.nodes[u])
	if !exist {
		return nil
	}

	var edges []int
	for _, dependency := range reg.binding.Dependencies() {
		if target, exist := g.dependencyNode(dependency, reg.binding); exist {
			edges = append(edges, target)
		}
	}

	return edges
}

// dependencyNode returns the node satisfying the given dependency of b, following the
// order the container resolves an argument in.
func (g dependencyGraph) dependencyNode(dependency reflect.Type, b *Binding) (int, bool) {
	for name := range b.DependencyNames() {
		if _, reg, exist := g.store.lookup(dependency, name); exist {
			return reg.node, true
		}
	}

	return 0, false
}

// pathOf renders a path as `main.Shape -> main.Database -> main.Shape`.
func (g dependencyGraph) pathOf(nodes []int) string {
	var path strings.Builder

	for i, u := range nodes {
		if i > 0 {
			path.WriteString(" -> ")
		}

		path.WriteString(g.store.nodes[u].String())
	}

	return path.String()
}
