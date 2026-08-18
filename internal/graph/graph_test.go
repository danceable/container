package graph_test

import (
	"testing"

	"github.com/danceable/container/internal/graph"
	"github.com/stretchr/testify/assert"
)

// adjacency is a Graph built straight from an adjacency list: node u has an edge to
// every node listed in adjacency[u].
type adjacency [][]int

var _ graph.Graph = adjacency{}

func (a adjacency) Order() int {
	return len(a)
}

func (a adjacency) EdgesFrom(u int) []int {
	return a[u]
}

func TestIsAcyclic(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		graph   adjacency
		acyclic bool
		cycle   []int
	}{
		"empty_graph": {
			graph:   adjacency{},
			acyclic: true,
		},
		"single_node_without_edges": {
			graph:   adjacency{nil},
			acyclic: true,
		},
		"self_loop": {
			graph:   adjacency{{0}},
			acyclic: false,
			cycle:   []int{0, 0},
		},
		"chain": {
			graph:   adjacency{{1}, {2}, nil},
			acyclic: true,
		},
		"two_node_cycle": {
			graph:   adjacency{{1}, {0}},
			acyclic: false,
			cycle:   []int{0, 1, 0},
		},
		"three_node_cycle": {
			graph:   adjacency{{1}, {2}, {0}},
			acyclic: false,
			cycle:   []int{0, 1, 2, 0},
		},
		"diamond_is_not_a_cycle": {
			// 0 → 1 → 3 and 0 → 2 → 3: node 3 is reached twice, through two routes.
			graph:   adjacency{{1, 2}, {3}, {3}, nil},
			acyclic: true,
		},
		"cycle_behind_a_prefix": {
			// The cycle is 1 → 2 → 1, entered through 0, which is not part of it.
			graph:   adjacency{{1}, {2}, {1}},
			acyclic: false,
			cycle:   []int{1, 2, 1},
		},
		"cycle_in_a_disconnected_component": {
			// Nothing links 0 to the cycle 1 → 2 → 1: only walking every node finds it.
			graph:   adjacency{nil, {2}, {1}},
			acyclic: false,
			cycle:   []int{1, 2, 1},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			acyclic, cycle := graph.IsAcyclic(test.graph)

			assert.Equal(t, test.acyclic, acyclic)
			assert.Equal(t, test.cycle, cycle)
		})
	}
}

func TestIsAcyclicFrom(t *testing.T) {
	t.Parallel()

	t.Run("finds_the_cycle_the_node_takes_part_in", func(t *testing.T) {
		t.Parallel()

		acyclic, cycle := graph.IsAcyclicFrom(adjacency{{1}, {2}, {0}}, 1)

		assert.False(t, acyclic)
		assert.Equal(t, []int{1, 2, 0, 1}, cycle)
	})

	t.Run("finds_a_cycle_reachable_from_the_node", func(t *testing.T) {
		t.Parallel()

		// 0 leads into the cycle 1 → 2 → 1 without being part of it.
		acyclic, cycle := graph.IsAcyclicFrom(adjacency{{1}, {2}, {1}}, 0)

		assert.False(t, acyclic)
		assert.Equal(t, []int{1, 2, 1}, cycle)
	})

	t.Run("walks_the_edges_of_an_acyclic_subgraph", func(t *testing.T) {
		t.Parallel()

		// 0 → 1 → 3 and 0 → 2 → 3: edges to follow, and no cycle at the end of them.
		acyclic, cycle := graph.IsAcyclicFrom(adjacency{{1, 2}, {3}, {3}, nil}, 0)

		assert.True(t, acyclic)
		assert.Nil(t, cycle)
	})

	t.Run("ignores_a_cycle_the_node_cannot_reach", func(t *testing.T) {
		t.Parallel()

		// The cycle 1 → 2 → 1 exists, but nothing leads to it from 0.
		acyclic, cycle := graph.IsAcyclicFrom(adjacency{nil, {2}, {1}}, 0)

		assert.True(t, acyclic)
		assert.Nil(t, cycle)
	})

	t.Run("acyclic_when_the_node_is_out_of_range", func(t *testing.T) {
		t.Parallel()

		g := adjacency{{0}}

		acyclic, cycle := graph.IsAcyclicFrom(g, 1)
		assert.True(t, acyclic)
		assert.Nil(t, cycle)

		acyclic, cycle = graph.IsAcyclicFrom(g, -1)
		assert.True(t, acyclic)
		assert.Nil(t, cycle)
	})
}
