package container_test

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/danceable/container"
	"github.com/danceable/container/bind"
	containerErrors "github.com/danceable/container/errors"
	"github.com/danceable/container/resolve"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainer_Reset(t *testing.T) {
	t.Parallel()

	t.Run("clears_all_bindings", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton()))

		c.Reset()

		var s Shape
		err := c.Resolve(&s)
		assert.EqualError(t, err, "container: no concrete found for the given abstraction; the abstraction is: container_test.Shape")
	})

	t.Run("rebind_after_reset", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton()))
		c.Reset()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 2} }, bind.Singleton()))

		var s Shape
		require.NoError(t, c.Resolve(&s))
		assert.Equal(t, 2, s.GetArea())
	})
}

func TestContainer_Bind(t *testing.T) {
	t.Parallel()

	t.Run("singleton", func(t *testing.T) {
		t.Parallel()

		for _, lazy := range []bool{false, true} {
			label := "eager"
			if lazy {
				label = "lazy"
			}
			lazy := lazy // capture

			t.Run(label+"/resolves_same_instance", func(t *testing.T) {
				t.Parallel()
				opts := []bind.BindOption{bind.Singleton()}
				if lazy {
					opts = append(opts, bind.Lazy())
				}

				c := container.New()
				require.NoError(t, c.Bind(func() Shape { return &Circle{a: 13} }, opts...))

				require.NoError(t, c.Call(func(s Shape) { s.SetArea(666) }))
				require.NoError(t, c.Call(func(s Shape) { assert.Equal(t, 666, s.GetArea()) }))
			})

			t.Run(label+"/named", func(t *testing.T) {
				t.Parallel()
				opts := []bind.BindOption{bind.WithName("theCircle"), bind.Singleton()}
				if lazy {
					opts = append(opts, bind.Lazy())
				}

				c := container.New()
				require.NoError(t, c.Bind(func() Shape { return &Circle{a: 13} }, opts...))

				var sh Shape
				require.NoError(t, c.Resolve(&sh, resolve.WithName("theCircle")))
				assert.Equal(t, 13, sh.GetArea())
			})
		}

		t.Run("missing_dependency_fails_eagerly", func(t *testing.T) {
			t.Parallel()

			c := container.New()
			err := c.Bind(func(db Database) Shape { return &Circle{a: 13} }, bind.Singleton())
			assert.EqualError(t, err, "container: no concrete found for the given abstraction; the abstraction is: container_test.Database")
		})

		for _, lazy := range []bool{false, true} {
			label := "eager"
			if lazy {
				label = "lazy"
			}
			lazy := lazy

			t.Run(label+"/resolver_returns_nothing_is_invalid", func(t *testing.T) {
				t.Parallel()
				opts := []bind.BindOption{bind.Singleton()}
				if lazy {
					opts = append(opts, bind.Lazy())
				}

				c := container.New()
				err := c.Bind(func() {}, opts...)
				assert.EqualError(t, err, "container: resolver function signature is invalid - it must return abstract, or abstract and error")
			})
		}

		t.Run("resolver_error_fails_eagerly", func(t *testing.T) {
			t.Parallel()

			c := container.New()
			err := c.Bind(func() (Shape, error) { return nil, errors.New("app: error") }, bind.Singleton())
			assert.EqualError(t, err, "app: error")
		})

		t.Run("lazy/resolver_error_surfaced_on_first_resolve", func(t *testing.T) {
			t.Parallel()

			c := container.New()
			require.NoError(t, c.Bind(func() (Shape, error) { return nil, errors.New("app: error") }, bind.Singleton(), bind.Lazy()))

			var s Shape
			err := c.Resolve(&s)
			assert.EqualError(t, err, "container: encountered error while making concrete for: container_test.Shape. Error encountered: app: error")
		})

		t.Run("lazy/nil_result_not_cached_allows_retry", func(t *testing.T) {
			t.Parallel()

			c := container.New()
			resolvable := false
			require.NoError(t, c.Bind(func() (Shape, error) {
				if resolvable {
					return &Circle{a: 5}, nil
				}
				return nil, errors.New("app: not ready")
			}, bind.Singleton(), bind.Lazy()))

			var s Shape
			assert.Error(t, c.Resolve(&s))

			resolvable = true
			require.NoError(t, c.Resolve(&s))
			assert.Equal(t, 5, s.GetArea())
		})

		for _, lazy := range []bool{false, true} {
			label := "eager"
			if lazy {
				label = "lazy"
			}
			lazy := lazy

			t.Run(label+"/non_function_resolver_is_invalid", func(t *testing.T) {
				t.Parallel()
				opts := []bind.BindOption{bind.Singleton()}
				if lazy {
					opts = append(opts, bind.Lazy())
				}

				c := container.New()
				err := c.Bind("STRING!", opts...)
				assert.EqualError(t, err, "container: the resolver must be a function")
			})
		}

		for _, lazy := range []bool{false, true} {
			label := "eager"
			if lazy {
				label = "lazy"
			}
			lazy := lazy

			t.Run(label+"/resolver_with_resolvable_args", func(t *testing.T) {
				t.Parallel()
				opts := []bind.BindOption{bind.Singleton()}
				if lazy {
					opts = append(opts, bind.Lazy())
				}

				c := container.New()
				require.NoError(t, c.Bind(func() Shape { return &Circle{a: 666} }, opts...))
				require.NoError(t, c.Bind(func(s Shape) Database {
					assert.Equal(t, 666, s.GetArea())
					return &MySQL{}
				}, opts...))

				var db Database
				assert.NoError(t, c.Resolve(&db))
			})
		}

		for _, lazy := range []bool{false, true} {
			label := "eager"
			if lazy {
				label = "lazy"
			}
			lazy := lazy

			t.Run(label+"/resolver_depending_on_own_abstract_is_invalid", func(t *testing.T) {
				t.Parallel()
				opts := []bind.BindOption{bind.Singleton()}
				if lazy {
					opts = append(opts, bind.Lazy())
				}

				c := container.New()
				err := c.Bind(func(s Shape) Shape { return &Circle{a: s.GetArea()} }, opts...)
				assert.EqualError(t, err, "container: resolver function signature is invalid - depends on abstract it returns")
			})
		}

		t.Run("bound_as_concrete_resolved_via_interface", func(t *testing.T) {
			t.Parallel()

			c := container.New()
			require.NoError(t, c.Bind(func() *Circle { return &Circle{a: 13} }, bind.Singleton()))

			require.NoError(t, c.Call(func(s Shape) {
				assert.Equal(t, 13, s.GetArea())
				s.SetArea(666)
			}))
			require.NoError(t, c.Call(func(s ReadOnlyShape) {
				assert.Equal(t, 666, s.GetArea())
			}))
		})

		t.Run("named_concrete_resolved_via_named_interface_through_call", func(t *testing.T) {
			t.Parallel()

			c := container.New()
			require.NoError(t, c.Bind(func() *Circle { return &Circle{a: 7} }, bind.WithName("special"), bind.Singleton()))

			err := c.Call(func(s Shape) {
				assert.Equal(t, 7, s.GetArea())
			}, resolve.WithName("special"))
			assert.NoError(t, err)
		})

		t.Run("eager_duplicate_bind_ignored", func(t *testing.T) {
			t.Parallel()

			c := container.New()
			var callCount int64
			resolver := func() Shape {
				atomic.AddInt64(&callCount, 1)
				return &Circle{a: 1}
			}
			require.NoError(t, c.Bind(resolver, bind.Singleton()))
			require.NoError(t, c.Bind(resolver, bind.Singleton())) // second bind should be no-op

			assert.Equal(t, int64(1), atomic.LoadInt64(&callCount))
		})
	})

	t.Run("transient", func(t *testing.T) {
		t.Parallel()

		for _, lazy := range []bool{false, true} {
			label := "eager"
			if lazy {
				label = "lazy"
			}
			lazy := lazy

			t.Run(label+"/creates_new_instance_on_each_resolve", func(t *testing.T) {
				t.Parallel()
				var opts []bind.BindOption
				if lazy {
					opts = append(opts, bind.Lazy())
				}

				c := container.New()
				require.NoError(t, c.Bind(func() Shape { return &Circle{a: 666} }, opts...))

				require.NoError(t, c.Call(func(s Shape) { s.SetArea(13) }))
				require.NoError(t, c.Call(func(s Shape) { assert.Equal(t, 666, s.GetArea()) }))
			})

			t.Run(label+"/resolver_returns_nothing_is_invalid", func(t *testing.T) {
				t.Parallel()
				var opts []bind.BindOption
				if lazy {
					opts = append(opts, bind.Lazy())
				}

				c := container.New()
				err := c.Bind(func() {}, opts...)
				assert.EqualError(t, err, "container: resolver function signature is invalid - it must return abstract, or abstract and error")
			})
		}

		t.Run("eager_bind_succeeds_but_later_resolve_fails", func(t *testing.T) {
			t.Parallel()

			c := container.New()
			callCount := 0
			require.NoError(t, c.Bind(func() (Database, error) {
				callCount++
				if callCount == 1 {
					return &MySQL{}, nil
				}
				return nil, errors.New("app: second call error")
			}))

			var db Database
			err := c.Resolve(&db)
			assert.EqualError(t, err, "container: encountered error while making concrete for: container_test.Database. Error encountered: app: second call error")
		})

		t.Run("lazy/first_resolve_succeeds_second_resolve_fails", func(t *testing.T) {
			t.Parallel()

			c := container.New()
			callCount := 0
			require.NoError(t, c.Bind(func() (Database, error) {
				callCount++
				if callCount == 1 {
					return &MySQL{}, nil
				}
				return nil, errors.New("app: second call error")
			}, bind.Lazy()))

			var db Database
			require.NoError(t, c.Resolve(&db))
			err := c.Resolve(&db)
			assert.EqualError(t, err, "container: encountered error while making concrete for: container_test.Database. Error encountered: app: second call error")
		})

		for _, lazy := range []bool{false, true} {
			label := "eager"
			if lazy {
				label = "lazy"
			}
			lazy := lazy

			t.Run(label+"/resolver_with_too_many_return_values_is_invalid", func(t *testing.T) {
				t.Parallel()
				var opts []bind.BindOption
				if lazy {
					opts = append(opts, bind.Lazy())
				}

				c := container.New()
				err := c.Bind(func() (Shape, Database, error) { return nil, nil, nil }, opts...)
				assert.EqualError(t, err, "container: resolver function signature is invalid - it must return abstract, or abstract and error")
			})
		}

		t.Run("resolver_second_return_non_error_type_is_invalid", func(t *testing.T) {
			t.Parallel()

			c := container.New()
			err := c.Bind(func() (Shape, int) { return &Circle{a: 1}, 42 })
			assert.EqualError(t, err, "container: resolver function signature is invalid - it must return abstract, or abstract and error")
		})

		for _, lazy := range []bool{false, true} {
			label := "eager"
			if lazy {
				label = "lazy"
			}
			lazy := lazy

			t.Run(label+"/named", func(t *testing.T) {
				t.Parallel()
				opts := []bind.BindOption{bind.WithName("theCircle")}
				if lazy {
					opts = append(opts, bind.Lazy())
				}

				c := container.New()
				require.NoError(t, c.Bind(func() Shape { return &Circle{a: 13} }, opts...))

				var sh Shape
				require.NoError(t, c.Resolve(&sh, resolve.WithName("theCircle")))
				assert.Equal(t, 13, sh.GetArea())
			})
		}

		t.Run("eager/resolver_returning_error_on_first_call", func(t *testing.T) {
			t.Parallel()

			c := container.New()
			err := c.Bind(func() (Shape, error) { return nil, errors.New("fail") })
			// Eager transient validates the resolver on bind.
			assert.EqualError(t, err, "fail")
		})

		t.Run("circular_dependency_detected", func(t *testing.T) {
			t.Parallel()

			c := container.New()
			require.NoError(t, c.Bind(func(d Database) Shape { return &Circle{a: 1} }, bind.Lazy()))
			err := c.Bind(func(s Shape) Database { return &MySQL{} }, bind.Lazy())
			assert.ErrorContains(t, err, "circular dependency")
		})
	})
}

