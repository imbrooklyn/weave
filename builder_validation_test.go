package weave

import (
	"errors"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestNilLikeFieldAndValueErrorsAreRecordedInDetectionOrder(t *testing.T) {
	builder := newConstructionBuilder()
	var field *struct{}
	var value *int
	builder.EQ(field, value)

	if len(builder.state.root.children) != 0 {
		t.Fatalf("root child count = %d, want 0", len(builder.state.root.children))
	}
	if len(builder.state.errors) != 2 {
		t.Fatalf("construction error count = %d, want 2", len(builder.state.errors))
	}
	fieldError, valueError := builder.state.errors[0], builder.state.errors[1]
	if !errors.Is(fieldError, ErrInvalidField) || !errors.Is(valueError, ErrInvalidValue) {
		t.Fatalf("errors = (%v, %v), want invalid field then invalid value", fieldError, valueError)
	}
	wantOrigin := Origin{Sequence: 1, Operator: OperatorEQ}
	if fieldError.Origin != wantOrigin || valueError.Origin != wantOrigin {
		t.Fatalf("error origins = (%#v, %#v), want %#v", fieldError.Origin, valueError.Origin, wantOrigin)
	}
	if fieldError.FieldType != reflect.TypeFor[*struct{}]() ||
		valueError.ValueType != reflect.TypeFor[*int]() {
		t.Fatalf(
			"error types = (%v, %v), want (*struct {}, *int)",
			fieldError.FieldType,
			valueError.ValueType,
		)
	}
}

func TestEveryFieldBearingFamilyRejectsNilLikeFields(t *testing.T) {
	tests := []struct {
		name string
		add  func(*Builder[constructionCondition, constructionExpression])
	}{
		{name: "comparison", add: func(builder *Builder[constructionCondition, constructionExpression]) { builder.GT((*int)(nil), 1) }},
		{name: "membership", add: func(builder *Builder[constructionCondition, constructionExpression]) {
			builder.In((*int)(nil), []int{1})
		}},
		{name: "range", add: func(builder *Builder[constructionCondition, constructionExpression]) {
			builder.Between((*int)(nil), 1, 2)
		}},
		{name: "null", add: func(builder *Builder[constructionCondition, constructionExpression]) { builder.IsNull((*int)(nil)) }},
		{name: "text", add: func(builder *Builder[constructionCondition, constructionExpression]) {
			builder.Contains((*int)(nil), "text")
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			builder := newConstructionBuilder()
			test.add(builder)
			if len(builder.state.root.children) != 0 {
				t.Fatalf("root child count = %d, want 0", len(builder.state.root.children))
			}
			if len(builder.state.errors) != 1 || !errors.Is(builder.state.errors[0], ErrInvalidField) {
				t.Fatalf("construction errors = %#v, want ErrInvalidField", builder.state.errors)
			}
		})
	}
}

func TestBetweenValidatesInvertedAndNaNBounds(t *testing.T) {
	tests := []struct {
		name string
		add  func(*Builder[constructionCondition, constructionExpression])
		want error
	}{
		{
			name: "inverted integers",
			add:  func(builder *Builder[constructionCondition, constructionExpression]) { builder.Between("field", 3, 2) },
			want: ErrInvalidRange,
		},
		{
			name: "float32 lower NaN",
			add: func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.Between("field", float32(math.NaN()), float32(2))
			},
			want: ErrInvalidValue,
		},
		{
			name: "float64 upper NaN",
			add: func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.Between("field", 1.0, math.NaN())
			},
			want: ErrInvalidValue,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			builder := newConstructionBuilder()
			test.add(builder)
			if len(builder.state.root.children) != 0 {
				t.Fatalf("root child count = %d, want 0", len(builder.state.root.children))
			}
			if len(builder.state.errors) != 1 || !errors.Is(builder.state.errors[0], test.want) {
				t.Fatalf("construction errors = %#v, want %v", builder.state.errors, test.want)
			}
		})
	}

	builder := newConstructionBuilder()
	builder.Between("field", 2, 2)
	rangeValue := requireSingleRootChild[*rangeNode](t, builder)
	if rangeValue.lower != 2 || rangeValue.upper != 2 {
		t.Fatalf("equal range = (%v, %v), want (2, 2)", rangeValue.lower, rangeValue.upper)
	}
	requireNoConstructionErrors(t, builder)
}

