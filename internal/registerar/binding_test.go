package registerar_test

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/danceable/container/internal/registerar"
	"github.com/stretchr/testify/assert"
)

func TestNewBinding(t *testing.T) {
	t.Parallel()

	t.Run("creates_binding_with_all_fields", func(t *testing.T) {
		t.Parallel()

		resolver := func() string { return "hello" }
		params := []reflect.Value{reflect.ValueOf(42)}
		named := []string{"dep1", "dep2"}

		b := registerar.NewBinding("myBinding", true, params, named, resolver, "cached")

		assert.Equal(t, "myBinding", b.GetName())
		assert.True(t, b.IsSingleton())
		assert.Equal(t, params, b.BindParams())
		assert.Equal(t, named, b.NamedBindings())
		assert.NotNil(t, b.Resolver())
		assert.Equal(t, "cached", b.Concrete())
	})

	t.Run("creates_binding_with_zero_values", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", false, nil, nil, func() int { return 0 }, nil)

		assert.Equal(t, "", b.GetName())
		assert.False(t, b.IsSingleton())
		assert.Nil(t, b.BindParams())
		assert.Nil(t, b.NamedBindings())
		assert.Nil(t, b.Concrete())
	})
}

func TestBinding_HasName(t *testing.T) {
	t.Parallel()

	t.Run("returns_true_when_name_is_set", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("someName", false, nil, nil, func() {}, nil)
		assert.True(t, b.HasName())
	})

	t.Run("returns_false_when_name_is_empty", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", false, nil, nil, func() {}, nil)
		assert.False(t, b.HasName())
	})
}

func TestBinding_GetName(t *testing.T) {
	t.Parallel()

	b := registerar.NewBinding("fooBar", false, nil, nil, func() {}, nil)
	assert.Equal(t, "fooBar", b.GetName())
}

func TestBinding_IsSingleton(t *testing.T) {
	t.Parallel()

	t.Run("returns_true_for_singleton", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, nil)
		assert.True(t, b.IsSingleton())
	})

	t.Run("returns_false_for_transient", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", false, nil, nil, func() {}, nil)
		assert.False(t, b.IsSingleton())
	})
}

func TestBinding_BindParams(t *testing.T) {
	t.Parallel()

	t.Run("returns_params", func(t *testing.T) {
		t.Parallel()

		params := []reflect.Value{reflect.ValueOf("a"), reflect.ValueOf(1)}
		b := registerar.NewBinding("", false, params, nil, func() {}, nil)

		assert.Equal(t, params, b.BindParams())
	})

	t.Run("returns_nil_when_no_params", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", false, nil, nil, func() {}, nil)
		assert.Nil(t, b.BindParams())
	})
}

func TestBinding_NamedBindings(t *testing.T) {
	t.Parallel()

	t.Run("returns_named_bindings", func(t *testing.T) {
		t.Parallel()

		names := []string{"x", "y"}
		b := registerar.NewBinding("", false, nil, names, func() {}, nil)
		assert.Equal(t, names, b.NamedBindings())
	})

	t.Run("returns_nil_when_no_named_bindings", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", false, nil, nil, func() {}, nil)
		assert.Nil(t, b.NamedBindings())
	})
}

func TestBinding_Resolver(t *testing.T) {
	t.Parallel()

	resolver := func() string { return "hello" }
	b := registerar.NewBinding("", false, nil, nil, resolver, nil)

	assert.NotNil(t, b.Resolver())
}

func TestBinding_HasConcrete(t *testing.T) {
	t.Parallel()

	t.Run("returns_false_when_concrete_is_nil", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, nil)
		assert.False(t, b.HasConcrete())
	})

	t.Run("returns_true_when_concrete_is_set", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, "instance")
		assert.True(t, b.HasConcrete())
	})
}

func TestBinding_Concrete(t *testing.T) {
	t.Parallel()

	t.Run("returns_nil_when_not_set", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, nil)
		assert.Nil(t, b.Concrete())
	})

	t.Run("returns_concrete_when_set_at_creation", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, "instance")
		assert.Equal(t, "instance", b.Concrete())
	})
}