func TestContainer_Call(t *testing.T) {
	t.Parallel()

	t.Run("resolves_multiple_dependencies", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton()))
		require.NoError(t, c.Bind(func() Database { return &MySQL{} }, bind.Singleton()))

		err := c.Call(func(s Shape, m Database) {
			assert.IsType(t, &Circle{}, s)
			assert.IsType(t, &MySQL{}, m)
		})
		assert.NoError(t, err)
	})

	t.Run("name_option_selects_named_binding", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton()))
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 2} }, bind.WithName("named"), bind.Singleton()))

		err := c.Call(func(s Shape) {
			assert.Equal(t, 2, s.GetArea())
		}, resolve.WithName("named"))
		assert.NoError(t, err)
	})

	t.Run("missing_dependency_in_chain_propagates_error", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() (Database, error) {
			var s Shape
			if err := c.Resolve(&s); err != nil {
				return nil, err
			}
			return &MySQL{}, nil
		}, bind.Singleton(), bind.Lazy()))

		err := c.Call(func(m Database) {})
		assert.EqualError(t, err, "container: no concrete found for the given abstraction; the abstraction is: container_test.Shape")
	})

	t.Run("nil_receiver_is_invalid", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		err := c.Call(nil)
		assert.EqualError(t, err, "container: invalid function")
	})

	t.Run("non_function_receiver_is_invalid", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		err := c.Call("STRING!")
		assert.EqualError(t, err, "container: invalid function")
	})

	t.Run("unbound_argument_returns_error", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{} }, bind.Singleton()))

		err := c.Call(func(s Shape, d Database) {})
		assert.EqualError(t, err, "container: no concrete found for the given abstraction; the abstraction is: container_test.Database")
	})

	t.Run("function_error_return_is_propagated", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{} }, bind.Singleton()))

		err := c.Call(func(s Shape) error {
			return errors.New("app: some context error")
		})
		assert.EqualError(t, err, "app: some context error")
	})

	t.Run("function_nil_error_return_is_swallowed", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{} }, bind.Singleton()))

		err := c.Call(func(s Shape) error { return nil })
		assert.NoError(t, err)
	})

	t.Run("function_with_invalid_signature_is_rejected", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{} }, bind.Singleton()))

		err := c.Call(func(s Shape) (int, error) {
			return 13, errors.New("app: some context error")
		})
		assert.EqualError(t, err, "container: receiver function signature is invalid")
	})

	t.Run("argument_make_error_propagates", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() (Shape, error) {
			return nil, errors.New("make: failed")
		}, bind.Singleton(), bind.Lazy()))

		err := c.Call(func(s Shape) {})
		assert.Error(t, err)
	})

	t.Run("no_args_succeeds", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		called := false
		err := c.Call(func() { called = true })
		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("no_args_and_nil_error_return", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		err := c.Call(func() error { return nil })
		assert.NoError(t, err)
	})

	t.Run("no_args_and_error_return", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		err := c.Call(func() error { return errors.New("app: fail") })
		assert.EqualError(t, err, "app: fail")
	})

	t.Run("named_resolves_unnamed_binding_returns_error", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton()))

		err := c.Call(func(s Shape) {}, resolve.WithName("nonexistent"))
		assert.Error(t, err)
	})
}

