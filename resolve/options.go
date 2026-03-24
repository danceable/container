package resolve

import (
	"reflect"
)

// resolveOptions holds runtime parameters for resolve-time injection.
type resolveOptions struct {
	Name   string
	Params []reflect.Value
}

// DefaultOptions returns a new instance of resolveOptions with default values.
func DefaultOptions() *resolveOptions {
	return &resolveOptions{}
}

// ResolveOption is a functional option for resolve-time parameters.
type ResolveOption func(*resolveOptions)

// WithName specifies the name of the binding to resolve.
func WithName(name string) ResolveOption {
	return func(o *resolveOptions) {
		o.Name = name
	}
}

// WithParams provides runtime values used to satisfy resolver arguments.
func WithParams(params ...any) ResolveOption {
	return func(o *resolveOptions) {
		for _, param := range params {
			o.Params = append(o.Params, reflect.ValueOf(param))
		}
	}
}
