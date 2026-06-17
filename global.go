package container

import (
	"github.com/danceable/container/bind"
	"github.com/danceable/container/resolve"
)

// Default is the default concrete of the Container.
var Default = New()

// Reset calls the same method of the default concrete.
func Reset() {
	Default.Reset()
}

// Scope calls the same method of the default concrete.
func Scope(name string) *Container {
	return Default.Scope(name)
}

// Derive calls the same method of the default concrete.
func Derive() *Container {
	return Default.Derive()
}

// Bind calls the same method of the default concrete.
func Bind(receiver any, opts ...bind.BindOption) error {
	return Default.Bind(receiver, opts...)
}

// Call calls the same method of the default concrete.
func Call(receiver any, opts ...resolve.ResolveOption) error {
	return Default.Call(receiver, opts...)
}

// Resolve calls the same method of the default concrete.
func Resolve(abstraction any, opts ...resolve.ResolveOption) error {
	return Default.Resolve(abstraction, opts...)
}

// Fill calls the same method of the default concrete.
func Fill(receiver any, opts ...resolve.ResolveOption) error {
	return Default.Fill(receiver, opts...)
}
