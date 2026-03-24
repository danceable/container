[![Go Reference](https://pkg.go.dev/badge/github.com/danceable/container.svg)](https://pkg.go.dev/github.com/danceable/container)
[![CI](https://github.com/danceable/container/actions/workflows/ci.yml/badge.svg)](https://github.com/danceable/container/actions/workflows/ci.yml)
[![CodeQL](https://github.com/danceable/container/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/danceable/container/actions/workflows/codeql-analysis.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/danceable/container)](https://goreportcard.com/report/github.com/danceable/container)
[![Coverage Status](https://coveralls.io/repos/github/danceable/container/badge.svg)](https://coveralls.io/github/danceable/container?branch=main)

<p align="center">
  <img src="logo.svg" alt="Container Logo" />
</p>

# Container
Container is a lightweight yet powerful IoC (dependency injection) container for Go projects.
It's built neat, easy-to-use, and performance-in-mind to be your ultimate requirement.

Features:
- Singleton and Transient bindings
- Named dependencies (bindings)
- Resolve by functions, variables, and structs
- Must helpers that convert errors to panics
- Optional lazy loading of bindings
- Global instance for small applications
- Concurrency-safe with no race conditions
- Bind-time and resolve-time parameter injection
- 100% Test coverage!

## Documentation
### Required Go Versions
It requires Go `v1.26` or newer versions.

### Installation
To install this package, run the following command in your project directory.

```bash
go get github.com/danceable/container
```

Next, include it in your application:

```go
import "github.com/danceable/container"
```

### Introduction
Container works by binding abstractions (interfaces) to their concrete implementations via resolver functions.
You register a binding with `Bind()`, passing a resolver function that returns the concrete type, along with optional configuration:

- **Singleton** (`bind.Singleton()`): The resolver is called once; the same instance is returned for every subsequent request.
- **Transient** (default): The resolver is called on every request, producing a new instance each time.
- **Named** (`bind.WithName("...")`) : Multiple concretes can be registered for the same abstraction under different names.
- **Lazy** (`bind.Lazy()`): Defers the resolver invocation until the binding is first resolved.

Once bindings are registered, you can retrieve concretes through:
- `Resolve(&target)` — fills a variable with the bound concrete.
- `Call(fn)` — invokes a function whose parameters are automatically resolved from the container.
- `Fill(&struct)` — injects dependencies into struct fields tagged with `container:"type"` or `container:"name"`.

Your code depends on abstractions, not implementations!

### Quick Start
The following example demonstrates a simple binding and resolving.

```go
err := container.Bind(func() Config {
    return &JsonConfig{...}
}, bind.Singleton())

var c Config
err = container.Resolve(&c)
```

### Examples

#### Global Instance

The package provides a default global `Container` instance — exposed as `container.Default` — for convenience in small applications. Instead of creating a container with `container.New()`, you can call `container.Bind()`, `container.Resolve()`, `container.Call()`, `container.Fill()`, and `container.Reset()` directly as package-level functions; they all delegate to `container.Default`.

You can also access `container.Default` directly if you need to pass the global instance to a function or a `Must` helper.

```go
// No need to create a container — uses the global instance (container.Default)
container.Bind(func() Database {
    return &MySQL{Host: "localhost"}
}, bind.Singleton())

var db Database
container.Resolve(&db)

container.Call(func(db Database) {
    db.Connect()
})

// Pass the global instance to a Must helper
container.MustBind(container.Default, func() Cache {
    return &RedisCache{}
}, bind.Singleton())

// Reset clears all bindings from the global instance
container.Reset()
```

#### Singleton Binding

A singleton binding creates one shared instance. The resolver is called once, and every subsequent resolve returns the same object.

```go
c := container.New()

err := c.Bind(func() Database {
    return &MySQL{Host: "localhost", Port: 3306}
}, bind.Singleton())

var db1, db2 Database

// db1 and db2 point to the same instance
c.Resolve(&db1)
c.Resolve(&db2)
```

#### Transient Binding

A transient binding (the default) calls the resolver on every resolve, producing a fresh instance each time.

```go
c := container.New()

err := c.Bind(func() Logger {
    return &FileLogger{Path: "/var/log/app.log"}
})

var l1, l2 Logger

// l1 and l2 are different instances
c.Resolve(&l1)
c.Resolve(&l2)
```

#### Named Binding

Named bindings allow registering multiple concretes for the same abstraction under different names.

```go
c := container.New()

c.Bind(func() Database {
    return &MySQL{Host: "primary"}
}, bind.Singleton(), bind.WithName("primary"))

c.Bind(func() Database {
    return &MySQL{Host: "replica"}
}, bind.Singleton(), bind.WithName("replica"))
```

#### Lazy Binding

A lazy binding defers resolver invocation until the first time the binding is resolved, rather than at bind time. This is useful when a dependency isn't always needed or is expensive to create.

```go
c := container.New()

err := c.Bind(func() Cache {
    return NewRedisCache("localhost:6379") // not called until first resolve
}, bind.Singleton(), bind.Lazy())
```

#### Eager Binding

An eager binding (the default) invokes the resolver immediately at bind time to validate it. For singletons, this also creates and caches the instance right away.

```go
c := container.New()

// The resolver runs immediately — any error is returned by Bind.
err := c.Bind(func() Database {
    return &MySQL{Host: "localhost"}
}, bind.Singleton())
```

#### Resolving by Name

Use `resolve.WithName()` to retrieve a specific named binding during `Resolve`, `Call`, or `Fill`.

```go
c := container.New()

c.Bind(func() Database {
    return &MySQL{Host: "primary"}
}, bind.Singleton(), bind.WithName("primary"))

c.Bind(func() Database {
    return &MySQL{Host: "replica"}
}, bind.Singleton(), bind.WithName("replica"))

var replica Database
c.Resolve(&replica, resolve.WithName("replica"))
```

#### Resolving with Runtime Parameters

Use `resolve.WithParams()` to supply values at resolve time. These are matched by type to the resolver's arguments and take precedence over container bindings.

```go
c := container.New()

c.Bind(func(dsn string) Database {
    return &MySQL{DSN: dsn}
}, bind.Lazy())

var db Database
c.Resolve(&db, resolve.WithParams("user:pass@tcp(localhost)/mydb"))
```

#### Binding with Pre-set Parameters

Use `bind.ResolveDepenenciesByParams()` to lock in specific parameter values at bind time. These take precedence over container lookups but can still be overridden by `resolve.WithParams()`.

```go
c := container.New()

c.Bind(func(timeout int) Cache {
    return &RedisCache{Timeout: timeout}
}, bind.Lazy(), bind.ResolveDepenenciesByParams(30))

var cache Cache
c.Resolve(&cache) // resolver receives timeout=30
```

#### Binding with Named Dependency Resolution

Use `bind.ResolveDependenciesByNamedBindings()` to wire a resolver's arguments to specific named bindings instead of the default (unnamed) ones.

```go
c := container.New()

c.Bind(func() Database {
    return &MySQL{Host: "replica"}
}, bind.WithName("replica"), bind.Singleton(), bind.Lazy())

c.Bind(func(db Database) ReportService {
    return &Reporter{DB: db}
}, bind.Lazy(), bind.ResolveDependenciesByNamedBindings("replica"))

var svc ReportService
c.Resolve(&svc) // Reporter receives the "replica" Database
```

#### Parameter Resolution Precedence

When a resolver function has arguments, the container resolves them using multiple sources. If the same argument type is available from more than one source, the following precedence applies (highest to lowest):

1. **Resolve-time params** (`resolve.WithParams()`) — values passed when calling `Resolve` or `Call`.
2. **Bind-time params** (`bind.ResolveDepenenciesByParams()`) — values locked in at binding time.
3. **Named bindings** (`bind.ResolveDependenciesByNamedBindings()`) — values pulled from named container entries.
4. **Container lookup** — the default unnamed binding for the matching type.

```go
c := container.New()

// 4. Container lookup (lowest priority)
c.Bind(func() Shape { return &Circle{Area: 99} }, bind.Singleton(), bind.Lazy())

// 3. Named binding
c.Bind(func() Shape { return &Circle{Area: 5} }, bind.WithName("special"), bind.Singleton(), bind.Lazy())

// Resolver with bind-time params (2) and named bindings (3) configured
c.Bind(func(x int, s Shape) Database {
    return &PostgreSQL{X: x, Area: s.GetArea()}
}, bind.Lazy(),
    bind.ResolveDepenenciesByParams(10),              // 2. bind-time param for int
    bind.ResolveDependenciesByNamedBindings("special"), // 3. named binding for Shape
)

var db Database

// Without resolve-time params: int=10 (bind-time), Shape.Area=5 (named binding)
c.Resolve(&db)

// With resolve-time params (highest priority): overrides both int and Shape
c.Resolve(&db, resolve.WithParams(42, &Circle{Area: 77}))
```

#### Container Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `New` | `New() *Container` | Creates a new container instance. |
| `Bind` | `Bind(resolver any, opts ...bind.BindOption) error` | Registers a resolver function that maps an abstraction to its concrete implementation. |
| `Resolve` | `Resolve(abstraction any, opts ...resolve.ResolveOption) error` | Fills a pointer-to-interface (or pointer-to-type) with the matching concrete from the container. |
| `Call` | `Call(function any, opts ...resolve.ResolveOption) error` | Invokes a function whose parameters are automatically resolved from the container. The function may optionally return an `error`. |
| `Fill` | `Fill(structure any, opts ...resolve.ResolveOption) error` | Injects dependencies into struct fields tagged with `container:"type"` or `container:"name"`. |
| `Reset` | `Reset()` | Removes all bindings and empties the container. |

Each method also has a `Must` variant (`MustBind`, `MustResolve`, `MustCall`, `MustFill`) that panics on error instead of returning it:

```go
// These two are equivalent:
err := c.Bind(func() Database { return &MySQL{} }, bind.Singleton())
if err != nil { panic(err) }

container.MustBind(c, func() Database { return &MySQL{} }, bind.Singleton())
```

#### Bind Options

Options passed to `Bind()` to configure how a binding behaves.

| Option | Description |
|--------|-------------|
| `bind.Singleton()` | Marks the binding as a singleton — the resolver is called once, and the same instance is returned on every resolve. |
| `bind.Lazy()` | Defers resolver invocation until the binding is first resolved. Without this, the resolver runs eagerly at bind time. |
| `bind.WithName(name)` | Assigns a name to the binding, allowing multiple concretes for the same abstraction. |
| `bind.ResolveDepenenciesByParams(params...)` | Provides concrete values at bind time to satisfy the resolver's arguments (matched by type). |
| `bind.ResolveDependenciesByNamedBindings(names...)` | Specifies named bindings to use when resolving the resolver's arguments (matched positionally). |

```go
c.Bind(func() Database {
    return &MySQL{Host: "replica"}
}, bind.Singleton(), bind.Lazy(), bind.WithName("replica"))
```

#### Resolve Options

Options passed to `Resolve()`, `Call()`, or `Fill()` to customize how a binding is looked up and invoked.

| Option | Description |
|--------|-------------|
| `resolve.WithName(name)` | Selects a specific named binding instead of the default (unnamed) one. |
| `resolve.WithParams(params...)` | Supplies runtime values to satisfy the resolver's arguments (matched by type). These take the highest precedence. |

```go
var db Database
c.Resolve(&db, resolve.WithName("replica"), resolve.WithParams("custom-dsn"))
```

#### Must Helpers (Panic on Error)

Every container method (`Bind`, `Resolve`, `Call`, `Fill`) has a corresponding `Must` variant that panics instead of returning an error. These are package-level functions that accept the container as the first argument. They are useful in application setup code where a failed binding or resolution indicates a programming error that should halt execution immediately.

| Function | Wraps | Description |
|----------|-------|-------------|
| `MustBind(c, resolver, opts...)` | `c.Bind(...)` | Registers a binding or panics. |
| `MustResolve(c, abstraction, opts...)` | `c.Resolve(...)` | Resolves a dependency or panics. |
| `MustCall(c, function, opts...)` | `c.Call(...)` | Calls a function with injected dependencies or panics. |
| `MustFill(c, structure, opts...)` | `c.Fill(...)` | Fills struct fields or panics. |

```go
c := container.New()

// Panics if the binding fails
container.MustBind(c, func() Database {
    return &MySQL{Host: "localhost"}
}, bind.Singleton())

// Panics if the resolution fails
var db Database
container.MustResolve(c, &db)

// Panics if the call fails
container.MustCall(c, func(db Database) {
    db.Connect()
})

// Panics if filling fails
type App struct {
    DB Database `container:"type"`
}
var app App
container.MustFill(c, &app)
```
