package registerar_test

import (
	"reflect"
	"slices"
	"sync"
	"testing"

	"github.com/danceable/container/errors"
	"github.com/danceable/container/internal/registerar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testShape interface {
	Area() int
}

type testCircle struct {
	R int
}

func (c *testCircle) Area() int {
	return c.R * c.R
}

type testDatabase interface {
	Name() string
}

type testMySQL struct{}

func (m *testMySQL) Name() string {
	return "mysql"
}

// testSquare is a second implementation of testShape.
type testSquare struct {
	S int
}

func (s *testSquare) Area() int {
	return s.S * s.S
}

type testLogger interface {
	Log()
}

func TestNewRegisterar(t *testing.T) {
	t.Parallel()

	r := registerar.NewRegisterar()
	assert.NotNil(t, r)
}

func TestRegistrar_Reset(t *testing.T) {
	t.Parallel()

	t.Run("clears_all_bindings", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		require.NoError(t, r.Set(typ, "", b))

		_, found := r.Get(typ, "")
		require.True(t, found)

		r.Reset()

		_, found = r.Get(typ, "")
		assert.False(t, found)
	})

	t.Run("reset_allows_re-registration", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		require.NoError(t, r.Set(typ, "", b))

		r.Reset()

		b2 := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{R: 99} }, nil)
		assert.NoError(t, r.Set(typ, "", b2))
	})
}

func TestRegistrar_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes_existing_binding", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		require.NoError(t, r.Set(typ, "", b))

		r.Delete(typ, "")

		_, found := r.Get(typ, "")
		assert.False(t, found)
	})

	t.Run("removes_named_binding", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("circle", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		require.NoError(t, r.Set(typ, "circle", b))

		r.Delete(typ, "circle")

		_, found := r.Get(typ, "circle")
		assert.False(t, found)
	})

	t.Run("noop_for_nonexistent_type", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		r.Delete(typ, "") // should not panic
	})

	t.Run("noop_for_nonexistent_name", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("a", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		require.NoError(t, r.Set(typ, "a", b))

		r.Delete(typ, "b") // different name

		_, found := r.Get(typ, "a")
		assert.True(t, found, "original binding should still exist")
	})
}

func TestRegistrar_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns_binding_by_exact_type", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		require.NoError(t, r.Set(typ, "", b))

		got, found := r.Get(typ, "")
		assert.True(t, found)
		assert.Equal(t, b, got)
	})

	t.Run("returns_named_binding", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("myShape", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		require.NoError(t, r.Set(typ, "myShape", b))

		got, found := r.Get(typ, "myShape")
		assert.True(t, found)
		assert.Equal(t, b, got)
	})

	t.Run("returns_false_for_unknown_type", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()

		_, found := r.Get(typ, "")
		assert.False(t, found)
	})

	t.Run("returns_false_for_wrong_name", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("alpha", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		require.NoError(t, r.Set(typ, "alpha", b))

		_, found := r.Get(typ, "beta")
		assert.False(t, found)
	})

	t.Run("does_not_fall_back_to_interface_implementation", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		concreteType := reflect.TypeFor[*testCircle]()
		b := registerar.NewBinding("", false, nil, nil, func() *testCircle { return &testCircle{} }, nil)
		require.NoError(t, r.Set(concreteType, "", b))

		ifaceType := reflect.TypeFor[testShape]()
		_, found := r.Get(ifaceType, "")
		assert.False(t, found, "Get should not do interface matching")
	})
}

