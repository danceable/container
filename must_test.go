package container_test

import (
	"testing"

	"github.com/danceable/container"
	"github.com/danceable/container/bind"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMustBind(t *testing.T) {
	t.Parallel()

	t.Run("panics_when_resolver_not_function", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		assert.Panics(t, func() {
			container.MustBind(c, 123)
		})
	})

	t.Run("panics_when_resolver_signature_invalid", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		assert.Panics(t, func() {
			container.MustBind(c, func() {})
		})
	})

	t.Run("panics_when_resolver_depends_on_abstract", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		assert.Panics(t, func() {
			container.MustBind(c, func(s Shape) Shape { return s })
		})
	})

	t.Run("no_panic_on_successful_bind", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		container.MustBind(c, func() Shape { return &Circle{a: 5} })

		var s Shape
		assert.NotPanics(t, func() {
			container.MustResolve(c, &s)
		})
		assert.Equal(t, 5, s.GetArea())
	})
}

func TestMustCall(t *testing.T) {
	t.Parallel()

	t.Run("panics_when_dependency_not_bound", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		assert.Panics(t, func() {
			container.MustCall(c, func(s Shape) { s.GetArea() })
		})
	})

	t.Run("no_panic_on_successful_resolution", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton()))

		assert.NotPanics(t, func() {
			container.MustCall(c, func(s Shape) {})
		})
	})
}

func TestMustResolve(t *testing.T) {
	t.Parallel()

	t.Run("panics_when_abstraction_not_bound", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		var s Shape
		assert.Panics(t, func() {
			container.MustResolve(c, &s)
		})
	})

	t.Run("no_panic_on_successful_resolution", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton()))

		var s Shape
		assert.NotPanics(t, func() {
			container.MustResolve(c, &s)
		})
	})
}

func TestMustFill(t *testing.T) {
	t.Parallel()

	t.Run("panics_when_binding_missing", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		myApp := struct {
			S Shape `container:"type"`
		}{}
		assert.Panics(t, func() {
			container.MustFill(c, &myApp)
		})
	})

	t.Run("no_panic_on_successful_fill", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton()))

		myApp := struct {
			S Shape `container:"type"`
		}{}
		assert.NotPanics(t, func() {
			container.MustFill(c, &myApp)
		})
	})
}
