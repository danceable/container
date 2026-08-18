package registerar

import "reflect"

// registration is what a slot holds: the binding, and its node in the dependency graph.
type registration struct {
	binding *Binding
	node    int
}

// store holds the registrations and answers lookups. It is the data structure alone:
// Registrar owns the locking around it, and acyclic the rules about what may go in.
type store struct {
	bindings map[reflect.Type]map[string]registration

	// nodes maps a dependencyGraph node back to its slot; the way there is the node the
	// slot holds. free holds the nodes deleted bindings left behind.
	nodes []Slot
	free  []int
}

func newStore() store {
	return store{bindings: make(map[reflect.Type]map[string]registration)}
}

func (s *store) reset() {
	s.bindings = make(map[reflect.Type]map[string]registration)
	s.nodes = nil
	s.free = nil
}

// get retrieves a binding by exact type match.
func (s *store) get(t reflect.Type, name string) (*Binding, bool) {
	if named, exist := s.bindings[t]; exist {
		if reg, ok := named[name]; ok {
			return reg.binding, true
		}
	}

	return nil, false
}

// find is lookup, keeping only the binding.
func (s *store) find(abstraction reflect.Type, name string) (*Binding, bool) {
	_, reg, exist := s.lookup(abstraction, name)

	return reg.binding, exist
}

// lookup retrieves the slot and the registration for the abstraction by exact type match,
// falling back to interface-implementation lookup. Where several types implement the
// interface, the one registered first answers, as the cycle check assumes.
func (s *store) lookup(abstraction reflect.Type, name string) (Slot, registration, bool) {
	if named, exist := s.bindings[abstraction]; exist {
		if reg, ok := named[name]; ok {
			return Slot{abstraction, name}, reg, true
		}
	}

	if abstraction.Kind() != reflect.Interface {
		return Slot{}, registration{}, false
	}

	var (
		match Slot
		found registration
		exist bool
	)

	for boundType, namedConcretes := range s.bindings {
		if !boundType.Implements(abstraction) {
			continue
		}

		reg, ok := namedConcretes[name]
		if !ok {
			continue
		}

		if !exist || reg.node < found.node {
			match, found, exist = Slot{boundType, name}, reg, true
		}
	}

	return match, found, exist
}

// taken reports whether the slot already holds a binding.
func (s *store) taken(slot Slot) bool {
	_, exist := s.bindings[slot.Type][slot.Name]

	return exist
}

// registration returns what the slot holds, if anything.
func (s *store) registration(slot Slot) (registration, bool) {
	if slot.Type == nil {
		return registration{}, false // a tombstone
	}

	reg, exist := s.bindings[slot.Type][slot.Name]

	return reg, exist
}

// registrations lists every registration, in the order the slots were first registered.
func (s *store) registrations() []Registration {
	registrations := make([]Registration, 0, len(s.nodes))
	for _, slot := range s.nodes {
		if reg, exist := s.registration(slot); exist {
			registrations = append(registrations, Registration{Slot: slot, Binding: reg.binding})
		}
	}

	return registrations
}

// put stores the binding in the slot, returning its node and the binding it displaced.
func (s *store) put(slot Slot, b *Binding) (node int, previous *Binding, replaced bool) {
	named, exist := s.bindings[slot.Type]
	if !exist {
		named = make(map[string]registration)
		s.bindings[slot.Type] = named
	}

	reg, replaced := named[slot.Name]
	if !replaced {
		reg.node = s.track(slot)
	}

	previous, reg.binding = reg.binding, b
	named[slot.Name] = reg

	return reg.node, previous, replaced
}

// restore undoes put.
func (s *store) restore(slot Slot, previous *Binding, replaced bool) {
	named := s.bindings[slot.Type]

	if replaced {
		reg := named[slot.Name]
		reg.binding = previous
		named[slot.Name] = reg

		return
	}

	node := named[slot.Name].node
	delete(named, slot.Name)
	if len(named) == 0 {
		delete(s.bindings, slot.Type)
	}

	s.untrack(node)
}

// delete removes the binding held by the slot.
func (s *store) delete(t reflect.Type, name string) {
	named, exist := s.bindings[t]
	if !exist {
		return
	}

	if reg, exist := named[name]; exist {
		delete(named, name)
		s.untrack(reg.node)
	}
}

// track numbers the slot as a graph node, reusing one a deleted binding left behind.
func (s *store) track(slot Slot) int {
	if free := len(s.free) - 1; free >= 0 {
		node := s.free[free]
		s.free = s.free[:free]
		s.nodes[node] = slot

		return node
	}

	s.nodes = append(s.nodes, slot)

	return len(s.nodes) - 1
}

// untrack drops the given node, leaving a reusable tombstone unless it is the last one:
// removing it outright would renumber the nodes that follow.
func (s *store) untrack(node int) {
	if node == len(s.nodes)-1 {
		s.nodes = s.nodes[:node]

		return
	}

	s.nodes[node] = Slot{}
	s.free = append(s.free, node)
}