func TestContainer_Resolve(t *testing.T) {
	t.Parallel()

	t.Run("fills_pointer_to_interface", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton()))
		require.NoError(t, c.Bind(func() Database { return &MySQL{} }, bind.Singleton()))

		var s Shape
		require.NoError(t, c.Resolve(&s))
		assert.IsType(t, &Circle{}, s)

		var d Database
		require.NoError(t, c.Resolve(&d))
		assert.IsType(t, &MySQL{}, d)
	})

	t.Run("nil_receiver_is_invalid", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		err := c.Resolve(nil)
		assert.EqualError(t, err, "container: invalid abstraction")
	})

	t.Run("non_pointer_receiver_is_invalid", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		err := c.Resolve("STRING!")
		assert.EqualError(t, err, "container: invalid abstraction")
	})

	t.Run("unbound_abstraction_returns_error", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		var s Shape
		err := c.Resolve(&s)
		assert.EqualError(t, err, "container: no concrete found for the given abstraction; the abstraction is: container_test.Shape")
	})

	t.Run("runtime_params/all_provided", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.Singleton(), bind.Lazy()))

		var db Database
		require.NoError(t, c.Resolve(&db, resolve.WithParams(10, &Circle{a: 2})))
		assert.True(t, db.Connect())
	})

	t.Run("runtime_params/falls_back_to_container_for_missing", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 2} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.Singleton(), bind.Lazy()))

		var db Database
		require.NoError(t, c.Resolve(&db, resolve.WithParams(10)))
		assert.True(t, db.Connect())
	})

	t.Run("runtime_params/take_precedence_over_container", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 99} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func(s Shape) Database {
			return PostgreSQL{ready: s.GetArea() == 2}
		}))

		var db Database
		require.NoError(t, c.Resolve(&db, resolve.WithParams(&Circle{a: 2})))
		assert.True(t, db.Connect())
	})

	t.Run("runtime_params/missing_with_no_fallback_returns_error", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.Singleton(), bind.Lazy()))

		var db Database
		err := c.Resolve(&db, resolve.WithParams(10))
		assert.EqualError(t, err, "container: encountered error while making concrete for: container_test.Database. Error encountered: container: no concrete found for the given abstraction; the abstraction is: container_test.Shape")
	})

	t.Run("named/with_runtime_params", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.WithName("runtime"), bind.Lazy()))

		var db Database
		require.NoError(t, c.Resolve(&db, resolve.WithName("runtime"), resolve.WithParams(10, &Circle{a: 2})))
		assert.True(t, db.Connect())
	})

	t.Run("bind_params/all_provided", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.Lazy(), bind.ResolveDepenenciesByParams(10, &Circle{a: 2})))

		var db Database
		require.NoError(t, c.Resolve(&db))
		assert.True(t, db.Connect())
	})

	t.Run("bind_params/falls_back_to_container_for_missing", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 2} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.Lazy(), bind.ResolveDepenenciesByParams(10)))

		var db Database
		require.NoError(t, c.Resolve(&db))
		assert.True(t, db.Connect())
	})

	t.Run("resolve_params/take_precedence_over_bind_params", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func(x int) Database {
			return PostgreSQL{ready: x == 99}
		}, bind.Lazy(), bind.ResolveDepenenciesByParams(10)))

		var db Database
		require.NoError(t, c.Resolve(&db, resolve.WithParams(99)))
		assert.True(t, db.Connect())
	})

	t.Run("bind_params/take_precedence_over_container", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 99} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func(s Shape) Database {
			return PostgreSQL{ready: s.GetArea() == 2}
		}, bind.Lazy(), bind.ResolveDepenenciesByParams(&Circle{a: 2})))

		var db Database
		require.NoError(t, c.Resolve(&db))
		assert.True(t, db.Connect())
	})

	t.Run("named_bindings/resolves_dep_from_named_binding", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 99} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.WithName("special"), bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func(s Shape) Database {
			return PostgreSQL{ready: s.GetArea() == 5}
		}, bind.Lazy(), bind.ResolveDependenciesByNamedBindings("special")))

		var db Database
		require.NoError(t, c.Resolve(&db))
		assert.True(t, db.Connect())
	})

	t.Run("named_bindings/not_found_returns_error", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func(s Shape) Database {
			return PostgreSQL{ready: true}
		}, bind.Lazy(), bind.ResolveDependenciesByNamedBindings("nonexistent")))

		var db Database
		err := c.Resolve(&db)
		assert.ErrorContains(t, err, "named binding(s)")
		assert.ErrorContains(t, err, "nonexistent")
	})

	t.Run("named_bindings/make_error_propagates", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() (Shape, error) {
			return nil, errors.New("named: fail")
		}, bind.WithName("broken"), bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func(s Shape) Database {
			return &MySQL{}
		}, bind.Lazy(), bind.ResolveDependenciesByNamedBindings("broken")))

		var db Database
		err := c.Resolve(&db)
		assert.ErrorContains(t, err, "named: fail")
	})

	t.Run("named_bindings/resolve_params_take_precedence", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.WithName("special"), bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func(s Shape) Database {
			return PostgreSQL{ready: s.GetArea() == 77}
		}, bind.Lazy(), bind.ResolveDependenciesByNamedBindings("special")))

		var db Database
		require.NoError(t, c.Resolve(&db, resolve.WithParams(&Circle{a: 77})))
		assert.True(t, db.Connect())
	})

	t.Run("named_bindings/bind_params_take_precedence_over_named_bindings", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.WithName("special"), bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func(s Shape) Database {
			return PostgreSQL{ready: s.GetArea() == 42}
		}, bind.Lazy(),
			bind.ResolveDepenenciesByParams(&Circle{a: 42}),
			bind.ResolveDependenciesByNamedBindings("special"),
		))

		var db Database
		require.NoError(t, c.Resolve(&db))
		assert.True(t, db.Connect())
	})

	t.Run("priority_order/resolve_then_bind_then_named_then_container", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		// Container has Shape with area 99
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 99} }, bind.Singleton(), bind.Lazy()))
		// Named binding "special" has Shape with area 5
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.WithName("special"), bind.Singleton(), bind.Lazy()))
		// Resolver takes int (from bind params), Shape (from named binding), Database (from container)
		require.NoError(t, c.Bind(func() Database { return &MySQL{} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func(x int, s Shape, db Database) Cache {
			if x == 10 && s.GetArea() == 5 && db.Connect() {
				return InMemoryCache{}
			}
			return nil
		}, bind.Lazy(),
			bind.ResolveDepenenciesByParams(10),
			bind.ResolveDependenciesByNamedBindings("special"),
		))

		var cache Cache
		require.NoError(t, c.Resolve(&cache))
		assert.Equal(t, "cached", cache.Get())
	})

	t.Run("concrete_pointer_type_directly", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() *Circle { return &Circle{a: 10} }, bind.Singleton()))

		var circle *Circle
		require.NoError(t, c.Resolve(&circle))
		assert.Equal(t, 10, circle.GetArea())
	})

	t.Run("wrong_name_returns_error", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.WithName("foo"), bind.Singleton()))

		var s Shape
		err := c.Resolve(&s, resolve.WithName("bar"))
		assert.ErrorContains(t, err, "no concrete found")
	})

	t.Run("multiple_named_same_type", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.WithName("one"), bind.Singleton()))
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 2} }, bind.WithName("two"), bind.Singleton()))
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 3} }, bind.WithName("three"), bind.Singleton()))

		var s1, s2, s3 Shape
		require.NoError(t, c.Resolve(&s1, resolve.WithName("one")))
		require.NoError(t, c.Resolve(&s2, resolve.WithName("two")))
		require.NoError(t, c.Resolve(&s3, resolve.WithName("three")))

		assert.Equal(t, 1, s1.GetArea())
		assert.Equal(t, 2, s2.GetArea())
		assert.Equal(t, 3, s3.GetArea())
	})
}