func TestNullableMembershipFormsAndErrors(t *testing.T) {
	t.Run("all nil In lowers to IsNull", func(t *testing.T) {
		builder := newConstructionBuilder()
		builder.In("field", []*int{nil, nil})
		node := requireSingleRootChild[*nullNode](t, builder)
		if node.operator != OperatorIsNull ||
			node.nodeOrigin() != (Origin{Sequence: 1, Operator: OperatorIn}) {
			t.Fatalf("lowered node = %#v", node)
		}
		requireNoConstructionErrors(t, builder)
	})

	t.Run("non-null pointer NotIn dereferences values", func(t *testing.T) {
		builder := newConstructionBuilder()
		one, two := 1, 2
		builder.NotIn("field", []*int{&one, &two})
		node := requireSingleRootChild[*membershipNode](t, builder)
		if !reflect.DeepEqual(node.values, []any{1, 2}) || node.elementType != reflect.TypeFor[int]() {
			t.Fatalf("normalized membership = %#v", node)
		}
		requireNoConstructionErrors(t, builder)
	})

	tests := []struct {
		name string
		add  func(*Builder[constructionCondition, constructionExpression])
	}{
		{
			name: "NotIn contains nil",
			add: func(builder *Builder[constructionCondition, constructionExpression]) {
				one := 1
				builder.NotIn("field", []*int{&one, nil})
			},
		},
		{
			name: "nested pointer element type",
			add: func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.In("field", []**int{})
			},
		},
		{
			name: "nil interface element",
			add: func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.In("field", []any{nil})
			},
		},
		{
			name: "typed nil in interface element",
			add: func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.In("field", []any{(*int)(nil)})
			},
		},
		{
			name: "nil map element",
			add: func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.In("field", []map[string]int{nil})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			builder := newConstructionBuilder()
			test.add(builder)
			if len(builder.state.root.children) != 0 {
				t.Fatalf("root child count = %d, want 0", len(builder.state.root.children))
			}
			if len(builder.state.errors) != 1 || !errors.Is(builder.state.errors[0], ErrInvalidValue) {
				t.Fatalf("construction errors = %#v, want ErrInvalidValue", builder.state.errors)
			}
		})
	}
}

func TestCoreDoesNotPrejudgeAdapterCompatibilityOrOpaquePayloadValidity(t *testing.T) {
	builder := newConstructionBuilder()
	builder.EQ(struct{ backendField string }{backendField: "field"}, struct{ value int }{value: 1})
	builder.LT("backend-specific-field", time.Unix(1, 0))
	builder.In("backend-specific-field", []map[string]int{{"value": 1}})
	builder.Between("backend-text-field", 1, 2)

	if len(builder.state.root.children) != 4 {
		t.Fatalf("root child count = %d, want 4", len(builder.state.root.children))
	}
	requireNoConstructionErrors(t, builder)

	opaque := newBuilder[*int, *int]()
	opaque.Native(nil).Expr(nil)
	if len(opaque.state.root.children) != 2 {
		t.Fatalf("opaque root child count = %d, want 2", len(opaque.state.root.children))
	}
	if len(opaque.state.errors) != 0 {
		t.Fatalf("opaque payload construction errors = %#v, want none", opaque.state.errors)
	}
}

