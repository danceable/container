package container

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/danceable/container/bind"
	dotpkg "github.com/danceable/container/internal/dot"
	"github.com/danceable/container/internal/registerar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cycleShape interface{ area() int }

type cycleCircle struct{}

func (cycleCircle) area() int { return 1 }

type cycleDatabase interface{ connect() bool }

type cycleMySQL struct{}

func (cycleMySQL) connect() bool { return true }

type cycleLogger interface{ log() string }

type cycleStdLogger struct{}

func (cycleStdLogger) log() string { return "logged" }

// TestDependencyGraph_DrawsACycleInRed closes a cycle by hand, since a container that
// holds up its end never produces one for the drawing to show.
func TestDependencyGraph_DrawsACycleInRed(t *testing.T) {
	t.Parallel()

	c := New()
	require.NoError(t, c.Bind(func() cycleShape { return cycleCircle{} }, bind.Lazy()))
	require.NoError(t, c.Bind(func(s cycleShape) cycleDatabase { return cycleMySQL{} }, bind.Lazy()))
	// The Logger leads into the cycle without taking part in it.
	require.NoError(t, c.Bind(func(d cycleDatabase) cycleLogger { return cycleStdLogger{} }, bind.Lazy()))

	g := newScopeGraph(c)

	shape := g.ids[scopeSlot{scope: c, slot: registerar.Slot{Type: reflect.TypeFor[cycleShape]()}}]
	database := g.ids[scopeSlot{scope: c, slot: registerar.Slot{Type: reflect.TypeFor[cycleDatabase]()}}]
	logger := g.ids[scopeSlot{scope: c, slot: registerar.Slot{Type: reflect.TypeFor[cycleLogger]()}}]
	require.Equal(t, []int{shape}, g.EdgesFrom(database), "the Database depends on the Shape")

	// Closing the loop: the Shape now depends on the Database in turn.
	g.nodes[shape].edges = append(g.nodes[shape].edges, database)

	var buf bytes.Buffer
	require.NoError(t, dotpkg.Write(&buf, g.drawing()))
	dot := buf.String()

	assert.Contains(t, dot, fmt.Sprintf("n%d [label = \"container.cycleShape\\ntransient\", color = red, fontcolor = red];", shape))
	assert.Contains(t, dot, fmt.Sprintf("n%d [label = \"container.cycleDatabase\\ntransient\", color = red, fontcolor = red];", database))
	assert.Contains(t, dot, fmt.Sprintf("n%d -> n%d [color = red, penwidth = 2];", shape, database))
	assert.Contains(t, dot, fmt.Sprintf("n%d -> n%d [color = red, penwidth = 2];", database, shape))

	// Only what the cycle runs through is drawn in red.
	assert.Contains(t, dot, fmt.Sprintf("n%d [label = \"container.cycleLogger\\ntransient\"];", logger))
	assert.Contains(t, dot, fmt.Sprintf("n%d -> n%d;", logger, database))
}

// TestContainer_Label checks the naming of the scopes a graph is drawn for.
func TestContainer_Label(t *testing.T) {
	t.Parallel()

	root := New()

	assert.Equal(t, "root", scopeLabel(root))
	assert.Equal(t, `scope "request"`, scopeLabel(root.Scope("request")))
	assert.Equal(t, "derived scope", scopeLabel(root.Derive()))
}

// TestContainer_VisualizeDerivedScope checks that a nameless scope is drawn too, with
// the ancestors it resolves from.
func TestContainer_VisualizeDerivedScope(t *testing.T) {
	t.Parallel()

	root := New()
	require.NoError(t, root.Bind(func() cycleDatabase { return cycleMySQL{} }, bind.Lazy()))

	derived := root.Derive()
	require.NoError(t, derived.Bind(func(d cycleDatabase) cycleShape { return cycleCircle{} }, bind.Lazy()))

	var buf bytes.Buffer
	require.NoError(t, derived.Visualize(&buf))
	dot := buf.String()

	assert.Contains(t, dot, `label = "root";`)
	assert.Contains(t, dot, `label = "derived scope";`)
	assert.Equal(t, 1, strings.Count(dot, " -> "), "the derived scope resolves the Database from the root")
}
