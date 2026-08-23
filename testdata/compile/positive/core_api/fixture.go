package coreapi

import (
	"time"

	"github.com/imbrooklyn/weave"
	"github.com/imbrooklyn/weave/when"
)

type condition []string
type expression struct{ name string }
type namedInt int64
type namedInts []namedInt

type compiler struct{}

// Compile implements weave.Compiler with a non-generic interface method.
func (compiler) Compile(weave.Predicate[condition, expression]) (condition, error) {
	return condition{"compiled"}, nil
}

// Capabilities returns the fixture compiler capabilities.
func (compiler) Capabilities() weave.Capabilities {
	return weave.Capabilities{}
}

var _ weave.Compiler[condition, expression] = compiler{}

func compileConcreteGenericMethods() {
	factory := weave.NewFactory[condition, expression](compiler{})
	values := namedInts{1, 2, 3}
	includeValues := when.Predicate[namedInts](when.NotEmpty[namedInts])
	includeRange := when.PairPredicate[namedInt, namedInt](when.ValidRange[namedInt])

	factory.New().
		EQ("field", namedInt(1), when.Positive[namedInt]).
		GTE("created_at", time.Unix(1, 0), when.NotZeroTime).
		In("field", values, includeValues).
		NotIn("field", values, when.NotEmpty[namedInts]).
		Between("field", namedInt(1), namedInt(3), includeRange, when.PairIf[namedInt, namedInt](true)).
		AllOf(func(group *weave.Group[expression]) {
			group.EQ("field", namedInt(1)).
				In("field", values, includeValues).
				Between("field", namedInt(1), namedInt(3), includeRange).
				Expr(expression{name: "native"})
		})
}
