package registerar

import "reflect"

// Slot identifies a registration: the type a binding produces plus its name.
type Slot struct {
	Type reflect.Type
	Name string
}

// String renders the Slot as `main.Shape`, or `main.Shape("cache")` when it is named.
func (s Slot) String() string {
	if s.Type == nil {
		return "<unknown>"
	}

	if s.Name == "" {
		return s.Type.String()
	}

	return s.Type.String() + `("` + s.Name + `")`
}

// Registration pairs a binding with the slot it is registered in.
type Registration struct {
	Slot    Slot
	Binding *Binding
}
