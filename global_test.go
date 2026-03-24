package container_test

import (
	"testing"

	"github.com/danceable/container"
	"github.com/danceable/container/bind"
	"github.com/danceable/container/resolve"
	"github.com/stretchr/testify/assert"
)

func TestBind(t *testing.T) {
	t.Run("singleton", func(t *testing.T) {
		container.Reset()
		err := container.Bind(func() Shape { return &Circle{a: 13} }, bind.Singleton())
		assert.NoError(t, err)
		var s Shape
		assert.NoError(t, container.Resolve(&s))
		assert.Equal(t, 13, s.GetArea())
	})

	t.Run("singleton lazy", func(t *testing.T) {
		container.Reset()
		err := container.Bind(func() Shape { return &Circle{a: 13} }, bind.Singleton(), bind.Lazy())
		assert.NoError(t, err)
		var s Shape
		assert.NoError(t, container.Resolve(&s))
		assert.Equal(t, 13, s.GetArea())
	})

	t.Run("named singleton", func(t *testing.T) {
		container.Reset()
		err := container.Bind(func() Shape { return &Circle{a: 13} }, bind.WithName("rounded"), bind.Singleton())
		assert.NoError(t, err)
		var s Shape
		assert.NoError(t, container.Resolve(&s, resolve.WithName("rounded")))
		assert.Equal(t, 13, s.GetArea())
	})

	t.Run("named singleton lazy", func(t *testing.T) {
		container.Reset()
		err := container.Bind(func() Shape { return &Circle{a: 13} }, bind.WithName("rounded"), bind.Singleton(), bind.Lazy())
		assert.NoError(t, err)
		var s Shape
		assert.NoError(t, container.Resolve(&s, resolve.WithName("rounded")))
		assert.Equal(t, 13, s.GetArea())
	})

	t.Run("transient", func(t *testing.T) {
		container.Reset()
		err := container.Bind(func() Shape { return &Circle{a: 13} })
		assert.NoError(t, err)
		var s Shape
		assert.NoError(t, container.Resolve(&s))
		assert.Equal(t, 13, s.GetArea())
	})

	t.Run("transient lazy", func(t *testing.T) {
		container.Reset()
		err := container.Bind(func() Shape { return &Circle{a: 13} }, bind.Lazy())
		assert.NoError(t, err)
		var s Shape
		assert.NoError(t, container.Resolve(&s))
		assert.Equal(t, 13, s.GetArea())
	})

	t.Run("named transient", func(t *testing.T) {
		container.Reset()
		err := container.Bind(func() Shape { return &Circle{a: 13} }, bind.WithName("rounded"))
		assert.NoError(t, err)
		var s Shape
		assert.NoError(t, container.Resolve(&s, resolve.WithName("rounded")))
		assert.Equal(t, 13, s.GetArea())
	})

	t.Run("named transient lazy", func(t *testing.T) {
		container.Reset()
		err := container.Bind(func() Shape { return &Circle{a: 13} }, bind.WithName("rounded"), bind.Lazy())
		assert.NoError(t, err)
		var s Shape
		assert.NoError(t, container.Resolve(&s, resolve.WithName("rounded")))
		assert.Equal(t, 13, s.GetArea())
	})
}

