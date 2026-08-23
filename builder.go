package weave

import "github.com/imbrooklyn/weave/when"

// Builder constructs one adapter-bound predicate. C is the Compiler's final
// condition type, and E is its opaque nestable expression carrier. Each
// construction method reserves an Origin before evaluating inclusion controls,
// including when the call is later omitted. Builder is mutable and is not safe
// for concurrent use. Callers must serialize all method calls.
type Builder[C, E any] struct {
	domain  *predicateDomain
	factory *Factory[C, E]
	state   *constructionState
}

func newBuilder[C, E any]() *Builder[C, E] {
	return newBuilderForDomain[C, E](newPredicateDomain())
}

func newBuilderForDomain[C, E any](domain *predicateDomain) *Builder[C, E] {
	if domain == nil {
		panic("weave: nil predicate domain")
	}
	builder := &Builder[C, E]{domain: domain}
	builder.ensureState()
	return builder
}

func newBuilderForFactory[C, E any](factory *Factory[C, E]) *Builder[C, E] {
	if !validFactory(factory) {
		panic("weave: invalid factory")
	}
	builder := &Builder[C, E]{
		domain:  factory.domain,
		factory: factory,
	}
	builder.ensureState()
	return builder
}

func (b *Builder[C, E]) ensureState() *constructionState {
	if b == nil {
		panic("weave: nil builder")
	}
	if b.domain == nil {
		b.domain = newPredicateDomain()
	}
	if b.state == nil {
		b.state = newConstructionState()
	}
	return b.state
}

func (b *Builder[C, E]) constructionContext() constructionContext {
	state := b.ensureState()
	return constructionContext{
		state:  state,
		parent: state.root,
		active: true,
	}
}

// EQ adds an equality comparison when every predicate reports true.
func (b *Builder[C, E]) EQ[T any](
	field any,
	value T,
	predicates ...when.Predicate[T],
) *Builder[C, E] {
	addComparison(b.constructionContext(), OperatorEQ, field, value, predicates)
	return b
}

// NEQ adds a non-null inequality comparison when every predicate reports
// true.
func (b *Builder[C, E]) NEQ[T any](
	field any,
	value T,
	predicates ...when.Predicate[T],
) *Builder[C, E] {
	addComparison(b.constructionContext(), OperatorNEQ, field, value, predicates)
	return b
}

// LT adds a less-than comparison when every predicate reports true.
func (b *Builder[C, E]) LT[T any](
	field any,
	value T,
	predicates ...when.Predicate[T],
) *Builder[C, E] {
	addComparison(b.constructionContext(), OperatorLT, field, value, predicates)
	return b
}

// LTE adds a less-than-or-equal comparison when every predicate reports true.
func (b *Builder[C, E]) LTE[T any](
	field any,
	value T,
	predicates ...when.Predicate[T],
) *Builder[C, E] {
	addComparison(b.constructionContext(), OperatorLTE, field, value, predicates)
	return b
}

// GT adds a greater-than comparison when every predicate reports true.
func (b *Builder[C, E]) GT[T any](
	field any,
	value T,
	predicates ...when.Predicate[T],
) *Builder[C, E] {
	addComparison(b.constructionContext(), OperatorGT, field, value, predicates)
	return b
}

// GTE adds a greater-than-or-equal comparison when every predicate reports
// true.
func (b *Builder[C, E]) GTE[T any](
	field any,
	value T,
	predicates ...when.Predicate[T],
) *Builder[C, E] {
	addComparison(b.constructionContext(), OperatorGTE, field, value, predicates)
	return b
}

// In adds a membership test when every predicate reports true. An empty input
// is normalized to false. A one-level pointer element type uses nil elements to
// represent explicit null values; mixed input lowers to In(nonNil) OR IsNull.
func (b *Builder[C, E]) In[T any, S ~[]T](
	field any,
	values S,
	predicates ...when.Predicate[S],
) *Builder[C, E] {
	addMembership(b.constructionContext(), OperatorIn, field, values, predicates)
	return b
}

