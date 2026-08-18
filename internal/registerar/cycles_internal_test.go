package registerar

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type graphShape interface{ area() int }

type graphCircle struct{}

func (graphCircle) area() int { return 1 }

type graphDatabase interface{ connect() bool }

type graphMySQL struct{}

func (graphMySQL) connect() bool { return true }

// TestDependencyGraph_EdgesFromATombstone covers the node a deleted binding leaves
// behind, which no walk reaches because nothing points at it any more.
func TestDependencyGraph_EdgesFromATombstone(t *testing.T) {
	t.Parallel()

	r := NewRegisterar()
	shapeType := reflect.TypeFor[graphShape]()
	dbType := reflect.TypeFor[graphDatabase]()

	require.NoError(t, r.Set(shapeType, "", NewBinding("", false, nil, nil, func() graphShape { return graphCircle{} }, nil)))
	require.NoError(t, r.Set(dbType, "", NewBinding("", false, nil, nil, func(s graphShape) graphDatabase { return graphMySQL{} }, nil)))

	g := dependencyGraph{store: &r.store}
	require.Equal(t, 2, g.Order())
	require.Equal(t, []int{0}, g.EdgesFrom(1), "the Database depends on the Shape")

	// Deleting the Shape leaves its node in place, so that the Database keeps its own.
	r.Delete(shapeType, "")

	assert.Equal(t, 2, g.Order())
	assert.Nil(t, g.EdgesFrom(0), "the node of the deleted binding has no edges")
	assert.Nil(t, g.EdgesFrom(1), "the dependency is no longer registered, so it is no edge")
}

// TestRegistrar_ReusesTheNodeOfADeletedBinding checks that churn hands the same numbers
// out again instead of growing the graph.
func TestRegistrar_ReusesTheNodeOfADeletedBinding(t *testing.T) {
	t.Parallel()

	r := NewRegisterar()
	shapeType := reflect.TypeFor[graphShape]()
	dbType := reflect.TypeFor[graphDatabase]()

	require.NoError(t, r.Set(shapeType, "", NewBinding("", false, nil, nil, func() graphShape { return graphCircle{} }, nil)))
	require.NoError(t, r.Set(dbType, "", NewBinding("", false, nil, nil, func() graphDatabase { return graphMySQL{} }, nil)))

	for range 3 {
		// The Shape is not the last node, so its number is kept and handed out again.
		r.Delete(shapeType, "")
		require.NoError(t, r.Set(shapeType, "", NewBinding("", false, nil, nil, func() graphShape { return graphCircle{} }, nil)))

		assert.Equal(t, 2, dependencyGraph{store: &r.store}.Order())
		assert.Empty(t, r.store.free)
	}

	// The Database is the last node: deleting it drops the node altogether.
	r.Delete(dbType, "")
	assert.Equal(t, 1, dependencyGraph{store: &r.store}.Order())
}
