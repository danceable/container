package bind

import "reflect"

// options holds the configuration options for a binding.
type options struct {
	Name                        string
	Singleton                   bool
	Lazy                        bool
	DependenciesByParams        []reflect.Value
	DependenciesByNamedBindings []string
}

func DefaultOptions() *options {
	return &options{}
}

// BindOption is a functional option for configuring a binding.
type BindOption func(*options)

// WithName sets a name for the binding, enabling multiple concretes per abstraction.
func WithName(name string) BindOption {
	return func(o *options) {
		o.Name = name
	}
}

// Singleton marks the binding as a singleton (one shared instance).
func Singleton() BindOption {
	return func(o *options) {
		o.Singleton = true
	}
}

// Lazy defers the resolver invocation until the first time the binding is resolved.
func Lazy() BindOption {
	return func(o *options) {
		o.Lazy = true
	}
}

// ResolveDepenenciesByParams specifies that the binding depenencies will be resolved by passed parameters.
func ResolveDepenenciesByParams(params ...any) BindOption {
	return func(o *options) {
		for _, param := range params {
			o.DependenciesByParams = append(o.DependenciesByParams, reflect.ValueOf(param))
		}
	}
}

// ResolveDependenciesByNamedBindings specifies that the binding dependencies will be resolved by named bindings.
func ResolveDependenciesByNamedBindings(bindings ...string) BindOption {
	return func(o *options) {
		for _, binding := range bindings {
			o.DependenciesByNamedBindings = append(o.DependenciesByNamedBindings, binding)
		}
	}
}