func TestRegistrar_Find(t *testing.T) {
	t.Parallel()

	t.Run("finds_by_exact_type", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		require.NoError(t, r.Set(typ, "", b))

		got, found := r.Find(typ, "")
		assert.True(t, found)
		assert.Equal(t, b, got)
	})

	t.Run("falls_back_to_interface_implementation", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		concreteType := reflect.TypeFor[*testCircle]()
		b := registerar.NewBinding("", false, nil, nil, func() *testCircle { return &testCircle{R: 42} }, nil)
		require.NoError(t, r.Set(concreteType, "", b))

		ifaceType := reflect.TypeFor[testShape]()
		got, found := r.Find(ifaceType, "")
		assert.True(t, found)
		assert.Equal(t, b, got)
	})

	t.Run("interface_fallback_respects_name", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		concreteType := reflect.TypeFor[*testCircle]()
		b := registerar.NewBinding("special", false, nil, nil, func() *testCircle { return &testCircle{} }, nil)
		require.NoError(t, r.Set(concreteType, "special", b))

		ifaceType := reflect.TypeFor[testShape]()

		got, found := r.Find(ifaceType, "special")
		assert.True(t, found)
		assert.Equal(t, b, got)

		_, found = r.Find(ifaceType, "other")
		assert.False(t, found)
	})

	t.Run("returns_false_for_unknown_type", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testDatabase]()

		_, found := r.Find(typ, "")
		assert.False(t, found)
	})

	t.Run("no_fallback_for_non_interface_type", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		// Bind the interface type
		ifaceType := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		require.NoError(t, r.Set(ifaceType, "", b))

		// Search by concrete type — should not match
		concreteType := reflect.TypeFor[*testCircle]()
		_, found := r.Find(concreteType, "")
		assert.False(t, found)
	})

	t.Run("exact_match_preferred_over_fallback", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		ifaceType := reflect.TypeFor[testShape]()
		concreteType := reflect.TypeFor[*testCircle]()

		bExact := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{R: 1} }, nil)
		require.NoError(t, r.Set(ifaceType, "", bExact))

		bImpl := registerar.NewBinding("", false, nil, nil, func() *testCircle { return &testCircle{R: 2} }, nil)
		require.NoError(t, r.Set(concreteType, "", bImpl))

		got, found := r.Find(ifaceType, "")
		assert.True(t, found)
		assert.Equal(t, bExact, got, "exact match should take precedence")
	})
}

