package registerar_test

import (
	"reflect"
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
