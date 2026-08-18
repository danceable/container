package container

import (
	"reflect"
	"slices"
	"sync"

	"github.com/danceable/container/internal/registerar"
)

// Container holds the registrar and provides methods to interact with bindings.
// It is the entry point in the package.
//
// Containers form a scope tree: every Container created with New is a root scope,
// and Scope derives nested child scopes from it. A scope can resolve bindings
// registered on itself or any of its ancestors, but bindings are never visible to
// ancestor or sibling scopes.
type Container struct {
	reg *registerar.Registrar

	name     string                // name is the scope name; empty for a root scope.
	root     *Container            // root points to the top-most scope of the tree.
	parent   *Container            // parent points to the enclosing scope; nil for a root scope.
	children map[string]*Container // children holds the nested scopes keyed by name.

	mu sync.Mutex // mu guards the children map.
}

// New creates a new root scope of the Container.
func New() *Container {
	c := &Container{
		reg:      registerar.NewRegisterar(),
		children: make(map[string]*Container),
	}
	c.root = c

	return c
}

// Scope returns a nested child scope that shares the same root scope, forming a scope tree.
// The child can resolve bindings registered on itself or on any ancestor scope, while its own
// bindings stay invisible to ancestor and sibling scopes. Calling Scope with a name that already
// exists on this scope returns the existing child, so the tree never holds duplicate siblings.
func (c *Container) Scope(name string) *Container {
	c.mu.Lock()
	defer c.mu.Unlock()

	if child, exist := c.children[name]; exist {
		return child
	}

	child := &Container{
		reg:      registerar.NewRegisterar(),
		name:     name,
		root:     c.root,
		parent:   c,
		children: make(map[string]*Container),
	}
	c.children[name] = child

	return child
}

// Derive returns an anonymous child scope that is NOT registered on its parent.
// It resolves bindings from itself and its ancestors exactly like a named Scope, but because
// the parent keeps no reference to it, the scope — and the bindings it owns — becomes eligible
// for garbage collection as soon as the caller drops it. This makes Derive the cheapest way to
// create an ephemeral, per-operation scope: there is nothing to clean up.
func (c *Container) Derive() *Container {
	return &Container{
		reg:      registerar.NewRegisterar(),
		root:     c.root,
		parent:   c,
		children: make(map[string]*Container),
	}
}

// Name returns the name of the scope, empty for a root or a derived one.
func (c *Container) Name() string {
	return c.name
}

// Root returns the root scope of the tree the scope belongs to.
func (c *Container) Root() *Container {
	return c.root
}

// Parent returns the enclosing scope, or nil if the scope is a root scope.
func (c *Container) Parent() *Container {
	return c.parent
}

// visibleScopes returns the ancestors of c from the root down, then c and its named
// descendants. A scope made with Derive can only appear as c, being unreachable
// from its parent.
func (c *Container) visibleScopes() []*Container {
	var ancestors []*Container
	for scope := c.parent; scope != nil; scope = scope.parent {
		ancestors = append(ancestors, scope)
	}
	slices.Reverse(ancestors)

	scopes := append(ancestors, c)

	// Each scope appended is walked in turn, descending the whole subtree.
	for i := len(ancestors); i < len(scopes); i++ {
		scopes = append(scopes, scopes[i].namedChildren()...)
	}

	return scopes
}

// namedChildren returns the child scopes of c, ordered by name so a container always
// draws the same.
func (c *Container) namedChildren() []*Container {
	c.mu.Lock()
	defer c.mu.Unlock()

	names := make([]string, 0, len(c.children))
	for name := range c.children {
		names = append(names, name)
	}
	slices.Sort(names)

	children := make([]*Container, 0, len(names))
	for _, name := range names {
		children = append(children, c.children[name])
	}

	return children
}

// lookup walks up the scope tree from c and returns the first binding registered for the
// exact type and name, together with the scope that owns it — the scope its own
// dependencies are then resolved from.
func (c *Container) lookup(t reflect.Type, name string) (*registerar.Binding, *Container, bool) {
	for scope := c; scope != nil; scope = scope.parent {
		if binding, exist := scope.reg.Get(t, name); exist {
			return binding, scope, true
		}
	}

	return nil, nil, false
}

// find is lookup with the interface-implementation fallback of the registrar.
func (c *Container) find(abstraction reflect.Type, name string) (*registerar.Binding, *Container, bool) {
	for scope := c; scope != nil; scope = scope.parent {
		if binding, exist := scope.reg.Find(abstraction, name); exist {
			return binding, scope, true
		}
	}

	return nil, nil, false
}
