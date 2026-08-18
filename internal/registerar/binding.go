package registerar

import (
	"iter"
	"reflect"
	"sync"
)

// Binding holds a resolver and a concrete (if already resolved).
// It is the break for the Container wall!
type Binding struct {
	name          string          // Binding name
	isSingleton   bool            // isSingleton is true if the Binding is a singleton.
	bindParams    []reflect.Value // bindParams holds params specified at bind time for dependency resolution.
	namedBindings []string        // namedBindings holds named Binding names specified at bind time.
	resolver      any             // resolver is the function that is responsible for making the concrete.
	concrete      any             // concrete is the stored instance for singleton Bindings.
	dependencies  []reflect.Type  // dependencies holds the resolver inputs the container resolves.

	mu sync.RWMutex // mux is a mutex that guards singleton initialization.
}

// NewBinding creates and returns a new Binding instance.
func NewBinding(
	name string,
	isSingleton bool,
	bindParams []reflect.Value,
	namedBindings []string,
	resolver any,
	concrete any,
) *Binding {
	b := &Binding{
		name:          name,
		isSingleton:   isSingleton,
		bindParams:    bindParams,
		namedBindings: namedBindings,
		resolver:      resolver,
		concrete:      concrete,
	}
	b.dependencies = b.unsatisfiedInputs()

	return b
}

// GetName returns the name of the Binding.
func (b *Binding) GetName() string {
	return b.name
}

// IsSingleton specifies whether the Binding is a singleton.
func (b *Binding) IsSingleton() bool {
	return b.isSingleton
}

// BindParams returns the parameters specified at bind time for dependency resolution.
func (b *Binding) BindParams() []reflect.Value {
	return b.bindParams
}

// NamedBindings returns the named Bindings specified at bind time for dependency resolution.
func (b *Binding) NamedBindings() []string {
	return b.namedBindings
}

// Resolver returns the resolver function of the Binding.
func (b *Binding) Resolver() any {
	return b.resolver
}

// Dependencies returns the resolver inputs left unsatisfied by the parameters given at
// bind time, worked out once since neither of the two ever changes.
func (b *Binding) Dependencies() []reflect.Type {
	return b.dependencies
}

func (b *Binding) unsatisfiedInputs() []reflect.Type {
	resolver := reflect.TypeOf(b.resolver)
	used := make([]bool, len(b.bindParams))

	var dependencies []reflect.Type
	for in := range resolver.Ins() {
		if !takes(in, b.bindParams, used) {
			dependencies = append(dependencies, in)
		}
	}

	return dependencies
}

// DependencyNames yields the names to look a dependency up under, in the order the
// container tries them.
func (b *Binding) DependencyNames() iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, name := range b.namedBindings {
			if !yield(name) {
				return
			}
		}

		yield(b.name)
	}
}

// takes reports whether an unused param satisfies the input, marking the one it takes.
func takes(in reflect.Type, params []reflect.Value, used []bool) bool {
	for i, param := range params {
		if used[i] {
			continue
		}

		if param.Type().AssignableTo(in) {
			used[i] = true

			return true
		}
	}

	return false
}

// HasConcrete checks if the Binding has a concrete instance.
func (b *Binding) HasConcrete() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.concrete != nil
}

// GetOrSetConcrete returns the existing concrete if set, otherwise calls factory exactly once
// to create it. Safe for concurrent use.
func (b *Binding) GetOrSetConcrete(
	factory func(b *Binding, params []reflect.Value) (any, error),
	params []reflect.Value,
) (any, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.concrete != nil {
		return b.concrete, nil
	}

	concrete, err := factory(b, params)
	if err != nil {
		return nil, err
	}

	b.concrete = concrete

	return b.concrete, nil
}