func TestRegistrar_Set(t *testing.T) {
	t.Parallel()

	t.Run("stores_binding", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)

		err := r.Set(typ, "", b)
		assert.NoError(t, err)

		got, found := r.Get(typ, "")
		assert.True(t, found)
		assert.Equal(t, b, got)
	})

	t.Run("stores_named_binding", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("foo", false, nil, nil, func() testShape { return &testCircle{} }, nil)

		err := r.Set(typ, "foo", b)
		assert.NoError(t, err)

		got, found := r.Get(typ, "foo")
		assert.True(t, found)
		assert.Equal(t, b, got)
	})

	t.Run("overwrites_existing_binding", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b1 := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{R: 1} }, nil)
		b2 := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{R: 2} }, nil)

		require.NoError(t, r.Set(typ, "", b1))
		require.NoError(t, r.Set(typ, "", b2))

		got, found := r.Get(typ, "")
		assert.True(t, found)
		assert.Equal(t, b2, got)
	})

	t.Run("multiple_names_under_same_type", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b1 := registerar.NewBinding("a", false, nil, nil, func() testShape { return &testCircle{R: 1} }, nil)
		b2 := registerar.NewBinding("b", false, nil, nil, func() testShape { return &testCircle{R: 2} }, nil)

		require.NoError(t, r.Set(typ, "a", b1))
		require.NoError(t, r.Set(typ, "b", b2))

		got1, found1 := r.Get(typ, "a")
		got2, found2 := r.Get(typ, "b")

		assert.True(t, found1)
		assert.True(t, found2)
		assert.Equal(t, b1, got1)
		assert.Equal(t, b2, got2)
	})

	t.Run("detects_self_referencing_resolver", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("", false, nil, nil, func(s testShape) testShape { return s }, nil)

		err := r.Set(typ, "", b)
		assert.ErrorIs(t, err, errors.ErrCircularDependency)
	})

	t.Run("no_cycle_when_self_ref_satisfied_by_bind_params", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		params := []reflect.Value{reflect.ValueOf(&testCircle{R: 5})}
		b := registerar.NewBinding("", false, params, nil, func(s testShape) testShape { return s }, nil)

		err := r.Set(typ, "", b)
		assert.NoError(t, err)
	})

	t.Run("no_cycle_when_indirect_dep_satisfied_by_bind_params", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		// Register testDatabase depending on testShape.
		bDB := registerar.NewBinding("", false, nil, nil, func(s testShape) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "", bDB))

		// Register testShape depending on testDatabase, but testDatabase is supplied via bindParams.
		params := []reflect.Value{reflect.ValueOf(&testMySQL{})}
		bShape := registerar.NewBinding("", false, params, nil, func(d testDatabase) testShape { return &testCircle{} }, nil)
		assert.NoError(t, r.Set(shapeType, "", bShape))
	})

	t.Run("cycle_detected_when_bind_params_satisfy_only_some_deps", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		// Register testDatabase depending on testShape.
		bDB := registerar.NewBinding("", false, nil, nil, func(s testShape) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "", bDB))

		// Register testShape needing (int, testDatabase). int is satisfied by bindParams,
		// but testDatabase still needs container lookup → cycle through testDatabase → testShape.
		params := []reflect.Value{reflect.ValueOf(42)}
		bShape := registerar.NewBinding("", false, params, nil, func(x int, d testDatabase) testShape { return &testCircle{} }, nil)
		err := r.Set(shapeType, "", bShape)
		assert.ErrorIs(t, err, errors.ErrCircularDependency)
	})

	t.Run("no_cycle_when_all_deps_satisfied_by_bind_params", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		// Register testDatabase depending on testShape.
		bDB := registerar.NewBinding("", false, nil, nil, func(s testShape) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "", bDB))

		// Register testShape needing (int, testDatabase). Both supplied via bindParams → no cycle.
		params := []reflect.Value{reflect.ValueOf(42), reflect.ValueOf(&testMySQL{})}
		bShape := registerar.NewBinding("", false, params, nil, func(x int, d testDatabase) testShape { return &testCircle{} }, nil)
		assert.NoError(t, r.Set(shapeType, "", bShape))
	})

	t.Run("bind_param_not_reused_for_duplicate_dep_types", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		shapeType := reflect.TypeFor[testShape]()

		// Resolver needs two testShape args, but only one bindParam is provided.
		// The second testShape must come from the container — self-referencing cycle.
		params := []reflect.Value{reflect.ValueOf(&testCircle{R: 1})}
		b := registerar.NewBinding("", false, params, nil, func(a testShape, b testShape) testShape { return a }, nil)

		err := r.Set(shapeType, "", b)
		assert.ErrorIs(t, err, errors.ErrCircularDependency)
	})

	t.Run("detects_indirect_circular_dependency", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		// Register B (testDatabase) depending on A (testShape).
		bDB := registerar.NewBinding("", false, nil, nil, func(s testShape) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "", bDB))

		// Try to register A (testShape) depending on B (testDatabase) — indirect cycle.
		bShape := registerar.NewBinding("", false, nil, nil, func(d testDatabase) testShape { return &testCircle{} }, nil)
		err := r.Set(shapeType, "", bShape)
		assert.ErrorIs(t, err, errors.ErrCircularDependency)
	})

	t.Run("no_cycle_with_registered_dependency", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		// Register B (testDatabase) with no dependencies.
		bDB := registerar.NewBinding("", false, nil, nil, func() testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "", bDB))

		// Register A (testShape) depending on B — no cycle.
		bShape := registerar.NewBinding("", false, nil, nil, func(d testDatabase) testShape { return &testCircle{} }, nil)
		assert.NoError(t, r.Set(shapeType, "", bShape))
	})

	t.Run("no_cycle_with_unregistered_dependency", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		shapeType := reflect.TypeFor[testShape]()

		// Register A depending on B which is NOT registered — no cycle.
		b := registerar.NewBinding("", false, nil, nil, func(d testDatabase) testShape { return &testCircle{} }, nil)
		assert.NoError(t, r.Set(shapeType, "", b))
	})

	t.Run("diamond_dependency_no_cycle", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		logType := reflect.TypeFor[testLogger]()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		// C (testLogger) has no deps.
		bLog := registerar.NewBinding("", false, nil, nil, func() testLogger { return nil }, nil)
		require.NoError(t, r.Set(logType, "", bLog))

		// B (testDatabase) depends on C.
		bDB := registerar.NewBinding("", false, nil, nil, func(l testLogger) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "", bDB))

		// A (testShape) depends on B and C — diamond shape, C visited twice.
		bShape := registerar.NewBinding("", false, nil, nil, func(d testDatabase, l testLogger) testShape { return &testCircle{} }, nil)
		assert.NoError(t, r.Set(shapeType, "", bShape))
	})
}

