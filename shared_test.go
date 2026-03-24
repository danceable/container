package container_test

type Shape interface {
	SetArea(int)
	GetArea() int
}

type ReadOnlyShape interface {
	GetArea() int
}

type Circle struct {
	a int
}

// Ensure Circle implements Shape interface.
var _ Shape = &Circle{}
var _ ReadOnlyShape = &Circle{}

func (c *Circle) SetArea(a int) {
	c.a = a
}

func (c Circle) GetArea() int {
	return c.a
}

type Database interface {
	Connect() bool
}

type MySQL struct{}

// Ensure MySQL implements Database interface.
var _ Database = MySQL{}

func (m MySQL) Connect() bool {
	return true
}

type PostgreSQL struct {
	ready bool
}

// Ensure PostgreSQL implements Database interface.
var _ Database = PostgreSQL{}

func (d PostgreSQL) Connect() bool {
	return d.ready
}

// Cache, Logger, Service — three independent interfaces used by deadlock / cycle tests.

type Cache interface{ Get() string }
type Logger interface{ Log() string }
type Service interface{ Run() string }

type InMemoryCache struct{}

var _ Cache = InMemoryCache{}

func (InMemoryCache) Get() string { return "cached" }

type StdLogger struct{}

var _ Logger = StdLogger{}

func (StdLogger) Log() string { return "logged" }

type AppService struct{}

var _ Service = AppService{}

func (AppService) Run() string { return "running" }
