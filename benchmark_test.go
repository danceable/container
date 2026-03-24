package container_test

import (
	"fmt"
	"testing"

	"github.com/danceable/container"
	"github.com/danceable/container/bind"
	"github.com/danceable/container/resolve"
)

func BenchmarkBind(b *testing.B) {
	b.Run("transient", func(b *testing.B) {
		for b.Loop() {
			c := container.New()
			_ = c.Bind(func() Shape { return &Circle{a: 1} })
		}
	})

	b.Run("singleton_eager", func(b *testing.B) {
		for b.Loop() {
			c := container.New()
			_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton())
		}
	})

	b.Run("singleton_lazy", func(b *testing.B) {
		for b.Loop() {
			c := container.New()
			_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton(), bind.Lazy())
		}
	})

	b.Run("named_singleton_lazy", func(b *testing.B) {
		for b.Loop() {
			c := container.New()
			_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.WithName("named"), bind.Singleton(), bind.Lazy())
		}
	})

	b.Run("with_dependency", func(b *testing.B) {
		for b.Loop() {
			c := container.New()
			_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton())
			_ = c.Bind(func(s Shape) Database { return &MySQL{} }, bind.Singleton())
		}
	})

	b.Run("many_bindings", func(b *testing.B) {
		for b.Loop() {
			c := container.New()
			for j := range 100 {
				_ = c.Bind(func() Shape { return &Circle{a: j} },
					bind.WithName(fmt.Sprintf("shape-%d", j)), bind.Singleton(), bind.Lazy())
			}
		}
	})
}

func BenchmarkResolve(b *testing.B) {
	b.Run("singleton", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton())
		b.ResetTimer()
		for b.Loop() {
			var s Shape
			_ = c.Resolve(&s)
		}
	})

	b.Run("singleton_lazy", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton(), bind.Lazy())
		var warm Shape
		_ = c.Resolve(&warm)
		b.ResetTimer()
		for b.Loop() {
			var s Shape
			_ = c.Resolve(&s)
		}
	})

	b.Run("transient", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} })
		b.ResetTimer()
		for b.Loop() {
			var s Shape
			_ = c.Resolve(&s)
		}
	})

	b.Run("named_singleton", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.WithName("named"), bind.Singleton())
		b.ResetTimer()
		for b.Loop() {
			var s Shape
			_ = c.Resolve(&s, resolve.WithName("named"))
		}
	})

	b.Run("with_dependency_chain", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton())
		_ = c.Bind(func(s Shape) Database { return &MySQL{} }, bind.Singleton())
		b.ResetTimer()
		for b.Loop() {
			var db Database
			_ = c.Resolve(&db)
		}
	})

	b.Run("with_runtime_params", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: true}
		}, bind.Singleton(), bind.Lazy())
		b.ResetTimer()
		for b.Loop() {
			var db Database
			_ = c.Resolve(&db, resolve.WithParams(10, &Circle{a: 2}))
		}
	})

	b.Run("interface_implementation_lookup", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() *Circle { return &Circle{a: 1} }, bind.Singleton())
		b.ResetTimer()
		for b.Loop() {
			var s Shape
			_ = c.Resolve(&s)
		}
	})

	b.Run("from_100_bindings", func(b *testing.B) {
		c := container.New()
		for j := range 100 {
			_ = c.Bind(func() Shape { return &Circle{a: j} },
				bind.WithName(fmt.Sprintf("shape-%d", j)), bind.Singleton(), bind.Lazy())
		}
		b.ResetTimer()
		for b.Loop() {
			var s Shape
			_ = c.Resolve(&s, resolve.WithName("shape-50"))
		}
	})
}