func TestRegistrar_SetIfAbsent(t *testing.T) {
	t.Parallel()

	t.Run("stores_when_slot_empty", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)

		wasNew, err := r.SetIfAbsent(typ, "", b)
		assert.NoError(t, err)
		assert.True(t, wasNew)

		got, found := r.Get(typ, "")
		assert.True(t, found)
		assert.Equal(t, b, got)
	})

	t.Run("returns_false_when_slot_occupied", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b1 := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{R: 1} }, nil)
		b2 := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{R: 2} }, nil)

		wasNew, err := r.SetIfAbsent(typ, "", b1)
		require.NoError(t, err)
		require.True(t, wasNew)

		wasNew, err = r.SetIfAbsent(typ, "", b2)
		assert.NoError(t, err)
		assert.False(t, wasNew)

		// Original binding is still stored.
		got, _ := r.Get(typ, "")
		assert.Equal(t, b1, got)
	})

	t.Run("different_names_are_independent_slots", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b1 := registerar.NewBinding("a", false, nil, nil, func() testShape { return &testCircle{R: 1} }, nil)
		b2 := registerar.NewBinding("b", false, nil, nil, func() testShape { return &testCircle{R: 2} }, nil)

		wasNew, err := r.SetIfAbsent(typ, "a", b1)
		require.NoError(t, err)
		assert.True(t, wasNew)

		wasNew, err = r.SetIfAbsent(typ, "b", b2)
		assert.NoError(t, err)
		assert.True(t, wasNew, "different name is an independent slot")
	})

	t.Run("detects_self_referencing_resolver", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("", false, nil, nil, func(s testShape) testShape { return s }, nil)

		wasNew, err := r.SetIfAbsent(typ, "", b)
		assert.ErrorIs(t, err, errors.ErrCircularDependency)
		assert.False(t, wasNew)
	})

	t.Run("no_cycle_when_self_ref_satisfied_by_bind_params", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()
		params := []reflect.Value{reflect.ValueOf(&testCircle{R: 3})}
		b := registerar.NewBinding("", false, params, nil, func(s testShape) testShape { return s }, nil)

		wasNew, err := r.SetIfAbsent(typ, "", b)
		assert.NoError(t, err)
		assert.True(t, wasNew)
	})
}

func TestRegistrar_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	t.Run("concurrent_set_and_get", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()

		var wg sync.WaitGroup
		for i := range 100 {
			wg.Add(2)
			go func(n int) {
				defer wg.Done()
				b := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{R: n} }, nil)
				_ = r.Set(typ, "", b)
			}(i)
			go func() {
				defer wg.Done()
				r.Get(typ, "")
			}()
		}
		wg.Wait()

		_, found := r.Get(typ, "")
		assert.True(t, found)
	})

	t.Run("concurrent_find_and_set", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		concreteType := reflect.TypeFor[*testCircle]()
		ifaceType := reflect.TypeFor[testShape]()

		var wg sync.WaitGroup
		for i := range 100 {
			wg.Add(2)
			go func(n int) {
				defer wg.Done()
				b := registerar.NewBinding("", false, nil, nil, func() *testCircle { return &testCircle{R: n} }, nil)
				_ = r.Set(concreteType, "", b)
			}(i)
			go func() {
				defer wg.Done()
				r.Find(ifaceType, "")
			}()
		}
		wg.Wait()
	})

	t.Run("concurrent_reset_and_get", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		typ := reflect.TypeFor[testShape]()

		var wg sync.WaitGroup
		for range 100 {
			wg.Go(func() {
				b := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)
				_ = r.Set(typ, "", b)
			})

			wg.Go(func() {
				r.Reset()
			})
		}
		wg.Wait()
	})
}