// NotIn adds a non-null exclusion test when every predicate reports true. An
// empty input is normalized to true. A one-level pointer input is accepted only
// when every element is non-nil; any nil element is an invalid value.
func (b *Builder[C, E]) NotIn[T any, S ~[]T](
	field any,
	values S,
	predicates ...when.Predicate[S],
) *Builder[C, E] {
	addMembership(b.constructionContext(), OperatorNotIn, field, values, predicates)
	return b
}

// Between adds the inclusive numeric range field >= lower AND field <= upper
// when every predicate reports true. An enabled inverted range is invalid, as
// is a range with a floating-point NaN bound.
func (b *Builder[C, E]) Between[T when.Number](
	field any,
	lower T,
	upper T,
	predicates ...when.PairPredicate[T, T],
) *Builder[C, E] {
	addRange(b.constructionContext(), field, lower, upper, predicates)
	return b
}

// IsNull adds an explicit-null test when every enabled value is true.
func (b *Builder[C, E]) IsNull(field any, enabled ...bool) *Builder[C, E] {
	addNull(b.constructionContext(), OperatorIsNull, field, enabled)
	return b
}

// NotNull adds a present, non-null test when every enabled value is true.
func (b *Builder[C, E]) NotNull(field any, enabled ...bool) *Builder[C, E] {
	addNull(b.constructionContext(), OperatorNotNull, field, enabled)
	return b
}

// Contains adds a literal substring test when every predicate reports true.
func (b *Builder[C, E]) Contains(
	field any,
	value string,
	predicates ...when.Predicate[string],
) *Builder[C, E] {
	addText(b.constructionContext(), OperatorContains, field, value, predicates)
	return b
}

// HasPrefix adds a literal prefix test when every predicate reports true.
func (b *Builder[C, E]) HasPrefix(
	field any,
	value string,
	predicates ...when.Predicate[string],
) *Builder[C, E] {
	addText(b.constructionContext(), OperatorHasPrefix, field, value, predicates)
	return b
}

// HasSuffix adds a literal suffix test when every predicate reports true.
func (b *Builder[C, E]) HasSuffix(
	field any,
	value string,
	predicates ...when.Predicate[string],
) *Builder[C, E] {
	addText(b.constructionContext(), OperatorHasSuffix, field, value, predicates)
	return b
}

// AllOf adds a group that requires every child to match. An enabled empty group
// is true. A disabled group is omitted and does not invoke scope.
func (b *Builder[C, E]) AllOf(scope Scope[E], enabled ...bool) *Builder[C, E] {
	addGroup(b.constructionContext(), LogicAllOf, scope, enabled)
	return b
}

// AnyOf adds a group that requires at least one child to match. An enabled
// empty group is false. A disabled group is omitted and does not invoke scope.
func (b *Builder[C, E]) AnyOf(scope Scope[E], enabled ...bool) *Builder[C, E] {
	addGroup(b.constructionContext(), LogicAnyOf, scope, enabled)
	return b
}

// NoneOf adds a group that requires no child to match. An enabled empty group
// is true. A disabled group is omitted and does not invoke scope.
func (b *Builder[C, E]) NoneOf(scope Scope[E], enabled ...bool) *Builder[C, E] {
	addGroup(b.constructionContext(), LogicNoneOf, scope, enabled)
	return b
}

// NotAllOf adds a group that requires at least one child not to match. An
// enabled empty group is false. A disabled group is omitted and does not invoke
// scope.
func (b *Builder[C, E]) NotAllOf(scope Scope[E], enabled ...bool) *Builder[C, E] {
	addGroup(b.constructionContext(), LogicNotAllOf, scope, enabled)
	return b
}

// Native adds an adapter-native final condition directly below the implicit
// root when every enabled value is true. Core treats condition as opaque apart
// from shallow-cloning a top-level slice. The caller and Compiler are
// responsible for the condition's backend validity and safety.
func (b *Builder[C, E]) Native(condition C, enabled ...bool) *Builder[C, E] {
	addNativeCondition(b.constructionContext(), condition, enabled)
	return b
}

// Expr adds an opaque adapter-native expression when every enabled value is
// true. Core does not inspect, clone, or validate expression. The caller must
// keep borrowed expression state immutable while a snapshot can use it.
func (b *Builder[C, E]) Expr(expression E, enabled ...bool) *Builder[C, E] {
	addNativeExpression(b.constructionContext(), expression, enabled)
	return b
}
