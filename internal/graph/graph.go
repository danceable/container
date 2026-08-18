// Package graph represents directed graphs and detects cycles in them.
//
// Nothing here knows what the nodes stand for: a caller exposes its own structure
// through the Graph interface by numbering its nodes.
package graph

// Graph is a directed graph whose nodes are the integers in the range [0, Order).
type Graph interface {
	// Order returns the number of nodes in the graph.
	Order() int

	// EdgesFrom returns the nodes that node u has an outgoing edge to.
	EdgesFrom(u int) []int
}

// IsAcyclic reports whether the graph is free of cycles, returning the nodes of the
// cycle it finds in traversal order, closing node repeated last.
func IsAcyclic(g Graph) (bool, []int) {
	order := g.Order()
	state := newVisit(order)

	for u := range order {
		state.unstack()

		if cycle := state.walk(g, u, nil); len(cycle) > 0 {
			return false, cycle
		}
	}

	return true, nil
}

// IsAcyclicFrom reports on the part of the graph reachable from node u. Adding a node to
// an acyclic graph only closes cycles running through it, so a caller checking each node
// as it is added never walks the whole graph.
func IsAcyclicFrom(g Graph, u int) (bool, []int) {
	if u < 0 || u >= g.Order() {
		return true, nil
	}

	if len(g.EdgesFrom(u)) == 0 {
		return true, nil
	}

	if cycle := newVisit(g.Order()).walk(g, u, nil); len(cycle) > 0 {
		return false, cycle
	}

	return true, nil
}

// node holds the depth-first search state of a single graph node.
type node struct {
	visited bool // visited is true once the node has been explored.
	onStack bool // onStack is true while the node is part of the current search path.
}

// visit holds the search state of every node, indexed by node.
type visit []node

func newVisit(order int) visit {
	return make(visit, order)
}

// unstack clears the search path, keeping the visited marks so that no node is ever
// explored twice.
func (v visit) unstack() {
	for i := range v {
		v[i].onStack = false
	}
}

// walk explores the graph depth-first from node u, where path holds the nodes leading to
// u, and returns the nodes of the first cycle it finds.
func (v visit) walk(g Graph, u int, path []int) []int {
	if v[u].visited {
		return nil
	}

	v[u].visited = true
	v[u].onStack = true

	path = append(path, u)

	for _, w := range g.EdgesFrom(u) {
		if !v[w].visited {
			if cycle := v.walk(g, w, path); len(cycle) > 0 {
				return cycle
			}

			continue
		}

		// Only an edge back into the current path closes a cycle; one to an explored node
		// off the path is a diamond.
		if v[w].onStack {
			return closeCycle(path, w)
		}
	}

	v[u].onStack = false

	return nil
}

// closeCycle trims the path down to the cycle the edge back to w closes, repeating w to
// spell it out.
func closeCycle(path []int, w int) []int {
	start := 0
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == w {
			start = i
			break
		}
	}

	cycle := make([]int, 0, len(path)-start+1)
	cycle = append(cycle, path[start:]...)

	return append(cycle, w)
}