// TestRegistrar_CircularDependencies covers what the graph adds on top of a plain yes or
// no: the path of the cycle, and the rollback of the refused registration.
func TestRegistrar_CircularDependencies(t *testing.T) {
	t.Parallel()

	t.Run("reports_the_path_of_the_cycle", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		bDB := registerar.NewBinding("", false, nil, nil, func(s testShape) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "", bDB))

		bShape := registerar.NewBinding("", false, nil, nil, func(d testDatabase) testShape { return &testCircle{} }, nil)
		err := r.Set(shapeType, "", bShape)

		require.ErrorIs(t, err, errors.ErrCircularDependency)
		assert.Contains(t, err.Error(), "registerar_test.testShape -> registerar_test.testDatabase -> registerar_test.testShape")
	})

	t.Run("names_the_binding_in_the_path_of_the_cycle", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		shapeType := reflect.TypeFor[testShape]()
		b := registerar.NewBinding("circle", false, nil, nil, func(s testShape) testShape { return s }, nil)

		err := r.Set(shapeType, "circle", b)

		require.ErrorIs(t, err, errors.ErrCircularDependency)
		assert.Contains(t, err.Error(), `registerar_test.testShape("circle") -> registerar_test.testShape("circle")`)
	})

	t.Run("detects_a_cycle_between_bindings_of_the_same_name", func(t *testing.T) {
		t.Parallel()

		// A named binding resolves under its own name.
		r := registerar.NewRegisterar()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		bDB := registerar.NewBinding("primary", false, nil, nil, func(s testShape) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "primary", bDB))

		bShape := registerar.NewBinding("primary", false, nil, nil, func(d testDatabase) testShape { return &testCircle{} }, nil)
		err := r.Set(shapeType, "primary", bShape)

		assert.ErrorIs(t, err, errors.ErrCircularDependency)
	})

	t.Run("ignores_a_binding_registered_under_another_name", func(t *testing.T) {
		t.Parallel()

		// Nothing is registered under "primary": no edge, and so no cycle.
		r := registerar.NewRegisterar()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		bDB := registerar.NewBinding("", false, nil, nil, func(s testShape) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "", bDB))

		bShape := registerar.NewBinding("primary", false, nil, nil, func(d testDatabase) testShape { return &testCircle{} }, nil)
		assert.NoError(t, r.Set(shapeType, "primary", bShape))
	})

	t.Run("follows_the_named_bindings_given_at_bind_time", func(t *testing.T) {
		t.Parallel()

		// Neither reaches the other under its own name: each is told to resolve from the
		// other's, which closes the cycle.
		r := registerar.NewRegisterar()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		bDB := registerar.NewBinding("primary", false, nil, []string{""}, func(s testShape) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "primary", bDB))

		bShape := registerar.NewBinding("", false, nil, []string{"primary"}, func(d testDatabase) testShape { return &testCircle{} }, nil)
		err := r.Set(shapeType, "", bShape)

		require.ErrorIs(t, err, errors.ErrCircularDependency)
		assert.Contains(t, err.Error(), `registerar_test.testShape -> registerar_test.testDatabase("primary") -> registerar_test.testShape`)
	})

	t.Run("falls_back_to_the_name_of_the_binding", func(t *testing.T) {
		t.Parallel()

		// The named binding given at bind time matches nothing, so the dependency is
		// looked up the usual way — and that is what closes the cycle.
		r := registerar.NewRegisterar()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		bDB := registerar.NewBinding("", false, nil, nil, func(s testShape) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "", bDB))

		bShape := registerar.NewBinding("", false, nil, []string{"nonexistent"}, func(d testDatabase) testShape { return &testCircle{} }, nil)
		err := r.Set(shapeType, "", bShape)

		assert.ErrorIs(t, err, errors.ErrCircularDependency)
	})

	t.Run("detects_a_cycle_through_an_interface_the_binding_implements", func(t *testing.T) {
		t.Parallel()

		// Nothing else implements the interface, so the binding would be handed itself.
		r := registerar.NewRegisterar()
		circleType := reflect.TypeFor[*testCircle]()
		b := registerar.NewBinding("", false, nil, nil, func(s testShape) *testCircle { return &testCircle{} }, nil)

		err := r.Set(circleType, "", b)

		require.ErrorIs(t, err, errors.ErrCircularDependency)
		assert.Contains(t, err.Error(), "*registerar_test.testCircle -> *registerar_test.testCircle")
	})

	t.Run("no_cycle_when_another_implementation_answers_first", func(t *testing.T) {
		t.Parallel()

		// A second implementation is registered first, and answers for the interface.
		circleType := reflect.TypeFor[*testCircle]()
		squareType := reflect.TypeFor[*testSquare]()

		for range 50 {
			r := registerar.NewRegisterar()
			square := registerar.NewBinding("", false, nil, nil, func() *testSquare { return &testSquare{S: 2} }, nil)
			require.NoError(t, r.Set(squareType, "", square))

			b := registerar.NewBinding("", false, nil, nil, func(s testShape) *testCircle { return &testCircle{} }, nil)
			require.NoError(t, r.Set(circleType, "", b), "the registration must not depend on the order of a map walk")
		}
	})

	t.Run("rolls_the_refused_registration_back", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		bDB := registerar.NewBinding("", false, nil, nil, func(s testShape) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "", bDB))

		bShape := registerar.NewBinding("", false, nil, nil, func(d testDatabase) testShape { return &testCircle{} }, nil)
		require.ErrorIs(t, r.Set(shapeType, "", bShape), errors.ErrCircularDependency)

		_, exist := r.Get(shapeType, "")
		assert.False(t, exist, "the refused binding must not be stored")

		// The slot is left free, so a binding that closes no cycle still fits in it.
		bFine := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		assert.NoError(t, r.Set(shapeType, "", bFine))

		got, exist := r.Get(shapeType, "")
		assert.True(t, exist)
		assert.Equal(t, bFine, got)
	})

	t.Run("leaves_the_other_bindings_of_the_type_alone", func(t *testing.T) {
		t.Parallel()

		// Rolling the second name back must not touch the one next to it.
		r := registerar.NewRegisterar()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		kept := registerar.NewBinding("kept", false, nil, nil, func() testShape { return &testCircle{R: 1} }, nil)
		require.NoError(t, r.Set(shapeType, "kept", kept))

		bDB := registerar.NewBinding("refused", false, nil, nil, func(s testShape) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "refused", bDB))

		cyclic := registerar.NewBinding("refused", false, nil, nil, func(d testDatabase) testShape { return &testCircle{R: 2} }, nil)
		require.ErrorIs(t, r.Set(shapeType, "refused", cyclic), errors.ErrCircularDependency)

		_, exist := r.Get(shapeType, "refused")
		assert.False(t, exist, "the refused binding must not be stored")

		got, exist := r.Get(shapeType, "kept")
		assert.True(t, exist, "the other name of the same type must survive")
		assert.Equal(t, kept, got)
	})

	t.Run("keeps_the_binding_the_refused_one_would_have_replaced", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		bShape := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{R: 1} }, nil)
		require.NoError(t, r.Set(shapeType, "", bShape))

		bDB := registerar.NewBinding("", false, nil, nil, func(s testShape) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "", bDB))

		// Rebinding the Shape to a resolver that needs the Database closes the cycle.
		cyclic := registerar.NewBinding("", false, nil, nil, func(d testDatabase) testShape { return &testCircle{R: 2} }, nil)
		require.ErrorIs(t, r.Set(shapeType, "", cyclic), errors.ErrCircularDependency)

		got, exist := r.Get(shapeType, "")
		assert.True(t, exist)
		assert.Equal(t, bShape, got, "the binding that was already there must survive")
	})

	t.Run("keeps_detecting_cycles_after_a_binding_is_deleted", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		logType := reflect.TypeFor[testLogger]()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		// Deleting the Logger leaves the graph with a slot it knows nothing about.
		bLog := registerar.NewBinding("", false, nil, nil, func() testLogger { return nil }, nil)
		require.NoError(t, r.Set(logType, "", bLog))

		bDB := registerar.NewBinding("", false, nil, nil, func(s testShape) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "", bDB))

		r.Delete(logType, "")

		bShape := registerar.NewBinding("", false, nil, nil, func(d testDatabase) testShape { return &testCircle{} }, nil)
		assert.ErrorIs(t, r.Set(shapeType, "", bShape), errors.ErrCircularDependency)
	})

	t.Run("detects_a_cycle_closed_by_a_dependency_registered_later", func(t *testing.T) {
		t.Parallel()

		// Neither of the first two closes a cycle: the Shape is still unknown.
		r := registerar.NewRegisterar()
		logType := reflect.TypeFor[testLogger]()
		dbType := reflect.TypeFor[testDatabase]()
		shapeType := reflect.TypeFor[testShape]()

		bDB := registerar.NewBinding("", false, nil, nil, func(s testShape) testDatabase { return &testMySQL{} }, nil)
		require.NoError(t, r.Set(dbType, "", bDB))

		bLog := registerar.NewBinding("", false, nil, nil, func(d testDatabase) testLogger { return nil }, nil)
		require.NoError(t, r.Set(logType, "", bLog))

		bShape := registerar.NewBinding("", false, nil, nil, func(l testLogger) testShape { return &testCircle{} }, nil)
		err := r.Set(shapeType, "", bShape)

		require.ErrorIs(t, err, errors.ErrCircularDependency)
		assert.Contains(t, err.Error(), "registerar_test.testShape -> registerar_test.testLogger -> registerar_test.testDatabase -> registerar_test.testShape")
	})
}

