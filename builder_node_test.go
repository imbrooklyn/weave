package weave

import (
	"reflect"
	"testing"
)

type constructionCondition []string

type constructionExpression struct {
	name string
}

type constructionNumbers []int

func TestPrivateNodeFamilyAndImplicitRoot(t *testing.T) {
	origin := Origin{Sequence: 7, Operator: OperatorEQ}
	tests := []struct {
		name string
		node node
		kind Kind
	}{
		{name: "constant", node: &constantNode{nodeBase: nodeBase{origin: origin}}, kind: KindConstant},
		{name: "comparison", node: &comparisonNode{nodeBase: nodeBase{origin: origin}}, kind: KindComparison},
		{name: "membership", node: &membershipNode{nodeBase: nodeBase{origin: origin}}, kind: KindMembership},
		{name: "range", node: &rangeNode{nodeBase: nodeBase{origin: origin}}, kind: KindRange},
		{name: "null", node: &nullNode{nodeBase: nodeBase{origin: origin}}, kind: KindNull},
		{name: "text", node: &textNode{nodeBase: nodeBase{origin: origin}}, kind: KindText},
		{name: "group", node: &groupNode{nodeBase: nodeBase{origin: origin}}, kind: KindGroup},
		{
			name: "native condition",
			node: &nativeConditionNode[constructionCondition]{
				nodeBase: nodeBase{origin: origin},
			},
			kind: KindNativeCondition,
		},
		{
			name: "native expression",
			node: &nativeExpressionNode[constructionExpression]{
				nodeBase: nodeBase{origin: origin},
			},
			kind: KindNativeExpression,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.node.nodeKind(); got != test.kind {
				t.Fatalf("nodeKind() = %v, want %v", got, test.kind)
			}
			if got := test.node.nodeOrigin(); got != origin {
				t.Fatalf("nodeOrigin() = %#v, want %#v", got, origin)
			}
		})
	}

	builder := newConstructionBuilder()
	root := builder.state.root
	if root.nodeKind() != KindGroup {
		t.Fatalf("root kind = %v, want %v", root.nodeKind(), KindGroup)
	}
	if root.logic != LogicAllOf {
		t.Fatalf("root logic = %v, want %v", root.logic, LogicAllOf)
	}
	if root.nodeOrigin() != (Origin{}) {
		t.Fatalf("root origin = %#v, want zero", root.nodeOrigin())
	}
	if len(root.children) != 0 {
		t.Fatalf("root child count = %d, want 0", len(root.children))
	}
}

func TestBuilderConstructsEveryComparisonOperator(t *testing.T) {
	tests := []struct {
		name     string
		operator Operator
		add      func(*Builder[constructionCondition, constructionExpression])
	}{
		{name: "EQ", operator: OperatorEQ, add: func(builder *Builder[constructionCondition, constructionExpression]) { builder.EQ("field", 11) }},
		{name: "NEQ", operator: OperatorNEQ, add: func(builder *Builder[constructionCondition, constructionExpression]) { builder.NEQ("field", 11) }},
		{name: "LT", operator: OperatorLT, add: func(builder *Builder[constructionCondition, constructionExpression]) { builder.LT("field", 11) }},
		{name: "LTE", operator: OperatorLTE, add: func(builder *Builder[constructionCondition, constructionExpression]) { builder.LTE("field", 11) }},
		{name: "GT", operator: OperatorGT, add: func(builder *Builder[constructionCondition, constructionExpression]) { builder.GT("field", 11) }},
		{name: "GTE", operator: OperatorGTE, add: func(builder *Builder[constructionCondition, constructionExpression]) { builder.GTE("field", 11) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			builder := newConstructionBuilder()
			test.add(builder)

			node := requireSingleRootChild[*comparisonNode](t, builder)
			if node.operator != test.operator {
				t.Fatalf("operator = %v, want %v", node.operator, test.operator)
			}
			if node.field != "field" || node.value != 11 {
				t.Fatalf("payload = (%v, %v), want (field, 11)", node.field, node.value)
			}
			if node.valueType != reflect.TypeFor[int]() {
				t.Fatalf("value type = %v, want int", node.valueType)
			}
			wantOrigin := Origin{Sequence: 1, Operator: test.operator}
			if node.nodeOrigin() != wantOrigin {
				t.Fatalf("origin = %#v, want %#v", node.nodeOrigin(), wantOrigin)
			}
			requireNoConstructionErrors(t, builder)
		})
	}
}

