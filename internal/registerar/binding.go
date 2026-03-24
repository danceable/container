package registerar

import (
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
	return &Binding{
		name:          name,
		isSingleton:   isSingleton,
		bindParams:    bindParams,
		namedBindings: namedBindings,
		resolver:      resolver,
		concrete:      concrete,
	}
}

// HasName specifies whether the Binding has a name.
func (b *Binding) HasName() bool {
	return len(b.name) > 0
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

// HasConcrete checks if the Binding has a concrete instance.
func (b *Binding) HasConcrete() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.concrete != nil
}

// Concrete returns the concrete instance of the Binding if it is a singleton and has been resolved, otherwise it returns nil.
func (b *Binding) Concrete() any {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.concrete
}

// SetConcrete sets the concrete instance of the Binding. Safe for concurrent use.
func (b *Binding) SetConcrete(concrete any) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.concrete = concrete
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
