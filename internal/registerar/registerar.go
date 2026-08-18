package registerar

import (
	"reflect"
	"sync"
)

// Registrar manages binding registerations. It is the way in to the store holding them
// and to the acyclic guard deciding what may join it, and owns the lock the two are used
// under.
type Registrar struct {
	store store

	mu sync.RWMutex
}

// NewRegisterar creates and returns a new Registrar instance.
func NewRegisterar() *Registrar {
	return &Registrar{store: newStore()}
}

// guard returns the acyclic guard over the registrations.
func (r *Registrar) guard() acyclic {
	return acyclic{store: &r.store}
}

// Reset clears all bindings.
func (r *Registrar) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.store.reset()
}

// Delete removes the binding by exact type match.
func (r *Registrar) Delete(t reflect.Type, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.store.delete(t, name)
}

// Get retrieves a binding by exact type match.
func (r *Registrar) Get(t reflect.Type, name string) (*Binding, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.store.get(t, name)
}

// Find retrieves a binding by exact type match, falling back to interface-implementation lookup.
func (r *Registrar) Find(abstraction reflect.Type, name string) (*Binding, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.store.find(abstraction, name)
}

// FindSlot is like Find but returns the slot holding the match instead of the binding.
func (r *Registrar) FindSlot(abstraction reflect.Type, name string) (Slot, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	slot, _, exist := r.store.lookup(abstraction, name)

	return slot, exist
}

// Registrations returns a snapshot of every registration, in the order the slots were
// first registered.
func (r *Registrar) Registrations() []Registration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.store.registrations()
}

// Set stores the binding, refusing it when the registration would introduce a
// circular dependency.
func (r *Registrar) Set(t reflect.Type, name string, b *Binding) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.guard().set(Slot{t, name}, b)
}

// SetIfAbsent is like Set but only stores b when the slot is currently empty.
// Returns wasNew=true if b was stored, false if the slot was already taken.
func (r *Registrar) SetIfAbsent(t reflect.Type, name string, b *Binding) (wasNew bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	slot := Slot{t, name}
	if r.store.taken(slot) {
		return false, nil
	}

	if err := r.guard().set(slot, b); err != nil {
		return false, err
	}

	return true, nil
}
