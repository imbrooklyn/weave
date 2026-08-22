package weave

import "github.com/imbrooklyn/weave/when"

// Group is a temporary mutable view used while a Scope callback is active. It
// freezes when the callback returns or a panic unwinds through it. An included
// method call after freezing cannot modify the tree and records ErrInvalidState
// on the owning Builder. Group is not safe for concurrent use and must not be
// retained after the callback returns.
type Group[E any] struct {
	state   *constructionState
	node    *groupNode
	depth   int
	control *groupControl
}

// Scope populates one explicit Boolean group. The Group argument is valid only
// for the duration of the call.
type Scope[E any] func(*Group[E])

func (g *Group[E]) constructionContext() constructionContext {
	if g == nil || g.state == nil || g.node == nil || g.control == nil {
		panic("weave: invalid zero group")
	}
	return constructionContext{
		state:       g.state,
		parent:      g.node,
		parentDepth: g.depth,
		active:      g.control.lifecycle == groupActive,
	}
}

// EQ adds an equality comparison when every predicate reports true.
func (g *Group[E]) EQ[T any](
	field any,
	value T,
	predicates ...when.Predicate[T],
) *Group[E] {
	addComparison(g.constructionContext(), OperatorEQ, field, value, predicates)
	return g
}

// NEQ adds a non-null inequality comparison when every predicate reports
// true.
func (g *Group[E]) NEQ[T any](
	field any,
	value T,
	predicates ...when.Predicate[T],
) *Group[E] {
	addComparison(g.constructionContext(), OperatorNEQ, field, value, predicates)
	return g
}

// LT adds a less-than comparison when every predicate reports true.
func (g *Group[E]) LT[T any](
	field any,
	value T,
	predicates ...when.Predicate[T],
) *Group[E] {
	addComparison(g.constructionContext(), OperatorLT, field, value, predicates)
	return g
}

// LTE adds a less-than-or-equal comparison when every predicate reports true.
func (g *Group[E]) LTE[T any](
	field any,
	value T,
	predicates ...when.Predicate[T],
) *Group[E] {
	addComparison(g.constructionContext(), OperatorLTE, field, value, predicates)
	return g
}

// GT adds a greater-than comparison when every predicate reports true.
func (g *Group[E]) GT[T any](
	field any,
	value T,
	predicates ...when.Predicate[T],
) *Group[E] {
	addComparison(g.constructionContext(), OperatorGT, field, value, predicates)
	return g
}

// GTE adds a greater-than-or-equal comparison when every predicate reports
// true.
func (g *Group[E]) GTE[T any](
	field any,
	value T,
	predicates ...when.Predicate[T],
) *Group[E] {
	addComparison(g.constructionContext(), OperatorGTE, field, value, predicates)
	return g
}

// In adds a membership test when every predicate reports true. A one-level
// pointer element type uses nil elements to represent explicit null values.
func (g *Group[E]) In[T any, S ~[]T](
	field any,
	values S,
	predicates ...when.Predicate[S],
) *Group[E] {
	addMembership(g.constructionContext(), OperatorIn, field, values, predicates)
	return g
}

// NotIn adds a non-null exclusion test when every predicate reports true.
func (g *Group[E]) NotIn[T any, S ~[]T](
	field any,
	values S,
	predicates ...when.Predicate[S],
) *Group[E] {
	addMembership(g.constructionContext(), OperatorNotIn, field, values, predicates)
	return g
}

// Between adds an inclusive numeric range when every predicate reports true.
func (g *Group[E]) Between[T when.Number](
	field any,
	lower T,
	upper T,
	predicates ...when.PairPredicate[T, T],
) *Group[E] {
	addRange(g.constructionContext(), field, lower, upper, predicates)
	return g
}

// IsNull adds an explicit-null test when every enabled value is true.
func (g *Group[E]) IsNull(field any, enabled ...bool) *Group[E] {
	addNull(g.constructionContext(), OperatorIsNull, field, enabled)
	return g
}

// NotNull adds a present, non-null test when every enabled value is true.
func (g *Group[E]) NotNull(field any, enabled ...bool) *Group[E] {
	addNull(g.constructionContext(), OperatorNotNull, field, enabled)
	return g
}

// Contains adds a literal substring test when every predicate reports true.
func (g *Group[E]) Contains(
	field any,
	value string,
	predicates ...when.Predicate[string],
) *Group[E] {
	addText(g.constructionContext(), OperatorContains, field, value, predicates)
	return g
}

// HasPrefix adds a literal prefix test when every predicate reports true.
func (g *Group[E]) HasPrefix(
	field any,
	value string,
	predicates ...when.Predicate[string],
) *Group[E] {
	addText(g.constructionContext(), OperatorHasPrefix, field, value, predicates)
	return g
}

// HasSuffix adds a literal suffix test when every predicate reports true.
func (g *Group[E]) HasSuffix(
	field any,
	value string,
	predicates ...when.Predicate[string],
) *Group[E] {
	addText(g.constructionContext(), OperatorHasSuffix, field, value, predicates)
	return g
}

// AllOf adds a nested group that requires every child to match. A disabled
// group does not invoke scope.
func (g *Group[E]) AllOf(scope Scope[E], enabled ...bool) *Group[E] {
	addGroup(g.constructionContext(), LogicAllOf, scope, enabled)
	return g
}

// AnyOf adds a nested group that requires at least one child to match. A
// disabled group does not invoke scope.
func (g *Group[E]) AnyOf(scope Scope[E], enabled ...bool) *Group[E] {
	addGroup(g.constructionContext(), LogicAnyOf, scope, enabled)
	return g
}

// NoneOf adds a nested group that requires no child to match. A disabled group
// does not invoke scope.
func (g *Group[E]) NoneOf(scope Scope[E], enabled ...bool) *Group[E] {
	addGroup(g.constructionContext(), LogicNoneOf, scope, enabled)
	return g
}

// NotAllOf adds a nested group that requires at least one child not to match.
// A disabled group does not invoke scope.
func (g *Group[E]) NotAllOf(scope Scope[E], enabled ...bool) *Group[E] {
	addGroup(g.constructionContext(), LogicNotAllOf, scope, enabled)
	return g
}

// Expr adds an opaque adapter-native expression when every enabled value is
// true. Core does not inspect, clone, or validate expression.
func (g *Group[E]) Expr(expression E, enabled ...bool) *Group[E] {
	addNativeExpression(g.constructionContext(), expression, enabled)
	return g
}
