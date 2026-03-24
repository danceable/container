package bind_test

import (
	"testing"

	"github.com/danceable/container/bind"
	"github.com/stretchr/testify/assert"
)

func TestDefaultBindOptions(t *testing.T) {
	t.Parallel()

	t.Run("default-options", func(t *testing.T) {
		t.Parallel()

		opts := bind.DefaultOptions()

		assert.NotNil(t, opts)
		assert.Equal(t, "", opts.Name)
		assert.False(t, opts.Singleton)
		assert.False(t, opts.Lazy)
	})

	t.Run("multiple-options", func(t *testing.T) {
		t.Parallel()

		opts := bind.DefaultOptions()
		bind.WithName("cache")(opts)
		bind.Singleton()(opts)
		bind.Lazy()(opts)

		assert.Equal(t, "cache", opts.Name)
		assert.True(t, opts.Singleton)
		assert.True(t, opts.Lazy)
	})
}

func TestWithName(t *testing.T) {
	t.Parallel()

	t.Run("non-empty-name", func(t *testing.T) {
		opts := bind.DefaultOptions()
		bind.WithName("myBinding")(opts)

		assert.Equal(t, "myBinding", opts.Name)
		assert.False(t, opts.Singleton)
		assert.False(t, opts.Lazy)
	})

	t.Run("empty-name", func(t *testing.T) {
		t.Parallel()

		opts := bind.DefaultOptions()
		bind.WithName("")(opts)

		assert.Equal(t, "", opts.Name)
		assert.False(t, opts.Singleton)
		assert.False(t, opts.Lazy)
	})
}

func TestSingleton(t *testing.T) {
	t.Parallel()

	opts := bind.DefaultOptions()
	bind.Singleton()(opts)

	assert.True(t, opts.Singleton)
	assert.Equal(t, "", opts.Name)
	assert.False(t, opts.Lazy)
}

func TestLazy(t *testing.T) {
	t.Parallel()

	opts := bind.DefaultOptions()
	bind.Lazy()(opts)

	assert.True(t, opts.Lazy)
	assert.Equal(t, "", opts.Name)
	assert.False(t, opts.Singleton)
}

func TestResolveDepenenciesByParams(t *testing.T) {
	t.Parallel()

	opts := bind.DefaultOptions()
	bind.ResolveDepenenciesByParams(1, "two", struct{}{})(opts)

	assert.Len(t, opts.DependenciesByParams, 3)
	assert.Equal(t, 1, opts.DependenciesByParams[0].Interface())
	assert.Equal(t, "two", opts.DependenciesByParams[1].Interface())
	assert.Equal(t, struct{}{}, opts.DependenciesByParams[2].Interface())
}

func TestResolveDependenciesByNamedBindings(t *testing.T) {
	t.Parallel()

	opts := bind.DefaultOptions()
	bind.ResolveDependenciesByNamedBindings("db", "cache")(opts)

	assert.Len(t, opts.DependenciesByNamedBindings, 2)
	assert.Equal(t, []string{"db", "cache"}, opts.DependenciesByNamedBindings)
}