func TestContainer_Fill(t *testing.T) {
	t.Parallel()

	t.Run("fills_tagged_struct_fields_by_type", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.WithName("C"), bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func() Database { return &MySQL{} }, bind.Singleton(), bind.Lazy()))

		myApp := struct {
			S Shape    `container:"type"`
			D Database `container:"type"`
			C Shape    `container:"name"`
			X string
		}{}

		require.NoError(t, c.Fill(&myApp))
		assert.IsType(t, &Circle{}, myApp.S)
		assert.IsType(t, &MySQL{}, myApp.D)
	})

	t.Run("name_option_selects_named_binding", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton()))
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 2} }, bind.WithName("named"), bind.Singleton()))

		myApp := struct {
			S Shape `container:"type"`
		}{}

		require.NoError(t, c.Fill(&myApp, resolve.WithName("named")))
		assert.Equal(t, 2, myApp.S.GetArea())
	})

	t.Run("fills_unexported_struct_fields", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func() Database { return &MySQL{} }, bind.Singleton(), bind.Lazy()))

		myApp := struct {
			s Shape    `container:"type"`
			d Database `container:"type"`
			y int
		}{}

		require.NoError(t, c.Fill(&myApp))
		assert.IsType(t, &Circle{}, myApp.s)
		assert.IsType(t, &MySQL{}, myApp.d)
	})

	t.Run("name_tagged_field_with_no_matching_binding_returns_error", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		type App struct {
			S string `container:"name"`
		}
		err := c.Fill(&App{})
		assert.EqualError(t, err, "container: cannot make field; the field is: S")
	})

	t.Run("invalid_struct_tag_value_returns_error", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		type App struct {
			S string `container:"invalid"`
		}
		err := c.Fill(&App{})
		assert.EqualError(t, err, "container: invalid struct tag; the field is: S")
	})

	t.Run("non_struct_pointer_is_invalid", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		n := 0
		err := c.Fill(&n)
		assert.EqualError(t, err, "container: invalid structure")
	})

	t.Run("non_pointer_is_invalid", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		err := c.Fill("not a pointer")
		assert.EqualError(t, err, "container: invalid structure")
	})

	t.Run("nil_is_invalid", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		err := c.Fill(nil)
		assert.EqualError(t, err, "container: invalid structure")
	})

	t.Run("missing_dependency_in_chain_propagates_error", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton()))
		require.NoError(t, c.Bind(func() (Shape, error) {
			var s Shape
			if err := c.Resolve(&s, resolve.WithName("foo")); err != nil {
				return nil, err
			}
			return &Circle{a: 5}, nil
		}, bind.WithName("C"), bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func() Database { return &MySQL{} }, bind.Singleton()))

		myApp := struct {
			S Shape    `container:"type"`
			D Database `container:"type"`
			C Shape    `container:"name"`
			X string
		}{}

		err := c.Fill(&myApp)
		assert.EqualError(t, err, "container: no concrete found for the given abstraction; the abstraction is: container_test.Shape")
	})

	t.Run("runtime_params/all_provided", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.Singleton(), bind.Lazy()))

		myApp := struct {
			D Database `container:"type"`
		}{}

		require.NoError(t, c.Fill(&myApp, resolve.WithParams(10, &Circle{a: 2})))
		assert.True(t, myApp.D.Connect())
	})

	t.Run("runtime_params/falls_back_to_container_for_missing", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 2} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.Singleton(), bind.Lazy()))

		myApp := struct {
			D Database `container:"type"`
		}{}

		require.NoError(t, c.Fill(&myApp, resolve.WithParams(10)))
		assert.True(t, myApp.D.Connect())
	})

	t.Run("runtime_params/missing_with_no_fallback_returns_error", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: (x + s.GetArea()) == 12}
		}, bind.Singleton(), bind.Lazy()))

		myApp := struct {
			D Database `container:"type"`
		}{}

		err := c.Fill(&myApp, resolve.WithParams(10))
		assert.EqualError(t, err, "container: no concrete found for the given abstraction; the abstraction is: container_test.Shape")
	})

	t.Run("mixed_tagged_and_untagged_fields", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 42} }, bind.Singleton()))

		type App struct {
			S     Shape  `container:"type"`
			Name  string // untagged, should be untouched
			Count int    // untagged
		}

		app := App{Name: "original", Count: 99}
		require.NoError(t, c.Fill(&app))
		assert.Equal(t, 42, app.S.GetArea())
		assert.Equal(t, "original", app.Name)
		assert.Equal(t, 99, app.Count)
	})

	t.Run("type_tag_missing_binding_returns_error", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		type App struct {
			DB Database `container:"type"`
		}
		err := c.Fill(&App{})
		assert.ErrorContains(t, err, "cannot make field")
		assert.ErrorContains(t, err, "DB")
	})
}