// TestRegistrar_FindPicksTheSameImplementationEveryTime covers the fallback when several
// bound types implement the interface, which a map walk alone would leave to chance.
func TestRegistrar_FindPicksTheSameImplementationEveryTime(t *testing.T) {
	t.Parallel()

	shapeType := reflect.TypeFor[testShape]()
	circleType := reflect.TypeFor[*testCircle]()
	squareType := reflect.TypeFor[*testSquare]()

	r := registerar.NewRegisterar()
	circle := registerar.NewBinding("", false, nil, nil, func() *testCircle { return &testCircle{R: 1} }, nil)
	square := registerar.NewBinding("", false, nil, nil, func() *testSquare { return &testSquare{S: 2} }, nil)

	require.NoError(t, r.Set(circleType, "", circle))
	require.NoError(t, r.Set(squareType, "", square))

	for range 100 {
		got, exist := r.Find(shapeType, "")
		require.True(t, exist)
		assert.Equal(t, circle, got, "the implementation registered first is the one that answers")

		slot, exist := r.FindSlot(shapeType, "")
		require.True(t, exist)
		assert.Equal(t, registerar.Slot{Type: circleType}, slot)
	}
}

func TestRegistrar_FindSlot(t *testing.T) {
	t.Parallel()

	shapeType := reflect.TypeFor[testShape]()
	circleType := reflect.TypeFor[*testCircle]()

	t.Run("returns_the_slot_of_an_exact_match", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		b := registerar.NewBinding("circle", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		require.NoError(t, r.Set(shapeType, "circle", b))

		slot, exist := r.FindSlot(shapeType, "circle")

		assert.True(t, exist)
		assert.Equal(t, registerar.Slot{Type: shapeType, Name: "circle"}, slot)
	})

	t.Run("returns_the_slot_of_the_implementation_of_an_interface", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		b := registerar.NewBinding("", false, nil, nil, func() *testCircle { return &testCircle{} }, nil)
		require.NoError(t, r.Set(circleType, "", b))

		slot, exist := r.FindSlot(shapeType, "")

		assert.True(t, exist)
		assert.Equal(t, registerar.Slot{Type: circleType}, slot, "the slot is the one of the concrete type")
	})

	t.Run("returns_nothing_when_no_binding_matches", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()

		slot, exist := r.FindSlot(shapeType, "")

		assert.False(t, exist)
		assert.Equal(t, registerar.Slot{}, slot)
	})
}

