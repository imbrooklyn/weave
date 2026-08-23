package weave

import "testing"

func TestPredicateRequirementsIncludeEveryNormalizedLeafExactly(t *testing.T) {
	builder := newConstructionBuilder()
	builder.
		EQ("field", 1).
		NEQ("field", 1).
		LT("field", 1).
		LTE("field", 1).
		GT("field", 1).
		GTE("field", 1).
		In("field", []int{1}).
		NotIn("field", []int{1}).
		Between("field", 1, 2).
		IsNull("field").
		NotNull("field").
		Contains("field", "value").
		HasPrefix("field", "value").
		HasSuffix("field", "value").
		EQ("field", 1).
		Native(constructionCondition{"native"}).
		Expr(constructionExpression{name: "root expression"}).
		AnyOf(func(group *Group[constructionExpression]) {
			group.Expr(constructionExpression{name: "nested expression"})
		})

	predicate := requireNormalizedPredicate(t, builder)
	want := Requirements{
		Operators: NewOperatorSet(
			OperatorEQ,
			OperatorNEQ,
			OperatorLT,
			OperatorLTE,
			OperatorGT,
			OperatorGTE,
			OperatorIn,
			OperatorNotIn,
			OperatorBetween,
			OperatorIsNull,
			OperatorNotNull,
			OperatorContains,
			OperatorHasPrefix,
			OperatorHasSuffix,
		),
		Features: NewFeatureSet(
			FeatureNativeCondition,
			FeatureNativeExpression,
		),
	}
	assertRequirementsEqual(t, predicate.Requirements(), want)
	assertRequirementsEqual(t, requirementsFromNodeViews(t, predicate.Root()), want)
}

func TestPredicateRequirementsIgnoreConstantsAndGroups(t *testing.T) {
	builder := newConstructionBuilder()
	builder.In("field", []int{})
	builder.NotIn("field", []int{})
	builder.AllOf(func(*Group[constructionExpression]) {})
	builder.AnyOf(func(*Group[constructionExpression]) {})
	builder.NoneOf(func(*Group[constructionExpression]) {})
	builder.NotAllOf(func(*Group[constructionExpression]) {})

	predicate := requireNormalizedPredicate(t, builder)
	assertRequirementsEqual(t, predicate.Requirements(), Requirements{})
	assertRequirementsEqual(t, requirementsFromNodeViews(t, predicate.Root()), Requirements{})
}

func TestPredicateRequirementsAreImmutablePerSnapshot(t *testing.T) {
	builder := newConstructionBuilder()
	builder.EQ("field", 1)
	first := requireNormalizedPredicate(t, builder)

	firstWant := Requirements{Operators: NewOperatorSet(OperatorEQ)}
	assertRequirementsEqual(t, first.Requirements(), firstWant)

	builder.GT("field", 2)
	builder.Native(constructionCondition{"native"})
	second := requireNormalizedPredicate(t, builder)
	secondWant := Requirements{
		Operators: NewOperatorSet(OperatorEQ, OperatorGT),
		Features:  NewFeatureSet(FeatureNativeCondition),
	}
	assertRequirementsEqual(t, second.Requirements(), secondWant)
	assertRequirementsEqual(t, first.Requirements(), firstWant)

	returned := first.Requirements()
	returned.Operators = NewOperatorSet(OperatorContains)
	returned.Features = NewFeatureSet(FeatureNativeExpression)
	assertRequirementsEqual(t, first.Requirements(), firstWant)
}

func TestZeroAndInvalidPredicatesHaveZeroRequirements(t *testing.T) {
	var zero Predicate[constructionCondition, constructionExpression]
	assertRequirementsEqual(t, zero.Requirements(), Requirements{})

	state := &predicateState{
		seal:         validPredicateSeal,
		domain:       newPredicateDomain(),
		requirements: Requirements{Operators: NewOperatorSet(OperatorEQ)},
	}
	invalid := Predicate[constructionCondition, constructionExpression]{state: state}
	assertRequirementsEqual(t, invalid.Requirements(), Requirements{})
}

func requirementsFromNodeViews[C, E any](
	t *testing.T,
	root NodeView[C, E],
) Requirements {
	t.Helper()
	var requirements Requirements
	stack := []NodeView[C, E]{root}
	for len(stack) != 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if !current.Valid() {
			t.Fatal("requirements traversal encountered an invalid node")
		}

		if group, ok := current.AsGroup(); ok {
			for index := group.ChildCount() - 1; index >= 0; index-- {
				child, childOK := group.Child(index)
				if !childOK {
					t.Fatalf("requirements Child(%d) failed", index)
				}
				stack = append(stack, child)
			}
			continue
		}
		if comparison, ok := current.AsComparison(); ok {
			addIndependentOperator(&requirements, comparison.Operator())
			continue
		}
		if membership, ok := current.AsMembership(); ok {
			addIndependentOperator(&requirements, membership.Operator())
			continue
		}
		if rangeView, ok := current.AsRange(); ok {
			addIndependentOperator(&requirements, rangeView.Operator())
			continue
		}
		if nullView, ok := current.AsNull(); ok {
			addIndependentOperator(&requirements, nullView.Operator())
			continue
		}
		if textView, ok := current.AsText(); ok {
			addIndependentOperator(&requirements, textView.Operator())
			continue
		}
		if _, ok := current.AsNativeCondition(); ok {
			addIndependentFeature(&requirements, FeatureNativeCondition)
			continue
		}
		if _, ok := current.AsNativeExpression(); ok {
			addIndependentFeature(&requirements, FeatureNativeExpression)
			continue
		}
		if _, ok := current.AsConstant(); !ok {
			t.Fatalf("requirements traversal does not recognize kind %v", current.Kind())
		}
	}
	return requirements
}

func addIndependentOperator(requirements *Requirements, operator Operator) {
	values := make([]Operator, 0, requirements.Operators.Count()+1)
	for index := 0; index < requirements.Operators.Count(); index++ {
		value, _ := requirements.Operators.At(index)
		values = append(values, value)
	}
	values = append(values, operator)
	requirements.Operators = NewOperatorSet(values...)
}

func addIndependentFeature(requirements *Requirements, feature Feature) {
	values := make([]Feature, 0, requirements.Features.Count()+1)
	for index := 0; index < requirements.Features.Count(); index++ {
		value, _ := requirements.Features.At(index)
		values = append(values, value)
	}
	values = append(values, feature)
	requirements.Features = NewFeatureSet(values...)
}