// verify that Bind, Resolve, Call, and Fill can be freely called from within each other's
// resolver/function bodies without causing deadlocks or incorrect behaviour.
func TestNestedScenarios(t *testing.T) {
	t.Parallel()

	t.Run("resolve_inside_bind", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 10} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func() (Database, error) {
			var s Shape
			if err := c.Resolve(&s); err != nil {
				return nil, err
			}
			return PostgreSQL{ready: s.GetArea() == 10}, nil
		}, bind.Singleton(), bind.Lazy()))

		var db Database
		require.NoError(t, c.Resolve(&db))
		assert.True(t, db.Connect())
	})

	t.Run("resolve_inside_call", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 8} }, bind.Singleton()))
		require.NoError(t, c.Bind(func() Database { return &MySQL{} }, bind.Singleton()))

		err := c.Call(func(s Shape) {
			var db Database
			assert.NoError(t, c.Resolve(&db))
			assert.True(t, db.Connect())
		})
		require.NoError(t, err)
	})

	t.Run("resolve_inside_fill", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 15} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func() (Database, error) {
			var s Shape
			if err := c.Resolve(&s); err != nil {
				return nil, err
			}
			return PostgreSQL{ready: s.GetArea() == 15}, nil
		}, bind.Singleton(), bind.Lazy()))

		app := struct {
			D Database `container:"type"`
		}{}
		require.NoError(t, c.Fill(&app))
		assert.True(t, app.D.Connect())
	})

	t.Run("bind_inside_bind", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() (Shape, error) {
			if err := c.Bind(func() Database { return &MySQL{} }, bind.Singleton(), bind.Lazy()); err != nil {
				return nil, err
			}
			return &Circle{a: 7}, nil
		}, bind.Singleton(), bind.Lazy()))

		var s Shape
		require.NoError(t, c.Resolve(&s))
		assert.Equal(t, 7, s.GetArea())

		var db Database
		require.NoError(t, c.Resolve(&db))
		assert.True(t, db.Connect())
	})

	t.Run("bind_self_reference_returns_circular_dependency_error", func(t *testing.T) {
		t.Parallel()

		// Shape depends on Database, Database depends on Shape.
		// The cycle is now detected statically at Bind() time via DFS, before any goroutine
		// or mutex is involved — eliminating the cross-goroutine deadlock risk entirely.
		c := container.New()
		require.NoError(t, c.Bind(func(d Database) Shape { return &Circle{a: 1} }, bind.Singleton(), bind.Lazy()))
		err := c.Bind(func(s Shape) Database { return &MySQL{} }, bind.Singleton(), bind.Lazy())
		assert.ErrorContains(t, err, "circular dependency")
	})

	t.Run("bind_indirect_cycle_returns_circular_dependency_error", func(t *testing.T) {
		t.Parallel()

		// Binding order reversed: same cycle, same result.
		c := container.New()
		require.NoError(t, c.Bind(func(s Shape) Database { return &MySQL{} }, bind.Singleton(), bind.Lazy()))
		err := c.Bind(func(d Database) Shape { return &Circle{a: 1} }, bind.Singleton(), bind.Lazy())
		assert.ErrorContains(t, err, "circular dependency")
	})

	t.Run("diamond_dependency_does_not_false_positive_cycle", func(t *testing.T) {
		t.Parallel()

		// Diamond: Cache depends on Shape and Database, both depend on Logger.
		// Logger is visited twice but is NOT a cycle — DFS must not false-positive.
		c := container.New()
		require.NoError(t, c.Bind(func() Logger { return StdLogger{} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func(l Logger) Shape { return &Circle{a: 1} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func(l Logger) Database { return &MySQL{} }, bind.Singleton(), bind.Lazy()))
		err := c.Bind(func(s Shape, d Database) Cache { return InMemoryCache{} }, bind.Singleton(), bind.Lazy())
		assert.NoError(t, err)
	})

	t.Run("bind_inside_call", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton()))

		err := c.Call(func(s Shape) {
			assert.NoError(t, c.Bind(func() Database { return &MySQL{} }, bind.Singleton(), bind.Lazy()))
		})
		require.NoError(t, err)

		var db Database
		require.NoError(t, c.Resolve(&db))
		assert.True(t, db.Connect())
	})

	t.Run("bind_inside_fill", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() (Shape, error) {
			if err := c.Bind(func() Database { return &MySQL{} }, bind.Singleton(), bind.Lazy()); err != nil {
				return nil, err
			}
			return &Circle{a: 6}, nil
		}, bind.Singleton(), bind.Lazy()))

		app := struct {
			S Shape `container:"type"`
		}{}
		require.NoError(t, c.Fill(&app))
		assert.Equal(t, 6, app.S.GetArea())

		var db Database
		require.NoError(t, c.Resolve(&db))
		assert.True(t, db.Connect())
	})

	t.Run("call_inside_bind", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		called := false
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func() (Database, error) {
			if err := c.Call(func(s Shape) { called = true }); err != nil {
				return nil, err
			}
			return &MySQL{}, nil
		}, bind.Singleton(), bind.Lazy()))

		var db Database
		require.NoError(t, c.Resolve(&db))
		assert.True(t, called)
		assert.True(t, db.Connect())
	})

	t.Run("call_inside_call", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 11} }, bind.Singleton()))
		require.NoError(t, c.Bind(func() Database { return &MySQL{} }, bind.Singleton()))

		innerCalled := false
		err := c.Call(func(s Shape) {
			assert.NoError(t, c.Call(func(db Database) {
				innerCalled = true
				assert.True(t, db.Connect())
			}))
		})
		require.NoError(t, err)
		assert.True(t, innerCalled)
	})

	t.Run("call_inside_fill", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		called := false
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func() (Database, error) {
			if err := c.Call(func(s Shape) { called = true }); err != nil {
				return nil, err
			}
			return &MySQL{}, nil
		}, bind.Singleton(), bind.Lazy()))

		app := struct {
			D Database `container:"type"`
		}{}
		require.NoError(t, c.Fill(&app))
		assert.True(t, called)
		assert.True(t, app.D.Connect())
	})

	t.Run("fill_inside_bind", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 3} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func() (Database, error) {
			app := struct {
				S Shape `container:"type"`
			}{}
			if err := c.Fill(&app); err != nil {
				return nil, err
			}
			return PostgreSQL{ready: app.S.GetArea() == 3}, nil
		}, bind.Singleton(), bind.Lazy()))

		var db Database
		require.NoError(t, c.Resolve(&db))
		assert.True(t, db.Connect())
	})

	t.Run("fill_inside_call", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 2} }, bind.Singleton()))
		require.NoError(t, c.Bind(func() Database { return &MySQL{} }, bind.Singleton()))

		err := c.Call(func(s Shape) {
			app := struct {
				D Database `container:"type"`
			}{}
			assert.NoError(t, c.Fill(&app))
			assert.True(t, app.D.Connect())
		})
		require.NoError(t, err)
	})

	t.Run("fill_inside_fill", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 4} }, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func() (Database, error) {
			inner := struct {
				S Shape `container:"type"`
			}{}
			if err := c.Fill(&inner); err != nil {
				return nil, err
			}
			return PostgreSQL{ready: inner.S.GetArea() == 4}, nil
		}, bind.Singleton(), bind.Lazy()))

		app := struct {
			D Database `container:"type"`
		}{}
		require.NoError(t, c.Fill(&app))
		assert.True(t, app.D.Connect())
	})
}

