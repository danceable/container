package resolve_test

import (
	"reflect"
	"testing"

	"github.com/danceable/container/resolve"
	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	t.Parallel()

	t.Run("default-options", func(t *testing.T) {
		t.Parallel()

		opts := resolve.DefaultOptions()

		assert.NotNil(t, opts)
		assert.Equal(t, "", opts.Name)
		assert.Empty(t, opts.Params)
	})

	t.Run("multiple-options", func(t *testing.T) {
		t.Parallel()

		opts := resolve.DefaultOptions()
		resolve.WithName("cache")(opts)
		resolve.WithParams("arg1", 99)(opts)

		assert.Equal(t, "cache", opts.Name)
		assert.Len(t, opts.Params, 2)
		assert.Equal(t, "arg1", opts.Params[0].Interface())
		assert.Equal(t, 99, opts.Params[1].Interface())
	})
}

func TestWithName(t *testing.T) {
	t.Parallel()

	t.Run("non-empty-name", func(t *testing.T) {
		t.Parallel()

		opts := resolve.DefaultOptions()
		resolve.WithName("myService")(opts)

		assert.Equal(t, "myService", opts.Name)
		assert.Empty(t, opts.Params)
	})

	t.Run("empty-name", func(t *testing.T) {
		t.Parallel()

		opts := resolve.DefaultOptions()
		resolve.WithName("")(opts)

		assert.Equal(t, "", opts.Name)
		assert.Empty(t, opts.Params)
	})
}

func TestWithParams(t *testing.T) {
	t.Parallel()

	t.Run("no-params", func(t *testing.T) {
		t.Parallel()

		opts := resolve.DefaultOptions()
		resolve.WithParams()(opts)

		assert.Empty(t, opts.Params)
	})

	t.Run("single-param", func(t *testing.T) {
		t.Parallel()

		opts := resolve.DefaultOptions()
		resolve.WithParams(42)(opts)

		assert.Len(t, opts.Params, 1)
		assert.Equal(t, reflect.ValueOf(42).Interface(), opts.Params[0].Interface())
		assert.Equal(t, "", opts.Name)
	})

	t.Run("multiple-params", func(t *testing.T) {
		t.Parallel()

		opts := resolve.DefaultOptions()
		resolve.WithParams("hello", 3.14, true)(opts)

		assert.Len(t, opts.Params, 3)
		assert.Equal(t, "hello", opts.Params[0].Interface())
		assert.Equal(t, 3.14, opts.Params[1].Interface())
		assert.Equal(t, true, opts.Params[2].Interface())
		assert.Equal(t, "", opts.Name)
	})

	t.Run("accumulate-params", func(t *testing.T) {
		t.Parallel()

		opts := resolve.DefaultOptions()
		resolve.WithParams(1, 2)(opts)
		resolve.WithParams(3)(opts)

		assert.Len(t, opts.Params, 3)
		assert.Equal(t, 1, opts.Params[0].Interface())
		assert.Equal(t, 2, opts.Params[1].Interface())
		assert.Equal(t, 3, opts.Params[2].Interface())
		assert.Equal(t, "", opts.Name)
	})
}