func BenchmarkCall(b *testing.B) {
	b.Run("no_args", func(b *testing.B) {
		c := container.New()
		b.ResetTimer()
		for b.Loop() {
			_ = c.Call(func() {})
		}
	})

	b.Run("one_dependency", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton())
		b.ResetTimer()
		for b.Loop() {
			_ = c.Call(func(s Shape) {})
		}
	})

	b.Run("two_dependencies", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton())
		_ = c.Bind(func() Database { return &MySQL{} }, bind.Singleton())
		b.ResetTimer()
		for b.Loop() {
			_ = c.Call(func(s Shape, db Database) {})
		}
	})

	b.Run("with_error_return", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton())
		b.ResetTimer()
		for b.Loop() {
			_ = c.Call(func(s Shape) error { return nil })
		}
	})
}

func BenchmarkFill(b *testing.B) {
	b.Run("one_field", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton())
		b.ResetTimer()
		for b.Loop() {
			app := struct {
				S Shape `container:"type"`
			}{}
			_ = c.Fill(&app)
		}
	})

	b.Run("two_fields", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton())
		_ = c.Bind(func() Database { return &MySQL{} }, bind.Singleton())
		b.ResetTimer()
		for b.Loop() {
			app := struct {
				S Shape    `container:"type"`
				D Database `container:"type"`
			}{}
			_ = c.Fill(&app)
		}
	})

	b.Run("named_field", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.WithName("S"), bind.Singleton())
		b.ResetTimer()
		for b.Loop() {
			app := struct {
				S Shape `container:"name"`
			}{}
			_ = c.Fill(&app)
		}
	})

	b.Run("with_runtime_params", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: true}
		}, bind.Singleton(), bind.Lazy())
		b.ResetTimer()
		for b.Loop() {
			app := struct {
				D Database `container:"type"`
			}{}
			_ = c.Fill(&app, resolve.WithParams(10, &Circle{a: 2}))
		}
	})
}

func BenchmarkResolve_Concurrent(b *testing.B) {
	b.Run("singleton_parallel", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton())
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var s Shape
				_ = c.Resolve(&s)
			}
		})
	})

	b.Run("transient_parallel", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} })
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var s Shape
				_ = c.Resolve(&s)
			}
		})
	})

	b.Run("mixed_types_parallel", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton())
		_ = c.Bind(func() Database { return &MySQL{} }, bind.Singleton())
		_ = c.Bind(func() Cache { return InMemoryCache{} }, bind.Singleton())
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				switch i % 3 {
				case 0:
					var s Shape
					_ = c.Resolve(&s)
				case 1:
					var db Database
					_ = c.Resolve(&db)
				case 2:
					var cache Cache
					_ = c.Resolve(&cache)
				}
				i++
			}
		})
	})

	b.Run("named_parallel", func(b *testing.B) {
		c := container.New()
		for i := range 10 {
			name := fmt.Sprintf("shape-%d", i)
			_ = c.Bind(func() Shape { return &Circle{a: i} },
				bind.WithName(name), bind.Singleton())
		}
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				var s Shape
				name := fmt.Sprintf("shape-%d", i%10)
				_ = c.Resolve(&s, resolve.WithName(name))
				i++
			}
		})
	})
}

func BenchmarkCall_Concurrent(b *testing.B) {
	b.Run("one_dependency_parallel", func(b *testing.B) {
		c := container.New()
		_ = c.Bind(func() Shape { return &Circle{a: 1} }, bind.Singleton())
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = c.Call(func(s Shape) {})
			}
		})
	})
}

func BenchmarkReset(b *testing.B) {
	b.Run("empty_container", func(b *testing.B) {
		c := container.New()
		b.ResetTimer()
		for b.Loop() {
			c.Reset()
		}
	})

	b.Run("with_50_bindings", func(b *testing.B) {
		for b.Loop() {
			b.StopTimer()
			c := container.New()
			for j := range 50 {
				_ = c.Bind(func() Shape { return &Circle{a: j} },
					bind.WithName(fmt.Sprintf("s-%d", j)), bind.Singleton(), bind.Lazy())
			}
			b.StartTimer()
			c.Reset()
		}
	})
}