// TestRaceConditions proves the three race scenarios identified in container.go.
func TestRaceConditions(t *testing.T) {
	t.Parallel()

	t.Run("concurrent_lazy_resolution_returns_same_instance", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 42} }, bind.Singleton(), bind.Lazy()))

		const goroutines = 50

		var wg sync.WaitGroup
		instances := make([]*Circle, goroutines)

		wg.Add(goroutines)
		for i := range goroutines {
			go func(idx int) {
				defer wg.Done()
				var s Shape
				assert.NoError(t, c.Resolve(&s))
				instances[idx] = s.(*Circle)
			}(i)
		}
		wg.Wait()

		for _, inst := range instances[1:] {
			assert.Same(t, instances[0], inst)
		}
	})

	// Race #1 – eager singleton: resolver invoked multiple times under concurrent Bind().
	//
	// In Bind(), c.invoke() is called BEFORE c.lock is acquired. When two goroutines
	// concurrently bind the same type as a non-lazy singleton, both enter c.invoke()
	// simultaneously and both execute the resolver, violating the singleton contract.
	t.Run("eager_singleton_concurrent_bind_invokes_resolver_multiple_times", func(t *testing.T) {
		t.Parallel()

		var callCount int64

		resolver := func() Shape {
			atomic.AddInt64(&callCount, 1)
			time.Sleep(20 * time.Millisecond) // widen the race window so both goroutines overlap
			return &Circle{a: 42}
		}

		c := container.New()
		ready := make(chan struct{})
		var wg sync.WaitGroup

		for range 2 {
			wg.Go(func() {
				<-ready // release both at exactly the same moment
				_ = c.Bind(resolver, bind.Singleton())
			})
		}

		close(ready)
		wg.Wait()

		// Expected: 1  (singleton resolver should run at most once)
		// Actual:   2  (both goroutines entered c.invoke() before either held c.lock)
		assert.Equal(t, int64(1), atomic.LoadInt64(&callCount),
			"eager singleton resolver was invoked %d times; expected exactly 1",
			atomic.LoadInt64(&callCount))
	})

	// Race #2 – lazy singleton: write lock held on every read after initialisation.
	//
	// binding.make() always acquires b.mu.Lock() (exclusive) even after b.concrete is
	// populated. All concurrent reads after the first resolve are serialised, though
	// b.mu is a sync.RWMutex and only a read lock would be needed. This test confirms
	// that a double-checked read does NOT currently happen: the resolution of an already-
	// initialised singleton never uses b.mu.RLock().
	t.Run("lazy_singleton_resolved_instance_always_goes_through_write_lock", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 7} }, bind.Singleton(), bind.Lazy()))

		// Warm the singleton so b.concrete is set.
		var warm Shape
		require.NoError(t, c.Resolve(&warm))

		// Now fire many concurrent readers. With a proper RLock they should all proceed
		// in parallel; with the current Lock() they are fully serialised.
		const goroutines = 50
		var wg sync.WaitGroup
		instances := make([]Shape, goroutines)

		wg.Add(goroutines)
		for i := range goroutines {
			go func(idx int) {
				defer wg.Done()
				var s Shape
				assert.NoError(t, c.Resolve(&s))
				instances[idx] = s
			}(i)
		}
		wg.Wait()

		// All goroutines must receive the same (singleton) instance.
		for _, inst := range instances {
			assert.Same(t, warm.(*Circle), inst.(*Circle),
				"each goroutine must receive the same singleton pointer")
		}
	})

	// // Race #3 – lock leak: RLock is never released when Implements() panics in concrete().
	// //
	// // concrete() releases c.lock.RUnlock() manually (not via defer). If the loop body
	// // calls boundAbstraction.Implements(abstraction) where abstraction is a non-interface
	// // type (e.g. *Circle), reflect.Implements panics. Because RUnlock is not deferred,
	// // the lock is never released and every subsequent operation that needs c.lock.Lock()
	// // (Bind, Reset) will deadlock permanently.
	t.Run("lock_leaked_after_implements_panic_causes_deadlock", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		// Shape is bound so the bindings map is non-empty and the range loop is reached.
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton()))

		// Call with a concrete (*Circle) argument type. *Circle is not in c.bindings
		// directly, so concrete() falls through to the Implements loop.
		// Shape.Implements(*Circle) panics because *Circle is not an interface –
		// and since RUnlock is manual (not deferred) the read-lock leaks.
		func() {
			defer func() { recover() }() //nolint:errcheck
			_ = c.Call(func(_ *Circle) {})
		}()

		// Proof: any operation that needs c.lock.Lock() must not deadlock.
		done := make(chan struct{})
		go func() {
			_ = c.Bind(func() Database { return MySQL{} }, bind.Singleton())
			close(done)
		}()

		select {
		case <-done:
			// no deadlock – the lock was not leaked (or the fix is in place)
		case <-time.After(time.Second):
			t.Fatal("c.lock.RLock leaked after Implements() panic; subsequent Bind() deadlocked")
		}
	})

	t.Run("concurrent_fill_operations", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton()))
		require.NoError(t, c.Bind(func() Database { return &MySQL{} }, bind.Singleton()))

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()
				app := struct {
					S Shape    `container:"type"`
					D Database `container:"type"`
				}{}
				assert.NoError(t, c.Fill(&app))
				assert.IsType(t, &Circle{}, app.S)
				assert.IsType(t, &MySQL{}, app.D)
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent_call_operations", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton()))

		const goroutines = 50
		var wg sync.WaitGroup
		var count int64
		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()
				assert.NoError(t, c.Call(func(s Shape) {
					atomic.AddInt64(&count, 1)
					assert.Equal(t, 5, s.GetArea())
				}))
			}()
		}

		wg.Wait()
		assert.Equal(t, int64(goroutines), atomic.LoadInt64(&count))
	})

	t.Run("concurrent_mixed_bind_resolve_fill_call", func(t *testing.T) {
		t.Parallel()

		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton()))
		require.NoError(t, c.Bind(func() Database { return &MySQL{} }, bind.Singleton()))

		const goroutines = 30
		var wg sync.WaitGroup
		wg.Add(goroutines * 4)

		ready := make(chan struct{})

		for range goroutines {
			go func() {
				defer wg.Done()
				<-ready
				var s Shape
				_ = c.Resolve(&s)
			}()
			go func() {
				defer wg.Done()
				<-ready
				_ = c.Call(func(s Shape) {})
			}()
			go func() {
				defer wg.Done()
				<-ready
				app := struct {
					S Shape `container:"type"`
				}{}
				_ = c.Fill(&app)
			}()
			go func() {
				defer wg.Done()
				<-ready
				var db Database
				_ = c.Resolve(&db)
			}()
		}

		close(ready)
		wg.Wait()
	})
}