func TestBinding_SetConcrete(t *testing.T) {
	t.Parallel()

	t.Run("sets_concrete_on_nil_binding", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, nil)
		assert.Nil(t, b.Concrete())

		b.SetConcrete("value")
		assert.Equal(t, "value", b.Concrete())
	})

	t.Run("overwrites_existing_concrete", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, "old")
		assert.Equal(t, "old", b.Concrete())

		b.SetConcrete("new")
		assert.Equal(t, "new", b.Concrete())
	})

	t.Run("sets_concrete_to_nil", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, "existing")
		assert.True(t, b.HasConcrete())

		b.SetConcrete(nil)
		assert.False(t, b.HasConcrete())
		assert.Nil(t, b.Concrete())
	})

	t.Run("is_safe_for_concurrent_use", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, nil)

		var wg sync.WaitGroup
		const goroutines = 50

		for i := range goroutines {
			wg.Add(1)
			go func(val int) {
				defer wg.Done()
				b.SetConcrete(val)
			}(i)
		}

		wg.Wait()

		assert.NotNil(t, b.Concrete())
	})
}

func TestBinding_GetOrSetConcrete(t *testing.T) {
	t.Parallel()

	t.Run("returns_existing_concrete_without_calling_factory", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, "existing")

		called := false
		factory := func(b *registerar.Binding, params []reflect.Value) (any, error) {
			called = true
			return "new", nil
		}

		result, err := b.GetOrSetConcrete(factory, nil)

		assert.NoError(t, err)
		assert.Equal(t, "existing", result)
		assert.False(t, called)
	})

	t.Run("calls_factory_when_concrete_is_nil", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, nil)

		factory := func(b *registerar.Binding, params []reflect.Value) (any, error) {
			return "created", nil
		}

		result, err := b.GetOrSetConcrete(factory, nil)

		assert.NoError(t, err)
		assert.Equal(t, "created", result)
		assert.Equal(t, "created", b.Concrete())
	})

	t.Run("passes_params_to_factory", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, nil)
		params := []reflect.Value{reflect.ValueOf(42), reflect.ValueOf("hello")}

		var receivedParams []reflect.Value
		factory := func(b *registerar.Binding, params []reflect.Value) (any, error) {
			receivedParams = params
			return "ok", nil
		}

		_, err := b.GetOrSetConcrete(factory, params)

		assert.NoError(t, err)
		assert.Equal(t, params, receivedParams)
	})

	t.Run("returns_error_from_factory", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, nil)

		factory := func(b *registerar.Binding, params []reflect.Value) (any, error) {
			return nil, errors.New("factory failed")
		}

		result, err := b.GetOrSetConcrete(factory, nil)

		assert.EqualError(t, err, "factory failed")
		assert.Nil(t, result)
		assert.False(t, b.HasConcrete())
	})

	t.Run("calls_factory_only_once_on_concurrent_access", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, nil)

		var callCount int
		var mu sync.Mutex

		factory := func(b *registerar.Binding, params []reflect.Value) (any, error) {
			mu.Lock()
			callCount++
			mu.Unlock()
			return "singleton", nil
		}

		var wg sync.WaitGroup
		const goroutines = 50

		results := make([]any, goroutines)
		errs := make([]error, goroutines)

		for i := range goroutines {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				results[idx], errs[idx] = b.GetOrSetConcrete(factory, nil)
			}(i)
		}

		wg.Wait()

		assert.Equal(t, 1, callCount)
		for i := range goroutines {
			assert.NoError(t, errs[i])
			assert.Equal(t, "singleton", results[i])
		}
	})

	t.Run("does_not_store_concrete_on_error", func(t *testing.T) {
		t.Parallel()

		b := registerar.NewBinding("", true, nil, nil, func() {}, nil)

		failFactory := func(b *registerar.Binding, params []reflect.Value) (any, error) {
			return nil, errors.New("fail")
		}

		_, err := b.GetOrSetConcrete(failFactory, nil)
		assert.Error(t, err)
		assert.False(t, b.HasConcrete())

		// A subsequent call with a working factory should succeed.
		successFactory := func(b *registerar.Binding, params []reflect.Value) (any, error) {
			return "recovered", nil
		}

		result, err := b.GetOrSetConcrete(successFactory, nil)
		assert.NoError(t, err)
		assert.Equal(t, "recovered", result)
		assert.True(t, b.HasConcrete())
	})
}
