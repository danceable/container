package errors

import "errors"

var (
	// ErrNonFunctionResolver is returned when the resolver is not a function.
	ErrNonFunctionResolver = errors.New("container: the resolver must be a function")

	// ErrInvalidResolver is returned when the resolver function signature is invalid.
	ErrInvalidResolver = errors.New("container: resolver function signature is invalid - it must return abstract, or abstract and error")

	// ErrResolverDependsOnAbstract is returned when the resolver function depends on the abstract it returns.
	ErrResolverDependsOnAbstract = errors.New("container: resolver function signature is invalid - depends on abstract it returns")

	// ErrInvalidAbstraction is returned when the abstraction provided to Resolve is invalid.
	ErrInvalidAbstraction = errors.New("container: invalid abstraction")

	// ErrEncounteredError is returned when an error is encountered while making a concrete.
	ErrEncounteredError = errors.New("container: encountered error while making concrete")

	// ErrNoConcreteFound is returned when no concrete is found for the given abstraction.
	ErrNoConcreteFound = errors.New("container: no concrete found for the given abstraction")

	// ErrInvalidFunction is returned when the function provided to Call is invalid.
	ErrInvalidFunction = errors.New("container: invalid function")

	// ErrInvalidFunctionSignature is returned when the function signature is invalid.
	ErrInvalidFunctionSignature = errors.New("container: receiver function signature is invalid")

	// ErrInvalidStructure is returned when the structure provided to Fill is invalid.
	ErrInvalidStructure = errors.New("container: invalid structure")

	// ErrInvalidStructTag is returned when a struct field has an invalid struct tag.
	ErrInvalidStructTag = errors.New("container: invalid struct tag")

	// ErrCannotMakeField is returned when a field with the `container` tag cannot be made.
	ErrCannotMakeField = errors.New("container: cannot make field")

	// ErrCircularDependency is returned when a circular dependency is detected during singleton initialization.
	ErrCircularDependency = errors.New("container: circular dependency detected")
)