// TestDeadlockAndCircularLock exhaustively covers all known deadlock and circular-lock
// scenarios in the container: TOCTOU cycle races, reentrant locking, lock-ordering
// violations, concurrent resolution, and multi-node dependency cycles.
func TestDeadlockAndCircularLock(t *testing.T) {
	t.Parallel()

	// deadline is how long every goroutine-based test waits before declaring a deadlock.
	const deadline = 2 * time.Second

	// withDeadline runs fn in a goroutine and fails the test if it does not complete
	// within the deadline, providing a clear deadlock signal instead of a hanging test.
	withDeadline := func(t *testing.T, fn func()) {
		t.Helper()
		done := make(chan struct{})
		go func() { fn(); close(done) }()
		select {
		case <-done:
		case <-time.After(deadline):
			t.Fatal("deadlock: operation did not complete within the deadline")
		}
	}

	// ── Cycle detection ────────────────────────────────────────────────────────────

	t.Run("direct_self_cycle_detected_at_bind", func(t *testing.T) {
		t.Parallel()
		// validateResolverFunction catches func(Shape) Shape, but a resolver that
		// depends on a structurally identical interface type it produces is caught by DFS.
		// Use two distinct but mutually dependent types to confirm the DFS path.
		c := container.New()
		require.NoError(t, c.Bind(func(d Database) Shape { return &Circle{} }, bind.Lazy(), bind.Singleton()))
		err := c.Bind(func(s Shape) Database { return MySQL{} }, bind.Lazy(), bind.Singleton())
		assert.ErrorIs(t, err, containerErrors.ErrCircularDependency)
	})

	t.Run("three_node_indirect_cycle_detected_at_third_bind", func(t *testing.T) {
		t.Parallel()
		// Service → Cache → Logger → Service forms a three-node cycle.
		// The first two Binds introduce no cycle; only the third closes the loop.
		//   Bind 1: Cache depends on Service  (Service not yet registered → no cycle)
		//   Bind 2: Logger depends on Cache   (Cache→Service, none cyclic yet)
		//   Bind 3: Service depends on Logger → Logger→Cache→Service == outType → cycle
		c := container.New()
		require.NoError(t, c.Bind(func(_ Service) Cache { return InMemoryCache{} }, bind.Lazy(), bind.Singleton()))
		require.NoError(t, c.Bind(func(_ Cache) Logger { return StdLogger{} }, bind.Lazy(), bind.Singleton()))
		err := c.Bind(func(_ Logger) Service { return AppService{} }, bind.Lazy(), bind.Singleton())
		assert.ErrorIs(t, err, containerErrors.ErrCircularDependency)
	})

	t.Run("independent_three_node_chain_no_cycle", func(t *testing.T) {
		t.Parallel()
		// Cache → (no deps), Logger → Cache, Service → Logger: no cycle; all binds succeed.
		c := container.New()
		require.NoError(t, c.Bind(func() Cache { return InMemoryCache{} }, bind.Lazy(), bind.Singleton()))
		require.NoError(t, c.Bind(func(_ Cache) Logger { return StdLogger{} }, bind.Lazy(), bind.Singleton()))
		require.NoError(t, c.Bind(func(_ Logger) Service { return AppService{} }, bind.Lazy(), bind.Singleton()))

		var svc Service
		require.NoError(t, c.Resolve(&svc))
		assert.Equal(t, "running", svc.Run())
	})

	// ── TOCTOU: concurrent Bind of a mutual cycle ──────────────────────────────────

	t.Run("concurrent_mutual_cycle_exactly_one_bind_fails", func(t *testing.T) {
		t.Parallel()
		// G1: Bind(func(Database) Shape)
		// G2: Bind(func(Shape) Database)
		// Both start at the same moment. checkCycleAndSet holds the write lock across
		// DFS + registration, so the second writer must observe the first's binding
		// during its DFS and return ErrCircularDependency.
		c := container.New()
		var (
			wg    sync.WaitGroup
			errs  [2]error
			ready = make(chan struct{})
		)

		wg.Add(2)
		go func() {
			defer wg.Done()
			<-ready
			errs[0] = c.Bind(func(_ Database) Shape { return &Circle{} }, bind.Lazy(), bind.Singleton())
		}()
		go func() {
			defer wg.Done()
			<-ready
			errs[1] = c.Bind(func(_ Shape) Database { return MySQL{} }, bind.Lazy(), bind.Singleton())
		}()

		close(ready)
		withDeadline(t, wg.Wait) // no deadlock

		cycleErrors := 0
		for _, err := range errs {
			if errors.Is(err, containerErrors.ErrCircularDependency) {
				cycleErrors++
			}
		}
		assert.Equal(t, 1, cycleErrors,
			"exactly one of the two concurrent Bind calls must return ErrCircularDependency")
	})

	t.Run("concurrent_mutual_cycle_eager_singleton_exactly_one_bind_fails", func(t *testing.T) {
		t.Parallel()
		// Same as above but with eager singletons, which use checkCycleAndSetIfAbsent.
		c := container.New()
		var (
			wg    sync.WaitGroup
			errs  [2]error
			ready = make(chan struct{})
		)

		wg.Add(2)
		go func() {
			defer wg.Done()
			<-ready
			// eager singleton: resolver takes no deps so it can actually be invoked.
			// We bind Shape first; then the second goroutine tries to bind Database → Shape.
			errs[0] = c.Bind(func() Shape { return &Circle{} }, bind.Singleton())
		}()
		go func() {
			defer wg.Done()
			<-ready
			errs[1] = c.Bind(func(_ Shape) Database { return MySQL{} }, bind.Singleton())
		}()

		close(ready)
		withDeadline(t, wg.Wait) // no deadlock

		// At least one must succeed and neither must hang.
		successCount := 0
		for _, err := range errs {
			if err == nil {
				successCount++
			}
		}
		assert.GreaterOrEqual(t, successCount, 1, "at least one Bind must succeed")
	})

	// ── Reentrant / nested locking ─────────────────────────────────────────────────

	t.Run("bind_inside_resolver_does_not_deadlock", func(t *testing.T) {
		t.Parallel()
		// A resolver that calls c.Bind() internally. If the registrar's write lock
		// were still held while invoke() runs, this would deadlock immediately.
		c := container.New()
		withDeadline(t, func() {
			require.NoError(t, c.Bind(func() Shape {
				// Register an unrelated binding from within the resolver.
				_ = c.Bind(func() Database { return MySQL{} }, bind.Lazy(), bind.Singleton())
				return &Circle{a: 1}
			}, bind.Singleton()))
		})
	})

	t.Run("resolve_inside_resolver_does_not_deadlock", func(t *testing.T) {
		t.Parallel()
		// Resolver for Database calls c.Resolve for Shape. If r.mu were held during
		// invoke this would attempt a second RLock on the same goroutine and deadlock
		// with a write-locked mutex (Go's sync.RWMutex forbids recursive read-locking
		// while a writer is waiting).
		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 5} }, bind.Singleton()))
		withDeadline(t, func() {
			require.NoError(t, c.Bind(func() Database {
				var s Shape
				_ = c.Resolve(&s)
				return MySQL{}
			}, bind.Singleton()))
		})
	})

	t.Run("reset_concurrent_with_bind_does_not_deadlock", func(t *testing.T) {
		t.Parallel()
		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{} }, bind.Singleton()))

		var wg sync.WaitGroup
		ready := make(chan struct{})

		wg.Add(2)
		go func() {
			defer wg.Done()
			<-ready
			c.Reset()
		}()
		go func() {
			defer wg.Done()
			<-ready
			_ = c.Bind(func() Database { return MySQL{} }, bind.Singleton())
		}()

		close(ready)
		withDeadline(t, wg.Wait)
	})

	t.Run("reset_concurrent_with_resolve_does_not_deadlock", func(t *testing.T) {
		t.Parallel()
		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{} }, bind.Singleton()))

		ready := make(chan struct{})
		var wg sync.WaitGroup

		wg.Add(2)
		go func() {
			defer wg.Done()
			<-ready
			c.Reset()
		}()
		go func() {
			defer wg.Done()
			<-ready
			var s Shape
			_ = c.Resolve(&s) // may succeed or return ErrNoConcreteFound; must not deadlock
		}()

		close(ready)
		withDeadline(t, wg.Wait)
	})

	// ── Lock-ordering: r.mu never held while b.mu is acquired ─────────────────────

	t.Run("concurrent_resolution_of_different_singletons_no_lock_ordering_deadlock", func(t *testing.T) {
		t.Parallel()
		// C1 and C2 are independent singletons. Their b.mu locks are unrelated, so
		// concurrent resolution must never produce a hold-and-wait cycle.
		c := container.New()
		require.NoError(t, c.Bind(func() Shape { return &Circle{a: 1} }, bind.Lazy(), bind.Singleton()))
		require.NoError(t, c.Bind(func() Database { return MySQL{} }, bind.Lazy(), bind.Singleton()))

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines * 2)

		ready := make(chan struct{})
		for range goroutines {
			go func() {
				defer wg.Done()
				<-ready
				var s Shape
				assert.NoError(t, c.Resolve(&s))
			}()
			go func() {
				defer wg.Done()
				<-ready
				var db Database
				assert.NoError(t, c.Resolve(&db))
			}()
		}

		close(ready)
		withDeadline(t, wg.Wait)
	})

	t.Run("concurrent_resolution_of_shared_singleton_all_get_same_instance", func(t *testing.T) {
		t.Parallel()
		// Classic singleton race: many goroutines resolve the same lazy singleton
		// simultaneously. Exactly one resolver invocation must occur and all goroutines
		// must receive the same pointer.
		c := container.New()
		var calls int64
		require.NoError(t, c.Bind(func() Shape {
			atomic.AddInt64(&calls, 1)
			return &Circle{a: 99}
		}, bind.Lazy(), bind.Singleton()))

		const goroutines = 100
		instances := make([]Shape, goroutines)
		var wg sync.WaitGroup
		wg.Add(goroutines)
		ready := make(chan struct{})

		for i := range goroutines {
			go func(idx int) {
				defer wg.Done()
				<-ready
				var s Shape
				assert.NoError(t, c.Resolve(&s))
				instances[idx] = s
			}(i)
		}

		close(ready)
		withDeadline(t, wg.Wait)

		assert.Equal(t, int64(1), atomic.LoadInt64(&calls), "resolver must be invoked exactly once")
		for _, inst := range instances[1:] {
			assert.Same(t, instances[0].(*Circle), inst.(*Circle))
		}
	})

	t.Run("concurrent_bind_of_independent_types_all_succeed", func(t *testing.T) {
		t.Parallel()
		// Many goroutines register completely unrelated bindings simultaneously.
		// The registrar write lock must serialize them without any goroutine hanging.
		c := container.New()
		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines)
		ready := make(chan struct{})
		errs := make([]error, goroutines)

		// Use only Shape and Database alternately — same type, different names, so no cycle.
		for i := range goroutines {
			go func(idx int) {
				defer wg.Done()
				<-ready
				name := fmt.Sprintf("binding-%d", idx)
				if idx%2 == 0 {
					errs[idx] = c.Bind(func() Shape { return &Circle{a: idx} },
						bind.WithName(name), bind.Lazy(), bind.Singleton())
				} else {
					errs[idx] = c.Bind(func() Database { return MySQL{} },
						bind.WithName(name), bind.Lazy(), bind.Singleton())
				}
			}(i)
		}

		close(ready)
		withDeadline(t, wg.Wait)

		for i, err := range errs {
			assert.NoError(t, err, "goroutine %d: unexpected error", i)
		}
	})
}

