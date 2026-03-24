package registerar

import (
	"reflect"
	"sync"

	"github.com/danceable/container/errors"
)

// Registrar manages binding registerations.
type Registrar struct {
	bindings map[reflect.Type]map[string]*Binding

	mu sync.RWMutex
}

// NewRegisterar creates and returns a new Registrar instance.
func NewRegisterar() *Registrar {
	return &Registrar{bindings: make(map[reflect.Type]map[string]*Binding)}
}

// Reset clears all bindings.
func (r *Registrar) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.bindings = make(map[reflect.Type]map[string]*Binding)
}

// Delete removes the binding by exact type match.
func (r *Registrar) Delete(t reflect.Type, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if named, ok := r.bindings[t]; ok {
		delete(named, name)
	}
}

// Get retrieves a binding by exact type match.
func (r *Registrar) Get(t reflect.Type, name string) (*Binding, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if named, bindingExists := r.bindings[t]; bindingExists {
		if b, ok := named[name]; ok {
			return b, true
		}
	}

	return nil, false
}

// Find retrieves a binding by exact type match, falling back to interface-implementation lookup.
func (r *Registrar) Find(abstraction reflect.Type, name string) (*Binding, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.find(abstraction, name)
}

// Set checks for a circular dependency and, if none is found, atomically stores the binding.
func (r *Registrar) Set(t reflect.Type, name string, b *Binding) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.hasCircularDependencies(t, name, reflect.TypeOf(b.resolver)) {
		return errors.ErrCircularDependency
	}

	if _, exist := r.bindings[t]; !exist {
		r.bindings[t] = make(map[string]*Binding)
	}
	r.bindings[t][name] = b

	return nil
}

// SetIfAbsent is like Set but only stores b when the slot is currently empty.
// Returns wasNew=true if b was stored, false if the slot was already taken.
func (r *Registrar) SetIfAbsent(t reflect.Type, name string, b *Binding) (wasNew bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if named, ok := r.bindings[t]; ok {
		if _, ok2 := named[name]; ok2 {
			return false, nil
		}
	}

	if r.hasCircularDependencies(t, name, reflect.TypeOf(b.resolver)) {
		return false, errors.ErrCircularDependency
	}

	if _, exist := r.bindings[t]; !exist {
		r.bindings[t] = make(map[string]*Binding)
	}

	r.bindings[t][name] = b

	return true, nil
}

// find retrieves a binding by exact type match, falling back to interface-implementation lookup.
func (r *Registrar) find(abstraction reflect.Type, name string) (*Binding, bool) {
	if named, bindingExists := r.bindings[abstraction]; bindingExists {
		if b, ok := named[name]; ok {
			return b, true
		}
	}

	if abstraction.Kind() == reflect.Interface {
		for boundType, namedConcretes := range r.bindings {
			if boundType.Implements(abstraction) {
				if b, ok := namedConcretes[name]; ok {
					return b, true
				}
			}
		}
	}

	return nil, false
}

// hasCircularDependencies checks if the resolver function for a binding would introduce a circular dependency.
func (r *Registrar) hasCircularDependencies(outType reflect.Type, name string, resolverType reflect.Type) bool {
	type node struct {
		t    reflect.Type
		name string
	}
	visited := map[node]bool{}

	var dfs func(typ reflect.Type, depName string) bool
	dfs = func(typ reflect.Type, depName string) bool {
		if typ == outType && depName == name {
			return true // reached the slot we are about to occupy — cycle confirmed
		}

		n := node{typ, depName}
		if visited[n] {
			return false
		}
		visited[n] = true

		b, exists := r.find(typ, depName)
		if !exists {
			return false
		}

		rt := reflect.TypeOf(b.resolver)
		for in := range rt.Ins() {
			if dfs(in, "") {
				return true
			}
		}

		return false
	}

	for in := range resolverType.Ins() {
		if dfs(in, "") {
			return true
		}
	}

	return false
}
