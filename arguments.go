package container

import (
	"fmt"
	"reflect"

	"github.com/danceable/container/errors"
	"github.com/danceable/container/internal/registerar"
)

// arguments resolves the arguments a function is called with, one source at a time until
// one of them answers. The order they are tried in is the precedence the container
// promises, and stating it once here is what keeps it a promise.
type arguments struct {
	scope *Container // the scope the dependencies are resolved from

	resolveParams []reflect.Value // params passed to Resolve or Call
	usedResolve   []bool

	bindParams []reflect.Value // params given at bind time
	usedBind   []bool

	namedBindings []string // named bindings given at bind time
	name          string   // the name of the binding being built
}

// of resolves every input of the given function.
func (a *arguments) of(function any) ([]reflect.Value, error) {
	reflected := reflect.TypeOf(function)
	values := make([]reflect.Value, reflected.NumIn())

	for i := range values {
		value, err := a.resolve(reflected.In(i))
		if err != nil {
			return nil, err
		}

		values[i] = value
	}

	return values, nil
}

func (a *arguments) resolve(abstraction reflect.Type) (reflect.Value, error) {
	if value, ok := takeParam(abstraction, a.resolveParams, a.usedResolve); ok {
		return value, nil
	}

	if value, ok := takeParam(abstraction, a.bindParams, a.usedBind); ok {
		return value, nil
	}

	if value, found, err := a.fromNamedBindings(abstraction); found {
		return value, err
	}

	if value, found, err := a.fromScope(abstraction, a.name); found {
		return value, err
	}

	if len(a.namedBindings) > 0 {
		return reflect.Value{}, fmt.Errorf("%w; named binding(s) %v specified at bind time could not resolve dependency: %s", errors.ErrNoConcreteFound, a.namedBindings, abstraction.String())
	}

	return reflect.Value{}, fmt.Errorf("%w; the abstraction is: %s", errors.ErrNoConcreteFound, abstraction.String())
}

func (a *arguments) fromNamedBindings(abstraction reflect.Type) (reflect.Value, bool, error) {
	for _, name := range a.namedBindings {
		if value, found, err := a.fromScope(abstraction, name); found {
			return value, true, err
		}
	}

	return reflect.Value{}, false, nil
}

func (a *arguments) fromScope(abstraction reflect.Type, name string) (reflect.Value, bool, error) {
	binding, owner, exist := a.scope.find(abstraction, name)
	if !exist {
		return reflect.Value{}, false, nil
	}

	instance, err := owner.make(binding, a.resolveParams)
	if err != nil {
		return reflect.Value{}, true, err
	}

	return reflect.ValueOf(instance), true, nil
}

// takeParam returns the first unused param assignable to the abstraction, marking it as
// taken so no param satisfies two arguments.
func takeParam(abstraction reflect.Type, params []reflect.Value, used []bool) (reflect.Value, bool) {
	for i, param := range params {
		if used[i] {
			continue
		}

		if param.Type().AssignableTo(abstraction) {
			used[i] = true
			if param.Type() == abstraction {
				return param, true
			}

			return param.Convert(abstraction), true
		}
	}

	return reflect.Value{}, false
}

// make returns the concrete of the binding, building it at most once when it is a
// singleton.
func (c *Container) make(binding *registerar.Binding, resolveParams []reflect.Value) (any, error) {
	if binding.IsSingleton() {
		return binding.GetOrSetConcrete(c.invoke, resolveParams)
	}

	return c.invoke(binding, resolveParams)
}

// invoke calls the resolver of the binding with its arguments resolved.
func (c *Container) invoke(binding *registerar.Binding, resolveParams []reflect.Value) (any, error) {
	a := arguments{
		scope:         c,
		resolveParams: resolveParams,
		usedResolve:   make([]bool, len(resolveParams)),
		bindParams:    binding.BindParams(),
		usedBind:      make([]bool, len(binding.BindParams())),
		namedBindings: binding.NamedBindings(),
		name:          binding.GetName(),
	}

	values, err := a.of(binding.Resolver())
	if err != nil {
		return nil, err
	}

	results := reflect.ValueOf(binding.Resolver()).Call(values)
	if len(results) == 2 && results[1].CanInterface() {
		if err, ok := results[1].Interface().(error); ok {
			return results[0].Interface(), err
		}
	}

	return results[0].Interface(), nil
}