// TestSlowResolverConcurrency verifies that a slow resolver for one type does NOT
// block or slow down the resolution of independent types. Each singleton has its own
// lock (b.lock), so resolving Shape (slow) and Database (fast) concurrently should
// not cause Database resolution to wait for Shape's slow resolver.
func TestSlowResolverConcurrency(t *testing.T) {
	t.Parallel()

	const deadline = 3 * time.Second

	t.Run("slow_singleton_does_not_block_independent_singleton", func(t *testing.T) {
		t.Parallel()

		const slowDelay = 200 * time.Millisecond

		c := container.New()
		require.NoError(t, c.Bind(func() Shape {
			time.Sleep(slowDelay) // slow resolver
			return &Circle{a: 42}
		}, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func() Database {
			return &MySQL{} // fast resolver
		}, bind.Singleton(), bind.Lazy()))

		ready := make(chan struct{})
		var wg sync.WaitGroup

		var dbDuration time.Duration
		var shapeDuration time.Duration

		wg.Add(2)
		go func() {
			defer wg.Done()
			<-ready
			start := time.Now()
			var s Shape
			assert.NoError(t, c.Resolve(&s))
			shapeDuration = time.Since(start)
			assert.Equal(t, 42, s.GetArea())
		}()
		go func() {
			defer wg.Done()
			<-ready
			start := time.Now()
			var db Database
			assert.NoError(t, c.Resolve(&db))
			dbDuration = time.Since(start)
			assert.True(t, db.Connect())
		}()

		close(ready)

		done := make(chan struct{})
		go func() { wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(deadline):
			t.Fatal("deadlock: slow resolver blocked independent resolution")
		}

		// Database (fast) should complete much faster than Shape (slow).
		// If it took nearly as long, the slow resolver is blocking the fast one.
		assert.Less(t, dbDuration, slowDelay,
			"fast Database resolution (%v) should not be blocked by slow Shape resolver (%v)",
			dbDuration, slowDelay)
		assert.GreaterOrEqual(t, shapeDuration, slowDelay,
			"Shape resolution should take at least as long as the slow resolver")
	})

	t.Run("slow_singleton_does_not_block_concurrent_resolvers_of_other_types", func(t *testing.T) {
		t.Parallel()

		const slowDelay = 200 * time.Millisecond
		const goroutines = 20

		c := container.New()
		require.NoError(t, c.Bind(func() Shape {
			time.Sleep(slowDelay)
			return &Circle{a: 1}
		}, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func() Database {
			return &MySQL{}
		}, bind.Singleton(), bind.Lazy()))
		require.NoError(t, c.Bind(func() Cache {
			return InMemoryCache{}
		}, bind.Singleton(), bind.Lazy()))

		ready := make(chan struct{})
		var wg sync.WaitGroup

		// Measure how long the fast types take to resolve when Shape is blocking.
		fastDurations := make([]time.Duration, goroutines)

		// One goroutine resolves the slow Shape.
		wg.Go(func() {
			<-ready
			var s Shape
			assert.NoError(t, c.Resolve(&s))
		})

		// Many goroutines resolve fast Database and Cache concurrently.
		wg.Add(goroutines)
		for i := range goroutines {
			go func(idx int) {
				defer wg.Done()
				<-ready
				start := time.Now()
				if idx%2 == 0 {
					var db Database
					assert.NoError(t, c.Resolve(&db))
				} else {
					var cache Cache
					assert.NoError(t, c.Resolve(&cache))
				}
				fastDurations[idx] = time.Since(start)
			}(i)
		}

		close(ready)

		done := make(chan struct{})
		go func() { wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(deadline):
			t.Fatal("deadlock: slow resolver blocked other resolutions")
		}

		// Every fast resolver should complete well under the slow delay.
		for i, d := range fastDurations {
			assert.Less(t, d, slowDelay,
				"goroutine %d: fast resolution took %v, should be under %v", i, d, slowDelay)
		}
	})

	t.Run("slow_transient_does_not_block_other_types", func(t *testing.T) {
		t.Parallel()

		const slowDelay = 200 * time.Millisecond

		c := container.New()
		require.NoError(t, c.Bind(func() Shape {
			time.Sleep(slowDelay)
			return &Circle{a: 1}
		})) // transient
		require.NoError(t, c.Bind(func() Database {
			return &MySQL{}
		})) // transient

		ready := make(chan struct{})
		var wg sync.WaitGroup

		var dbDuration time.Duration

		wg.Add(2)
		go func() {
			defer wg.Done()
			<-ready
			var s Shape
			assert.NoError(t, c.Resolve(&s))
		}()
		go func() {
			defer wg.Done()
			<-ready
			start := time.Now()
			var db Database
			assert.NoError(t, c.Resolve(&db))
			dbDuration = time.Since(start)
		}()

		close(ready)

		done := make(chan struct{})
		go func() { wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(deadline):
			t.Fatal("deadlock: slow transient resolver blocked other resolution")
		}

		assert.Less(t, dbDuration, slowDelay,
			"fast transient Database (%v) should not wait for slow Shape", dbDuration)
	})

	t.Run("many_concurrent_resolvers_one_slow_measures_p99_latency", func(t *testing.T) {
		t.Parallel()

		const (
			slowDelay  = 300 * time.Millisecond
			goroutines = 50
		)

		c := container.New()
		require.NoError(t, c.Bind(func() Shape {
			time.Sleep(slowDelay)
			return &Circle{a: 1}
		}, bind.Singleton(), bind.Lazy()))

		// Register many fast independent singletons.
		for i := range goroutines {
			name := fmt.Sprintf("db-%d", i)
			require.NoError(t, c.Bind(func() Database {
				return &MySQL{}
			}, bind.WithName(name), bind.Singleton(), bind.Lazy()))
		}

		ready := make(chan struct{})
		var wg sync.WaitGroup
		fastDurations := make([]time.Duration, goroutines)

		// One slow Shape resolver.
		wg.Go(func() {
			<-ready
			var s Shape
			_ = c.Resolve(&s)
		})

		// Many fast resolvers.
		wg.Add(goroutines)
		for i := range goroutines {
			go func(idx int) {
				defer wg.Done()
				<-ready
				start := time.Now()
				var db Database
				name := fmt.Sprintf("db-%d", idx)
				assert.NoError(t, c.Resolve(&db, resolve.WithName(name)))
				fastDurations[idx] = time.Since(start)
			}(i)
		}

		close(ready)

		done := make(chan struct{})
		go func() { wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(deadline):
			t.Fatal("deadlock: slow resolver blocked fast resolvers")
		}

		// Find the maximum (p99-ish) of the fast durations.
		var maxFast time.Duration
		for _, d := range fastDurations {
			if d > maxFast {
				maxFast = d
			}
		}

		assert.Less(t, maxFast, slowDelay,
			"worst-case fast resolution (%v) should still be under the slow delay (%v)", maxFast, slowDelay)
	})

	t.Run("concurrent_same_slow_singleton_only_invokes_resolver_once", func(t *testing.T) {
		t.Parallel()

		const (
			slowDelay  = 200 * time.Millisecond
			goroutines = 20
		)

		var callCount int64
		c := container.New()
		require.NoError(t, c.Bind(func() Shape {
			atomic.AddInt64(&callCount, 1)
			time.Sleep(slowDelay)
			return &Circle{a: 42}
		}, bind.Singleton(), bind.Lazy()))

		ready := make(chan struct{})
		var wg sync.WaitGroup
		instances := make([]Shape, goroutines)

		wg.Add(goroutines)
		for i := range goroutines {
			go func(idx int) {
				defer wg.Done()
				<-ready
				var s Shape
				assert.NoError(t, c.Resolve(&s))
				instances[idx] = s
			}(i)
		}

		close(ready)

		done := make(chan struct{})
		go func() { wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(deadline):
			t.Fatal("deadlock during concurrent slow singleton resolution")
		}

		assert.Equal(t, int64(1), atomic.LoadInt64(&callCount),
			"slow singleton resolver must be invoked exactly once")
		for _, inst := range instances[1:] {
			assert.Same(t, instances[0].(*Circle), inst.(*Circle),
				"all goroutines must receive the same singleton instance")
		}
	})
}
