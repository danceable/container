// Package container is a lightweight yet powerful IoC container for Go projects.
// It provides an easy-to-use interface and performance-in-mind container to be your ultimate requirement.
package container

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/danceable/container/bind"
	"github.com/danceable/container/errors"
	"github.com/danceable/container/internal/registerar"
	"github.com/danceable/container/resolve"
)

// Reset deletes all the existing bindings and empties the container.
func (c *Container) Reset() {
	c.reg.Reset()
}

// Bind maps an abstraction to concrete and instantiates if it is a singleton binding.
func (c *Container) Bind(resolver any, opts ...bind.BindOption) error {
	options := bind.DefaultOptions()
	for _, o := range opts {
		o(options)
	}

	reflected, err := validateResolver(resolver)
	if err != nil {
		return err
	}

	binding := registerar.NewBinding(
		options.Name,
		options.Singleton,
		options.DependenciesByParams,
		options.DependenciesByNamedBindings,
		resolver,
		nil,
	)

	abstraction := reflected.Out(0)
	if options.Singleton {
		return c.bindSingleton(abstraction, options.Name, options.Lazy, binding)
	}

	return c.bindTransient(abstraction, options.Name, options.Lazy, binding)
}

// bindSingleton registers the binding before building it, so that concurrent Binds of the
// same abstraction all settle on the instance the winner builds.
func (c *Container) bindSingleton(abstraction reflect.Type, name string, lazy bool, binding *registerar.Binding) error {
	wasNew, err := c.reg.SetIfAbsent(abstraction, name, binding)
	if err != nil || !wasNew || lazy {
		return err
	}

	// Through the binding, so that a Resolve arriving before the concrete is stored waits
	// for this instance instead of building a second one.
	if _, err := binding.GetOrSetConcrete(c.invoke, nil); err != nil {
		c.reg.Delete(abstraction, name)

		return err
	}

	return nil
}

// bindTransient builds the concrete before registering the binding, nothing being able to
// resolve an instance no one shares.
func (c *Container) bindTransient(abstraction reflect.Type, name string, lazy bool, binding *registerar.Binding) error {
	if !lazy {
		if _, err := c.invoke(binding, nil); err != nil {
			return err
		}
	}

	return c.reg.Set(abstraction, name, binding)
}

// Resolve takes an abstraction (reference of an interface type) and fills it with the related concrete.
func (c *Container) Resolve(abstraction any, opts ...resolve.ResolveOption) error {
	options := resolve.DefaultOptions()
	for _, o := range opts {
		o(options)
	}

	receiverType := reflect.TypeOf(abstraction)
	if receiverType == nil {
		return errors.ErrInvalidAbstraction
	}

	if receiverType.Kind() != reflect.Pointer {
		return errors.ErrInvalidAbstraction
	}

	if reflect.ValueOf(abstraction).IsNil() {
		return errors.ErrInvalidAbstraction
	}

	elem := receiverType.Elem()

	if binding, owner, exist := c.lookup(elem, options.Name); exist {
		instance, err := owner.make(binding, options.Params)
		if err == nil {
			reflect.ValueOf(abstraction).Elem().Set(reflect.ValueOf(instance))
			return nil
		}

		return fmt.Errorf("%w for: %s. Error encountered: %w", errors.ErrEncounteredError, elem.String(), err)
	}

	return fmt.Errorf("%w; the abstraction is: %s", errors.ErrNoConcreteFound, elem.String())
}

// Call takes a receiver function with one or more arguments of the abstractions (interfaces).
// It invokes the receiver function and passes the related concretes.
func (c *Container) Call(function any, opts ...resolve.ResolveOption) error {
	receiverType := reflect.TypeOf(function)
	if receiverType == nil || receiverType.Kind() != reflect.Func {
		return errors.ErrInvalidFunction
	}

	options := resolve.DefaultOptions()
	for _, o := range opts {
		o(options)
	}

	a := arguments{
		scope:         c,
		resolveParams: options.Params,
		usedResolve:   make([]bool, len(options.Params)),
		name:          options.Name,
	}

	values, err := a.of(function)
	if err != nil {
		return err
	}

	result := reflect.ValueOf(function).Call(values)

	if len(result) == 0 {
		return nil
	} else if len(result) == 1 && result[0].CanInterface() {
		if isNil(result[0]) {
			return nil
		}
		if err, ok := result[0].Interface().(error); ok {
			return err
		}
	}

	return errors.ErrInvalidFunctionSignature
}

// isNil reports whether the value is nil, answering false for the kinds reflect panics on.
func isNil(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return value.IsNil()
	default:
		return false
	}
}

// Fill takes a struct and resolves the fields with the tag `container:"inject"`
func (c *Container) Fill(structure any, opts ...resolve.ResolveOption) error {
	receiverType := reflect.TypeOf(structure)
	if receiverType == nil {
		return errors.ErrInvalidStructure
	}

	if receiverType.Kind() != reflect.Pointer {
		return errors.ErrInvalidStructure
	}

	elem := receiverType.Elem()
	if elem.Kind() != reflect.Struct {
		return errors.ErrInvalidStructure
	}

	if reflect.ValueOf(structure).IsNil() {
		return errors.ErrInvalidStructure
	}

	s := reflect.ValueOf(structure).Elem()

	options := resolve.DefaultOptions()
	for _, o := range opts {
		o(options)
	}

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)

		if t, exist := s.Type().Field(i).Tag.Lookup("container"); exist {
			var name string

			switch t {
			case "type":
				name = options.Name
			case "name":
				name = s.Type().Field(i).Name
			default:
				return fmt.Errorf("%w; the field is: %s", errors.ErrInvalidStructTag, s.Type().Field(i).Name)
			}

			if binding, owner, exist := c.lookup(f.Type(), name); exist {
				instance, err := owner.make(binding, options.Params)
				if err != nil {
					return err
				}

				ptr := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
				ptr.Set(reflect.ValueOf(instance))

				continue
			}

			return fmt.Errorf("%w; the field is: %s", errors.ErrCannotMakeField, s.Type().Field(i).Name)
		}
	}

	return nil
}

// validateResolver checks that the resolver is a function the container can build a
// concrete with, and returns its type.
func validateResolver(resolver any) (reflect.Type, error) {
	reflected := reflect.TypeOf(resolver)
	if reflected == nil || reflected.Kind() != reflect.Func || reflect.ValueOf(resolver).IsNil() {
		return nil, errors.ErrNonFunctionResolver
	}

	returns := reflected.NumOut()
	if returns == 0 || returns > 2 {
		return nil, errors.ErrInvalidResolver
	}

	if returns == 2 && reflected.Out(1) != reflect.TypeFor[error]() {
		return nil, errors.ErrInvalidResolver
	}

	abstraction := reflected.Out(0)
	for in := range reflected.Ins() {
		if in == abstraction {
			return nil, errors.ErrResolverDependsOnAbstract
		}
	}

	return reflected, nil
}
