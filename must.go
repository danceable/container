package container

import (
	"github.com/danceable/container/bind"
	"github.com/danceable/container/resolve"
)

// MustBind wraps the `Bind` method and panics on errors instead of returning the errors.
func MustBind(c *Container, resolver any, opts ...bind.BindOption) {
	if err := c.Bind(resolver, opts...); err != nil {
		panic(err)
	}
}

// MustCall wraps the `Call` method and panics on errors instead of returning the errors.
func MustCall(c *Container, function any, opts ...resolve.ResolveOption) {
	if err := c.Call(function, opts...); err != nil {
		panic(err)
	}
}

// MustResolve wraps the `Resolve` method and panics on errors instead of returning the errors.
func MustResolve(c *Container, abstraction any, opts ...resolve.ResolveOption) {
	if err := c.Resolve(abstraction, opts...); err != nil {
		panic(err)
	}
}

// MustFill wraps the `Fill` method and panics on errors instead of returning the errors.
func MustFill(c *Container, structure any, opts ...resolve.ResolveOption) {
	if err := c.Fill(structure, opts...); err != nil {
		panic(err)
	}
}