func TestBuilderConstructsMembershipRangeNullAndTextNodes(t *testing.T) {
	builder := newConstructionBuilder()
	values := constructionNumbers{1, 2, 2}
	builder.
		In("in", values).
		NotIn("not-in", values).
		Between("between", 3, 9).
		IsNull("is-null").
		NotNull("not-null").
		Contains("contains", "middle").
		HasPrefix("prefix", "start").
		HasSuffix("suffix", "end")

	children := builder.state.root.children
	if len(children) != 8 {
		t.Fatalf("root child count = %d, want 8", len(children))
	}

	for index, operator := range []Operator{OperatorIn, OperatorNotIn} {
		node, ok := children[index].(*membershipNode)
		if !ok {
			t.Fatalf("child %d type = %T, want *membershipNode", index, children[index])
		}
		if node.operator != operator {
			t.Fatalf("child %d operator = %v, want %v", index, node.operator, operator)
		}
		if !reflect.DeepEqual(node.values, []any{1, 2, 2}) {
			t.Fatalf("child %d values = %#v, want [1 2 2]", index, node.values)
		}
		if node.inputSliceType != reflect.TypeFor[constructionNumbers]() ||
			node.inputElementType != reflect.TypeFor[int]() ||
			node.elementType != reflect.TypeFor[int]() {
			t.Fatalf(
				"child %d membership types = (%v, %v, %v)",
				index,
				node.inputSliceType,
				node.inputElementType,
				node.elementType,
			)
		}
	}

	rangeValue, ok := children[2].(*rangeNode)
	if !ok {
		t.Fatalf("child 2 type = %T, want *rangeNode", children[2])
	}
	if rangeValue.operator != OperatorBetween ||
		rangeValue.lower != 3 ||
		rangeValue.upper != 9 ||
		rangeValue.boundType != reflect.TypeFor[int]() {
		t.Fatalf("unexpected range payload: %#v", rangeValue)
	}

	for index, operator := range []Operator{OperatorIsNull, OperatorNotNull} {
		node, ok := children[index+3].(*nullNode)
		if !ok || node.operator != operator {
			t.Fatalf("child %d = %#v, want null operator %v", index+3, children[index+3], operator)
		}
	}

	for index, expected := range []struct {
		operator Operator
		value    string
	}{
		{operator: OperatorContains, value: "middle"},
		{operator: OperatorHasPrefix, value: "start"},
		{operator: OperatorHasSuffix, value: "end"},
	} {
		node, ok := children[index+5].(*textNode)
		if !ok || node.operator != expected.operator || node.value != expected.value {
			t.Fatalf("child %d = %#v, want %#v", index+5, children[index+5], expected)
		}
	}
	requireNoConstructionErrors(t, builder)
}

func TestBuilderConstructsFourExplicitGroupsNativeAndExpr(t *testing.T) {
	builder := newConstructionBuilder()
	expression := constructionExpression{name: "root expression"}
	builder.
		AllOf(func(group *Group[constructionExpression]) { group.EQ("field", 1) }).
		AnyOf(func(group *Group[constructionExpression]) { group.EQ("field", 2) }).
		NoneOf(func(group *Group[constructionExpression]) { group.EQ("field", 3) }).
		NotAllOf(func(group *Group[constructionExpression]) { group.EQ("field", 4) }).
		Native(constructionCondition{"native-a", "native-b"}).
		Expr(expression)

	children := builder.state.root.children
	if len(children) != 6 {
		t.Fatalf("root child count = %d, want 6", len(children))
	}
	for index, logic := range []Logic{LogicAllOf, LogicAnyOf, LogicNoneOf, LogicNotAllOf} {
		group, ok := children[index].(*groupNode)
		if !ok || group.logic != logic || len(group.children) != 1 {
			t.Fatalf("child %d = %#v, want one-child %v group", index, children[index], logic)
		}
	}

	native, ok := children[4].(*nativeConditionNode[constructionCondition])
	if !ok || !reflect.DeepEqual(native.condition, constructionCondition{"native-a", "native-b"}) {
		t.Fatalf("native child = %#v", children[4])
	}
	expr, ok := children[5].(*nativeExpressionNode[constructionExpression])
	if !ok || expr.expression != expression {
		t.Fatalf("expression child = %#v", children[5])
	}
	requireNoConstructionErrors(t, builder)
}