func TestCall(t *testing.T) {
	t.Run("no dependencies", func(t *testing.T) {
		container.Reset()

		err := container.Call(func() {})
		assert.NoError(t, err)
	})

	t.Run("with resolved dependencies", func(t *testing.T) {
		container.Reset()

		err := container.Bind(func() Shape {
			return &Circle{a: 5}
		}, bind.Singleton())
		assert.NoError(t, err)

		err = container.Call(func(s Shape) {
			assert.Equal(t, 5, s.GetArea())
		})
		assert.NoError(t, err)
	})

	t.Run("with named dependency", func(t *testing.T) {
		container.Reset()

		err := container.Bind(func() Shape {
			return &Circle{a: 1}
		}, bind.Singleton())
		assert.NoError(t, err)

		err = container.Bind(func() Shape {
			return &Circle{a: 2}
		}, bind.WithName("named"), bind.Singleton())
		assert.NoError(t, err)

		err = container.Call(func(s Shape) {
			assert.Equal(t, 2, s.GetArea())
		}, resolve.WithName("named"))
		assert.NoError(t, err)
	})

	t.Run("with missing dependency", func(t *testing.T) {
		container.Reset()

		err := container.Call(func(s Shape) {})
		assert.EqualError(t, err, "container: no concrete found for the given abstraction; the abstraction is: container_test.Shape")
	})

	t.Run("with returning error", func(t *testing.T) {
		container.Reset()

		err := container.Bind(func() Shape {
			return &Circle{}
		}, bind.Singleton())
		assert.NoError(t, err)

		err = container.Call(func(s Shape) error {
			return assert.AnError
		})
		assert.EqualError(t, err, assert.AnError.Error())
	})

	t.Run("with returning nil error", func(t *testing.T) {
		container.Reset()

		err := container.Bind(func() Shape {
			return &Circle{}
		}, bind.Singleton())
		assert.NoError(t, err)

		err = container.Call(func(s Shape) error {
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("with invalid receiver", func(t *testing.T) {
		container.Reset()

		err := container.Call("not a function")
		assert.EqualError(t, err, "container: invalid function")
	})

	t.Run("with invalid function signature", func(t *testing.T) {
		container.Reset()

		err := container.Bind(func() Shape {
			return &Circle{}
		}, bind.Singleton())
		assert.NoError(t, err)

		err = container.Call(func(s Shape) (int, error) {
			return 0, nil
		})
		assert.EqualError(t, err, "container: receiver function signature is invalid")
	})
}

func TestResolve(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		container.Reset()

		var s Shape

		err := container.Bind(func() Shape {
			return &Circle{a: 13}
		}, bind.Singleton())
		assert.NoError(t, err)

		err = container.Resolve(&s)
		assert.NoError(t, err)
	})

	t.Run("named", func(t *testing.T) {
		container.Reset()

		var s Shape

		err := container.Bind(func() Shape {
			return &Circle{a: 13}
		}, bind.WithName("rounded"), bind.Singleton())
		assert.NoError(t, err)

		err = container.Resolve(&s, resolve.WithName("rounded"))
		assert.NoError(t, err)
	})

	t.Run("with runtime params", func(t *testing.T) {
		container.Reset()

		err := container.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.Singleton(), bind.Lazy())
		assert.NoError(t, err)

		var db Database
		err = container.Resolve(&db, resolve.WithParams(10, &Circle{a: 2}))
		assert.NoError(t, err)
		assert.True(t, db.Connect())
	})

	t.Run("named with runtime params", func(t *testing.T) {
		container.Reset()

		err := container.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.WithName("rounded"), bind.Lazy())
		assert.NoError(t, err)

		var db Database
		err = container.Resolve(&db, resolve.WithName("rounded"), resolve.WithParams(10, &Circle{a: 2}))
		assert.NoError(t, err)
		assert.True(t, db.Connect())
	})

	t.Run("with runtime params and container fallback", func(t *testing.T) {
		container.Reset()

		err := container.Bind(func() Shape {
			return &Circle{a: 2}
		}, bind.Singleton())
		assert.NoError(t, err)

		err = container.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.Singleton(), bind.Lazy())
		assert.NoError(t, err)

		var db Database
		err = container.Resolve(&db, resolve.WithParams(10))
		assert.NoError(t, err)
		assert.True(t, db.Connect())
	})

	t.Run("runtime params take precedence over container", func(t *testing.T) {
		container.Reset()

		err := container.Bind(func() Shape {
			return &Circle{a: 99}
		}, bind.Singleton())
		assert.NoError(t, err)

		err = container.Bind(func(s Shape) Database {
			return PostgreSQL{ready: s.GetArea() == 2}
		})
		assert.NoError(t, err)

		var db Database
		err = container.Resolve(&db, resolve.WithParams(&Circle{a: 2}))
		assert.NoError(t, err)
		assert.True(t, db.Connect())
	})

	t.Run("missing runtime params with no fallback", func(t *testing.T) {
		container.Reset()

		err := container.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.Singleton(), bind.Lazy())
		assert.NoError(t, err)

		var db Database
		err = container.Resolve(&db, resolve.WithParams(10))
		assert.EqualError(t, err, "container: encountered error while making concrete for: container_test.Database. Error encountered: container: no concrete found for the given abstraction; the abstraction is: container_test.Shape")
	})
}

func TestFill(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		container.Reset()

		err := container.Bind(func() Shape {
			return &Circle{a: 13}
		}, bind.Singleton())
		assert.NoError(t, err)

		myApp := struct {
			S Shape `container:"type"`
		}{}

		err = container.Fill(&myApp)
		assert.NoError(t, err)
		assert.IsType(t, &Circle{}, myApp.S)
	})

	t.Run("with runtime params", func(t *testing.T) {
		container.Reset()

		err := container.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.Singleton(), bind.Lazy())
		assert.NoError(t, err)

		myApp := struct {
			D Database `container:"type"`
		}{}

		err = container.Fill(&myApp, resolve.WithParams(10, &Circle{a: 2}))
		assert.NoError(t, err)
		assert.True(t, myApp.D.Connect())
	})

	t.Run("with runtime params and container fallback", func(t *testing.T) {
		container.Reset()

		err := container.Bind(func() Shape {
			return &Circle{a: 2}
		}, bind.Singleton())
		assert.NoError(t, err)

		err = container.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.Singleton(), bind.Lazy())
		assert.NoError(t, err)

		myApp := struct {
			D Database `container:"type"`
		}{}

		err = container.Fill(&myApp, resolve.WithParams(10))
		assert.NoError(t, err)
		assert.True(t, myApp.D.Connect())
	})

	t.Run("missing runtime params with no fallback", func(t *testing.T) {
		container.Reset()

		err := container.Bind(func(x int, s Shape) Database {
			return PostgreSQL{ready: x+s.GetArea() == 12}
		}, bind.Singleton(), bind.Lazy())
		assert.NoError(t, err)

		myApp := struct {
			D Database `container:"type"`
		}{}

		err = container.Fill(&myApp, resolve.WithParams(10))
		assert.EqualError(t, err, "container: no concrete found for the given abstraction; the abstraction is: container_test.Shape")
	})
}