func TestConstructionClonesOwnedTopLevelSlicesOnly(t *testing.T) {
	builder := newBuilder[[]int, []int]()
	comparisonValue := []byte{1, 2}
	membershipValues := []int{3, 4}
	nativeValue := []int{5, 6}
	expressionValue := []int{7, 8}

	builder.EQ("bytes", comparisonValue)
	builder.In("members", membershipValues)
	builder.Native(nativeValue)
	builder.Expr(expressionValue)

	comparisonValue[0] = 11
	membershipValues[0] = 13
	nativeValue[0] = 15
	expressionValue[0] = 17

	children := builder.state.root.children
	comparison := children[0].(*comparisonNode)
	if got := comparison.value.([]byte)[0]; got != 1 {
		t.Fatalf("cloned byte value[0] = %d, want 1", got)
	}
	membership := children[1].(*membershipNode)
	if got := membership.values[0].(int); got != 3 {
		t.Fatalf("cloned membership value[0] = %d, want 3", got)
	}
	native := children[2].(*nativeConditionNode[[]int])
	if got := native.condition[0]; got != 5 {
		t.Fatalf("cloned native value[0] = %d, want 5", got)
	}
	expression := children[3].(*nativeExpressionNode[[]int])
	if got := expression.expression[0]; got != 17 {
		t.Fatalf("borrowed expression value[0] = %d, want 17", got)
	}
	if len(builder.state.errors) != 0 {
		t.Fatalf("construction errors = %#v, want none", builder.state.errors)
	}
}

func TestConstructionDepthLimitAndSequenceExhaustion(t *testing.T) {
	t.Run("depth 128 succeeds", func(t *testing.T) {
		builder := newConstructionBuilder()
		builder.AllOf(nestedConstructionScope(MaxPredicateDepth - 2))
		requireNoConstructionErrors(t, builder)
	})

	t.Run("depth 129 fails", func(t *testing.T) {
		builder := newConstructionBuilder()
		builder.AllOf(nestedConstructionScope(MaxPredicateDepth - 1))
		if len(builder.state.errors) != 1 || !errors.Is(builder.state.errors[0], ErrDepthLimit) {
			t.Fatalf("construction errors = %#v, want ErrDepthLimit", builder.state.errors)
		}
	})

	t.Run("nullable lowering respects depth", func(t *testing.T) {
		builder := newConstructionBuilder()
		builder.AllOf(nestedNullableConstructionScope(MaxPredicateDepth - 2))
		if len(builder.state.errors) != 1 || !errors.Is(builder.state.errors[0], ErrDepthLimit) {
			t.Fatalf("construction errors = %#v, want ErrDepthLimit", builder.state.errors)
		}
	})

	t.Run("sequence does not wrap", func(t *testing.T) {
		builder := newConstructionBuilder()
		builder.state.sequence = math.MaxUint64
		builder.EQ("field", 1)
		if builder.state.sequence != math.MaxUint64 {
			t.Fatalf("sequence = %d, want MaxUint64", builder.state.sequence)
		}
		if len(builder.state.root.children) != 0 {
			t.Fatalf("root child count = %d, want 0", len(builder.state.root.children))
		}
		if len(builder.state.errors) != 1 || !errors.Is(builder.state.errors[0], ErrInvalidState) {
			t.Fatalf("construction errors = %#v, want ErrInvalidState", builder.state.errors)
		}
	})
}

func nestedConstructionScope(groupsBelow int) Scope[constructionExpression] {
	return func(group *Group[constructionExpression]) {
		if groupsBelow == 0 {
			group.EQ("field", 1)
			return
		}
		group.AllOf(nestedConstructionScope(groupsBelow - 1))
	}
}

func nestedNullableConstructionScope(groupsBelow int) Scope[constructionExpression] {
	return func(group *Group[constructionExpression]) {
		if groupsBelow == 0 {
			one := 1
			group.In("field", []*int{&one, nil})
			return
		}
		group.AllOf(nestedNullableConstructionScope(groupsBelow - 1))
	}
}
