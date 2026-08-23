package adapteralias

import "github.com/imbrooklyn/weave"

// Conditions is the named slice result used by this alias fixture.
type Conditions []string

// Expression is the native expression carrier used by this alias fixture.
type Expression = string

// Factory exposes the adapter-bound core factory shape.
type Factory = weave.Factory[Conditions, Expression]

// Group exposes the adapter-bound concrete generic group shape.
type Group = weave.Group[Expression]

// Scope exposes the adapter-bound scope shape.
type Scope = weave.Scope[Expression]

// Predicate exposes the adapter-bound predicate shape.
type Predicate = weave.Predicate[Conditions, Expression]

// Compiler exposes the adapter-bound compiler interface shape.
type Compiler = weave.Compiler[Conditions, Expression]

type compiler struct{}

// Compile implements Compiler without a generic interface method.
func (compiler) Compile(Predicate) (Conditions, error) {
	return Conditions{"compiled"}, nil
}

// Capabilities returns the fixture compiler capabilities.
func (compiler) Capabilities() weave.Capabilities {
	return weave.Capabilities{}
}

var _ Compiler = compiler{}

// NewFactory returns the aliased adapter factory shape.
func NewFactory() *Factory {
	return weave.NewFactory[Conditions, Expression](compiler{})
}

func reusableScope() Scope {
	return func(group *Group) {
		group.EQ("field", 1).Expr("native expression")
	}
}

func compileAliasShapes() {
	NewFactory().New().
		AllOf(reusableScope()).
		Native(Conditions{"native condition"})
}