func TestGroupConstructsEverySupportedNodeFamily(t *testing.T) {
	builder := newConstructionBuilder()
	builder.AllOf(func(group *Group[constructionExpression]) {
		group.
			EQ("field", 1).
			NEQ("field", 2).
			LT("field", 3).
			LTE("field", 4).
			GT("field", 5).
			GTE("field", 6).
			In("field", constructionNumbers{1, 2}).
			NotIn("field", constructionNumbers{3, 4}).
			Between("field", 1, 4).
			IsNull("field").
			NotNull("field").
			Contains("field", "contains").
			HasPrefix("field", "prefix").
			HasSuffix("field", "suffix").
			AllOf(func(*Group[constructionExpression]) {}).
			AnyOf(func(*Group[constructionExpression]) {}).
			NoneOf(func(*Group[constructionExpression]) {}).
			NotAllOf(func(*Group[constructionExpression]) {}).
			Expr(constructionExpression{name: "nested"})
	})

	outer := requireSingleRootChild[*groupNode](t, builder)
	if len(outer.children) != 19 {
		t.Fatalf("group child count = %d, want 19", len(outer.children))
	}
	for index, operator := range []Operator{
		OperatorEQ,
		OperatorNEQ,
		OperatorLT,
		OperatorLTE,
		OperatorGT,
		OperatorGTE,
	} {
		node, ok := outer.children[index].(*comparisonNode)
		if !ok || node.operator != operator {
			t.Fatalf("group child %d = %#v, want comparison %v", index, outer.children[index], operator)
		}
	}
	if _, ok := outer.children[6].(*membershipNode); !ok {
		t.Fatalf("group child 6 type = %T, want *membershipNode", outer.children[6])
	}
	if _, ok := outer.children[7].(*membershipNode); !ok {
		t.Fatalf("group child 7 type = %T, want *membershipNode", outer.children[7])
	}
	if _, ok := outer.children[8].(*rangeNode); !ok {
		t.Fatalf("group child 8 type = %T, want *rangeNode", outer.children[8])
	}
	for _, index := range []int{9, 10} {
		if _, ok := outer.children[index].(*nullNode); !ok {
			t.Fatalf("group child %d type = %T, want *nullNode", index, outer.children[index])
		}
	}
	for _, index := range []int{11, 12, 13} {
		if _, ok := outer.children[index].(*textNode); !ok {
			t.Fatalf("group child %d type = %T, want *textNode", index, outer.children[index])
		}
	}
	for offset, logic := range []Logic{LogicAllOf, LogicAnyOf, LogicNoneOf, LogicNotAllOf} {
		group, ok := outer.children[offset+14].(*groupNode)
		if !ok || group.logic != logic {
			t.Fatalf("group child %d = %#v, want %v group", offset+14, outer.children[offset+14], logic)
		}
	}
	if _, ok := outer.children[18].(*nativeExpressionNode[constructionExpression]); !ok {
		t.Fatalf("group child 18 type = %T, want native expression", outer.children[18])
	}
	requireNoConstructionErrors(t, builder)
}

func newConstructionBuilder() *Builder[constructionCondition, constructionExpression] {
	return newBuilder[constructionCondition, constructionExpression]()
}

func requireSingleRootChild[T node](
	t *testing.T,
	builder *Builder[constructionCondition, constructionExpression],
) T {
	t.Helper()
	children := builder.state.root.children
	if len(children) != 1 {
		t.Fatalf("root child count = %d, want 1", len(children))
	}
	value, ok := children[0].(T)
	if !ok {
		t.Fatalf("root child type = %T", children[0])
	}
	return value
}

func requireNoConstructionErrors(
	t *testing.T,
	builder *Builder[constructionCondition, constructionExpression],
) {
	t.Helper()
	if len(builder.state.errors) != 0 {
		t.Fatalf("construction errors = %#v, want none", builder.state.errors)
	}
}