func TestRegistrar_Registrations(t *testing.T) {
	t.Parallel()

	t.Run("returns_the_registrations_in_the_order_they_were_made", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		shapeType := reflect.TypeFor[testShape]()
		dbType := reflect.TypeFor[testDatabase]()

		bShape := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		bNamed := registerar.NewBinding("circle", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		bDB := registerar.NewBinding("", false, nil, nil, func() testDatabase { return &testMySQL{} }, nil)

		require.NoError(t, r.Set(shapeType, "", bShape))
		require.NoError(t, r.Set(shapeType, "circle", bNamed))
		require.NoError(t, r.Set(dbType, "", bDB))

		assert.Equal(t, []registerar.Registration{
			{Slot: registerar.Slot{Type: shapeType, Name: ""}, Binding: bShape},
			{Slot: registerar.Slot{Type: shapeType, Name: "circle"}, Binding: bNamed},
			{Slot: registerar.Slot{Type: dbType, Name: ""}, Binding: bDB},
		}, r.Registrations())
	})

	t.Run("leaves_out_the_deleted_bindings", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		shapeType := reflect.TypeFor[testShape]()
		dbType := reflect.TypeFor[testDatabase]()

		bShape := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)
		bDB := registerar.NewBinding("", false, nil, nil, func() testDatabase { return &testMySQL{} }, nil)

		require.NoError(t, r.Set(shapeType, "", bShape))
		require.NoError(t, r.Set(dbType, "", bDB))
		r.Delete(shapeType, "")

		assert.Equal(t, []registerar.Registration{
			{Slot: registerar.Slot{Type: dbType, Name: ""}, Binding: bDB},
		}, r.Registrations())
	})

	t.Run("is_empty_after_a_reset", func(t *testing.T) {
		t.Parallel()

		r := registerar.NewRegisterar()
		shapeType := reflect.TypeFor[testShape]()
		require.NoError(t, r.Set(shapeType, "", registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)))

		r.Reset()

		assert.Empty(t, r.Registrations())
	})
}

