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

// Container holds the registrar and provides methods to interact with bindings.
// It is the entry point in the package.
type Container struct {
	reg *registerar.Registrar
}

// New creates a new concrete of the Container.
func New() *Container {
	return &Container{reg: registerar.NewRegisterar()}
}

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

	reflectedResolver := reflect.TypeOf(resolver)
	if reflectedResolver.Kind() != reflect.Func {
		return errors.ErrNonFunctionResolver
	}

	if err := c.validateResolverFunction(reflectedResolver); err != nil {
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

	bindedType := reflectedResolver.Out(0)
	if options.Singleton {
		// Register first so concurrent Bind calls for the same type
		// don't all invoke the resolver (only the winner does).
		wasNew, err := c.reg.SetIfAbsent(
			bindedType,
			options.Name,
			binding,
		)
		if err != nil {
			return err
		}
		if !wasNew {
			return nil
		}

		if !options.Lazy {
			concrete, err := c.invoke(binding, nil)
			if err != nil {
				c.reg.Delete(bindedType, options.Name)
				return err
			}
			binding.SetConcrete(concrete)
		}

		return nil
	}

	if !options.Lazy {
		if _, err := c.invoke(binding, nil); err != nil {
			return err
		}
	}

	return c.reg.Set(
		bindedType,
		options.Name,
		binding,
	)
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

	elem := receiverType.Elem()

	if binding, exist := c.reg.Get(elem, options.Name); exist {
		instance, err := c.make(binding, options.Params)
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

	arguments, err := c.arguments(function, options.Params, nil, nil, options.Name)
	if err != nil {
		return err
	}

	result := reflect.ValueOf(function).Call(arguments)

	if len(result) == 0 {
		return nil
	} else if len(result) == 1 && result[0].CanInterface() {
		if result[0].IsNil() {
			return nil
		}
		if err, ok := result[0].Interface().(error); ok {
			return err
		}
	}

	return errors.ErrInvalidFunctionSignature
}

// Fill takes a struct and resolves the fields with the tag `container:"inject"`
func (c *Container) Fill(structure any, opts ...resolve.ResolveOption) error {
	receiverType := reflect.TypeOf(structure)
	if receiverType == nil {
		return errors.ErrInvalidStructure
	}

	if receiverType.Kind() == reflect.Pointer {
		elem := receiverType.Elem()
		if elem.Kind() == reflect.Struct {
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

					if binding, exist := c.reg.Get(f.Type(), name); exist {
						instance, err := c.make(binding, options.Params)
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
	}

	return errors.ErrInvalidStructure
}

// validateResolverFunction checks if the resolver function signature is valid.
func (c *Container) validateResolverFunction(funcType reflect.Type) error {
	retCount := funcType.NumOut()

	if retCount == 0 || retCount > 2 {
		return errors.ErrInvalidResolver
	}

	if retCount == 2 {
		if funcType.Out(1) != reflect.TypeFor[error]() {
			return errors.ErrInvalidResolver
		}
	}

	resolveType := funcType.Out(0)
	for in := range funcType.Ins() {
		if in == resolveType {
			return errors.ErrResolverDependsOnAbstract
		}
	}

	return nil
}

// make resolves the dependencies of the binding and returns the concrete instance.
func (c *Container) make(binding *registerar.Binding, resolveParams []reflect.Value) (any, error) {
	if binding.IsSingleton() {
		return binding.GetOrSetConcrete(c.invoke, resolveParams)
	}

	return c.invoke(binding, resolveParams)
}

// invoke calls the provided function with the given parameters and returns the result or an error if it occurs.
func (c *Container) invoke(binding *registerar.Binding, resolveParams []reflect.Value) (any, error) {
	arguments, err := c.arguments(binding.Resolver(), resolveParams, binding.BindParams(), binding.NamedBindings(), binding.GetName())
	if err != nil {
		return nil, err
	}

	values := reflect.ValueOf(binding.Resolver()).Call(arguments)
	if len(values) == 2 && values[1].CanInterface() {
		if err, ok := values[1].Interface().(error); ok {
			return values[0].Interface(), err
		}
	}

	return values[0].Interface(), nil
}

// arguments returns the list of resolved arguments for a function.
// Resolution order per argument:
//  1. Resolve-time params (passed at Resolve/Call time)
//  2. Bind-time params (specified via bind.ResolveDepenenciesByParams)
//  3. Bind-time named bindings (specified via bind.ResolveDependenciesByNamedBindings)
//  4. Container fallback (standard type+name lookup)
func (c *Container) arguments(function any, resolveParams, bindParams []reflect.Value, namedBindings []string, name string) ([]reflect.Value, error) {
	reflectedFunction := reflect.TypeOf(function)
	argumentsCount := reflectedFunction.NumIn()
	arguments := make([]reflect.Value, argumentsCount)
	usedResolveParams := make([]bool, len(resolveParams))
	usedBindParams := make([]bool, len(bindParams))

	for i := range argumentsCount {
		abstraction := reflectedFunction.In(i)

		// 1. Resolve-time params take highest priority.
		if value, ok := takeParam(abstraction, resolveParams, usedResolveParams); ok {
			arguments[i] = value
			continue
		}

		// 2a. Bind-time params.
		if value, ok := takeParam(abstraction, bindParams, usedBindParams); ok {
			arguments[i] = value
			continue
		}

		// 2b. Bind-time named bindings.
		resolved := false
		for _, nbName := range namedBindings {
			if binding, exist := c.reg.Find(abstraction, nbName); exist {
				instance, err := c.make(binding, resolveParams)
				if err != nil {
					return nil, err
				}
				arguments[i] = reflect.ValueOf(instance)
				resolved = true
				break
			}
		}
		if resolved {
			continue
		}

		// 3. Container fallback.
		if binding, exist := c.reg.Find(abstraction, name); exist {
			instance, err := c.make(binding, resolveParams)
			if err != nil {
				return nil, err
			}

			arguments[i] = reflect.ValueOf(instance)
		} else if len(namedBindings) > 0 {
			// When named bindings were specified but could not resolve this dependency,
			// return an error that mentions them to aid debugging.
			return nil, fmt.Errorf("%w; named binding(s) %v specified at bind time could not resolve dependency: %s", errors.ErrNoConcreteFound, namedBindings, abstraction.String())
		} else {
			return nil, fmt.Errorf("%w; the abstraction is: %s", errors.ErrNoConcreteFound, abstraction.String())
		}
	}

	return arguments, nil
}

// takeParam checks if any of the provided parameters can be used to satisfy the given abstraction.
// It returns the first matching parameter and a boolean indicating if a match was found.
func takeParam(abstraction reflect.Type, params []reflect.Value, usedParams []bool) (reflect.Value, bool) {
	for i, param := range params {
		if usedParams[i] {
			continue
		}

		if param.Type().AssignableTo(abstraction) {
			usedParams[i] = true
			if param.Type() == abstraction {
				return param, true
			}

			return param.Convert(abstraction), true
		}
	}

	return reflect.Value{}, false
}