func TestSlot_String(t *testing.T) {
	t.Parallel()

	shapeType := reflect.TypeFor[testShape]()

	assert.Equal(t, "registerar_test.testShape", registerar.Slot{Type: shapeType}.String())
	assert.Equal(t, `registerar_test.testShape("circle")`, registerar.Slot{Type: shapeType, Name: "circle"}.String())
	assert.Equal(t, "<unknown>", registerar.Slot{}.String())
}

func TestBinding_Dependencies(t *testing.T) {
	t.Parallel()

	shapeType := reflect.TypeFor[testShape]()
	dbType := reflect.TypeFor[testDatabase]()

	t.Run("returns_the_inputs_of_the_resolver", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", false, nil, nil, func(s testShape, d testDatabase) testShape { return s }, nil)

		assert.Equal(t, []reflect.Type{shapeType, dbType}, b.Dependencies())
	})

	t.Run("leaves_out_the_inputs_the_bind_time_params_satisfy", func(t *testing.T) {
		t.Parallel()

		params := []reflect.Value{reflect.ValueOf(&testCircle{R: 1})}
		b := registerar.NewBinding("", false, params, nil, func(s testShape, d testDatabase) testShape { return s }, nil)

		assert.Equal(t, []reflect.Type{dbType}, b.Dependencies())
	})

	t.Run("uses_a_bind_time_param_once", func(t *testing.T) {
		t.Parallel()

		params := []reflect.Value{reflect.ValueOf(&testCircle{R: 1})}
		b := registerar.NewBinding("", false, params, nil, func(a testShape, b testShape) testShape { return a }, nil)

		assert.Equal(t, []reflect.Type{shapeType}, b.Dependencies())
	})

	t.Run("returns_nothing_for_a_resolver_without_inputs", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)

		assert.Empty(t, b.Dependencies())
	})
}

func TestBinding_DependencyNames(t *testing.T) {
	t.Parallel()

	t.Run("yields_the_named_bindings_before_the_name_of_the_binding", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("own", false, nil, []string{"first", "second"}, func() testShape { return &testCircle{} }, nil)

		assert.Equal(t, []string{"first", "second", "own"}, slices.Collect(b.DependencyNames()))
	})

	t.Run("yields_the_name_of_the_binding_alone", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", false, nil, nil, func() testShape { return &testCircle{} }, nil)

		assert.Equal(t, []string{""}, slices.Collect(b.DependencyNames()))
	})

	t.Run("stops_when_the_caller_stops", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("own", false, nil, []string{"first", "second"}, func() testShape { return &testCircle{} }, nil)

		var names []string
		for name := range b.DependencyNames() {
			names = append(names, name)
			break
		}

		assert.Equal(t, []string{"first"}, names)
	})
}
