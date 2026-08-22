package weave

import (
	"reflect"
	"testing"
)

func TestNodeViewsExposeEveryNodeFamily(t *testing.T) {
	builder := newConstructionBuilder()
	builder.state.root.children = append(
		builder.state.root.children,
		&constantNode{
			nodeBase: nodeBase{origin: Origin{Sequence: 1}},
			value:    false,
		},
	)
	builder.state.sequence = 1
	builder.
		EQ("comparison", 42).
		In("membership", constructionNumbers{2, 3, 3}).
		Between("range", 4, 9).
		IsNull("null").
		Contains("text", "needle").
		AnyOf(func(group *Group[constructionExpression]) {
			group.HasSuffix("nested-text", "suffix")
			group.Expr(constructionExpression{name: "nested expression"})
		}).
		Native(constructionCondition{"native"}).
		Expr(constructionExpression{name: "root expression"})

	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v, want nil", err)
	}
	root := predicate.Root()
	rootGroup, ok := root.AsGroup()
	if !ok || rootGroup.ChildCount() != 9 {
		t.Fatalf("root child count = %d, want 9", rootGroup.ChildCount())
	}
	if _, ok := root.AsComparison(); ok {
		t.Fatal("root AsComparison() succeeded")
	}

	constantNodeView := requireViewChild(t, rootGroup, 0, KindConstant)
	constant, ok := constantNodeView.AsConstant()
	if !ok || constant.Value() {
		t.Fatalf("constant view = %#v, want false", constant)
	}
	if constantNodeView.Origin() != (Origin{Sequence: 1}) {
		t.Fatalf("constant origin = %#v, want sequence 1", constantNodeView.Origin())
	}

	comparisonNodeView := requireViewChild(t, rootGroup, 1, KindComparison)
	comparison, ok := comparisonNodeView.AsComparison()
	if !ok ||
		comparison.Operator() != OperatorEQ ||
		comparison.Field() != "comparison" ||
		comparison.Value() != 42 ||
		comparison.ValueType() != reflect.TypeFor[int]() {
		t.Fatalf("comparison view = %#v", comparison)
	}
	if _, ok := comparisonNodeView.AsConstant(); ok {
		t.Fatal("comparison AsConstant() succeeded")
	}

	membershipNodeView := requireViewChild(t, rootGroup, 2, KindMembership)
	membership, ok := membershipNodeView.AsMembership()
	if !ok ||
		membership.Operator() != OperatorIn ||
		membership.Field() != "membership" ||
		membership.ValueCount() != 3 ||
		membership.InputSliceType() != reflect.TypeFor[constructionNumbers]() ||
		membership.InputElementType() != reflect.TypeFor[int]() ||
		membership.ElementType() != reflect.TypeFor[int]() {
		t.Fatalf("membership view = %#v", membership)
	}
	for index, want := range []int{2, 3, 3} {
		got, ok := membership.Value(index)
		if !ok || got != want {
			t.Fatalf("membership Value(%d) = (%v, %t), want (%d, true)", index, got, ok, want)
		}
	}
	if value, ok := membership.Value(-1); ok || value != nil {
		t.Fatalf("membership Value(-1) = (%v, %t), want (nil, false)", value, ok)
	}
	if value, ok := membership.Value(3); ok || value != nil {
		t.Fatalf("membership Value(3) = (%v, %t), want (nil, false)", value, ok)
	}

	rangeNodeView := requireViewChild(t, rootGroup, 3, KindRange)
	rangeView, ok := rangeNodeView.AsRange()
	if !ok ||
		rangeView.Operator() != OperatorBetween ||
		rangeView.Field() != "range" ||
		rangeView.Lower() != 4 ||
		rangeView.Upper() != 9 ||
		rangeView.BoundType() != reflect.TypeFor[int]() {
		t.Fatalf("range view = %#v", rangeView)
	}

	nullNodeView := requireViewChild(t, rootGroup, 4, KindNull)
	nullView, ok := nullNodeView.AsNull()
	if !ok || nullView.Operator() != OperatorIsNull || nullView.Field() != "null" {
		t.Fatalf("null view = %#v", nullView)
	}

	textNodeView := requireViewChild(t, rootGroup, 5, KindText)
	textView, ok := textNodeView.AsText()
	if !ok ||
		textView.Operator() != OperatorContains ||
		textView.Field() != "text" ||
		textView.Value() != "needle" {
		t.Fatalf("text view = %#v", textView)
	}

	groupNodeView := requireViewChild(t, rootGroup, 6, KindGroup)
	groupView, ok := groupNodeView.AsGroup()
	if !ok || groupView.Logic() != LogicAnyOf || groupView.ChildCount() != 2 {
		t.Fatalf("group view = %#v", groupView)
	}
	nestedTextNode := requireViewChild(t, groupView, 0, KindText)
	nestedText, _ := nestedTextNode.AsText()
	if nestedText.Operator() != OperatorHasSuffix || nestedText.Value() != "suffix" {
		t.Fatalf("nested text view = %#v", nestedText)
	}
	nestedExpressionNode := requireViewChild(t, groupView, 1, KindNativeExpression)
	nestedExpression, ok := nestedExpressionNode.AsNativeExpression()
	if !ok || nestedExpression.Expression() != (constructionExpression{name: "nested expression"}) {
		t.Fatalf("nested expression view = %#v", nestedExpression)
	}

	nativeNodeView := requireViewChild(t, rootGroup, 7, KindNativeCondition)
	native, ok := nativeNodeView.AsNativeCondition()
	if !ok || !reflect.DeepEqual(native.Condition(), constructionCondition{"native"}) {
		t.Fatalf("native condition view = %#v", native)
	}

	expressionNodeView := requireViewChild(t, rootGroup, 8, KindNativeExpression)
	expression, ok := expressionNodeView.AsNativeExpression()
	if !ok || expression.Expression() != (constructionExpression{name: "root expression"}) {
		t.Fatalf("native expression view = %#v", expression)
	}

	wantPaths := []string{
		"root.allOf[0].constant",
		"root.allOf[1].eq",
		"root.allOf[2].in",
		"root.allOf[3].between",
		"root.allOf[4].is_null",
		"root.allOf[5].contains",
		"root.allOf[6].anyOf",
		"root.allOf[7].native_condition",
		"root.allOf[8].native_expression",
	}
	for index, want := range wantPaths {
		child, _ := rootGroup.Child(index)
		if got := child.Path().String(); got != want {
			t.Fatalf("child %d path = %q, want %q", index, got, want)
		}
	}
	if got := nestedExpressionNode.Path().String(); got != "root.allOf[6].anyOf[1].native_expression" {
		t.Fatalf("nested expression path = %q", got)
	}

	wantOrigins := []Origin{
		{Sequence: 1},
		{Sequence: 2, Operator: OperatorEQ},
		{Sequence: 3, Operator: OperatorIn},
		{Sequence: 4, Operator: OperatorBetween},
		{Sequence: 5, Operator: OperatorIsNull},
		{Sequence: 6, Operator: OperatorContains},
		{Sequence: 7},
		{Sequence: 10},
		{Sequence: 11},
	}
	for index, want := range wantOrigins {
		child, _ := rootGroup.Child(index)
		if got := child.Origin(); got != want {
			t.Fatalf("child %d origin = %#v, want %#v", index, got, want)
		}
	}
}

func TestNullableMembershipViewsPreserveTypesOriginsAndPaths(t *testing.T) {
	builder := newConstructionBuilder()
	one, two := 1, 2
	builder.In("field", []*int{&one, nil, &two})

	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v, want nil", err)
	}
	root, _ := predicate.Root().AsGroup()
	loweredNode := requireViewChild(t, root, 0, KindGroup)
	lowered, _ := loweredNode.AsGroup()
	if lowered.Logic() != LogicAnyOf || lowered.ChildCount() != 2 {
		t.Fatalf("lowered group = %#v, want two-child AnyOf", lowered)
	}
	wantOrigin := Origin{Sequence: 1, Operator: OperatorIn}
	if loweredNode.Origin() != wantOrigin {
		t.Fatalf("lowered group origin = %#v, want %#v", loweredNode.Origin(), wantOrigin)
	}

	membershipNode := requireViewChild(t, lowered, 0, KindMembership)
	membership, _ := membershipNode.AsMembership()
	if membership.InputSliceType() != reflect.TypeFor[[]*int]() ||
		membership.InputElementType() != reflect.TypeFor[*int]() ||
		membership.ElementType() != reflect.TypeFor[int]() {
		t.Fatalf(
			"nullable membership types = (%v, %v, %v)",
			membership.InputSliceType(),
			membership.InputElementType(),
			membership.ElementType(),
		)
	}
	if membershipNode.Origin() != wantOrigin {
		t.Fatalf("membership origin = %#v, want %#v", membershipNode.Origin(), wantOrigin)
	}
	if got := membershipNode.Path().String(); got != "root.allOf[0].anyOf[0].in" {
		t.Fatalf("membership path = %q", got)
	}

	nullNode := requireViewChild(t, lowered, 1, KindNull)
	nullView, _ := nullNode.AsNull()
	if nullView.Operator() != OperatorIsNull || nullNode.Origin() != wantOrigin {
		t.Fatalf("lowered null view = %#v, origin %#v", nullView, nullNode.Origin())
	}
	if got := nullNode.Path().String(); got != "root.allOf[0].anyOf[1].is_null" {
		t.Fatalf("null path = %q", got)
	}
}

func TestNodeViewFailuresAndZeroTypedAccessorsAreDeterministic(t *testing.T) {
	var invalid NodeView[constructionCondition, constructionExpression]
	if invalid.Valid() ||
		invalid.Kind() != 0 ||
		invalid.Origin() != (Origin{}) ||
		invalid.Path().String() != "" {
		t.Fatalf("zero NodeView accessors are not zero: %#v", invalid)
	}
	if _, ok := invalid.AsConstant(); ok {
		t.Fatal("zero NodeView AsConstant() succeeded")
	}
	if _, ok := invalid.AsComparison(); ok {
		t.Fatal("zero NodeView AsComparison() succeeded")
	}
	if _, ok := invalid.AsMembership(); ok {
		t.Fatal("zero NodeView AsMembership() succeeded")
	}
	if _, ok := invalid.AsRange(); ok {
		t.Fatal("zero NodeView AsRange() succeeded")
	}
	if _, ok := invalid.AsNull(); ok {
		t.Fatal("zero NodeView AsNull() succeeded")
	}
	if _, ok := invalid.AsText(); ok {
		t.Fatal("zero NodeView AsText() succeeded")
	}
	if _, ok := invalid.AsGroup(); ok {
		t.Fatal("zero NodeView AsGroup() succeeded")
	}
	if _, ok := invalid.AsNativeCondition(); ok {
		t.Fatal("zero NodeView AsNativeCondition() succeeded")
	}
	if _, ok := invalid.AsNativeExpression(); ok {
		t.Fatal("zero NodeView AsNativeExpression() succeeded")
	}

	if (ConstantView{}).Value() {
		t.Fatal("zero ConstantView Value() = true")
	}
	comparison := ComparisonView{}
	if comparison.Operator() != 0 ||
		comparison.Field() != nil ||
		comparison.Value() != nil ||
		comparison.ValueType() != nil {
		t.Fatalf("zero ComparisonView accessors are not zero: %#v", comparison)
	}
	membership := MembershipView{}
	if membership.Operator() != 0 ||
		membership.Field() != nil ||
		membership.ValueCount() != 0 ||
		membership.InputSliceType() != nil ||
		membership.InputElementType() != nil ||
		membership.ElementType() != nil {
		t.Fatalf("zero MembershipView accessors are not zero: %#v", membership)
	}
	if value, ok := membership.Value(0); ok || value != nil {
		t.Fatalf("zero MembershipView Value(0) = (%v, %t)", value, ok)
	}
	rangeView := RangeView{}
	if rangeView.Operator() != 0 ||
		rangeView.Field() != nil ||
		rangeView.Lower() != nil ||
		rangeView.Upper() != nil ||
		rangeView.BoundType() != nil {
		t.Fatalf("zero RangeView accessors are not zero: %#v", rangeView)
	}
	nullView := NullView{}
	if nullView.Operator() != 0 || nullView.Field() != nil {
		t.Fatalf("zero NullView accessors are not zero: %#v", nullView)
	}
	textView := TextView{}
	if textView.Operator() != 0 || textView.Field() != nil || textView.Value() != "" {
		t.Fatalf("zero TextView accessors are not zero: %#v", textView)
	}
	group := GroupView[constructionCondition, constructionExpression]{}
	if group.Logic() != 0 || group.ChildCount() != 0 {
		t.Fatalf("zero GroupView accessors are not zero: %#v", group)
	}
	if child, ok := group.Child(0); ok || child.Valid() {
		t.Fatalf("zero GroupView Child(0) = (%#v, %t)", child, ok)
	}
	native := NativeConditionView[constructionCondition]{}
	if native.Condition() != nil {
		t.Fatalf("zero NativeConditionView Condition() = %#v, want nil", native.Condition())
	}
	expression := NativeExpressionView[*constructionExpression]{}
	if expression.Expression() != nil {
		t.Fatalf("zero NativeExpressionView Expression() = %#v, want nil", expression.Expression())
	}
}

func TestGroupChildAndNodePathBoundsDoNotPanic(t *testing.T) {
	builder := newConstructionBuilder()
	builder.AnyOf(func(group *Group[constructionExpression]) {
		group.NoneOf(func(group *Group[constructionExpression]) {
			group.Between("field", 1, 2)
		})
	})
	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v, want nil", err)
	}
	root, _ := predicate.Root().AsGroup()
	for _, index := range []int{-1, 1} {
		child, ok := root.Child(index)
		if ok || child.Valid() {
			t.Fatalf("root Child(%d) = (%#v, %t), want invalid and false", index, child, ok)
		}
	}

	anyNode, _ := root.Child(0)
	anyGroup, _ := anyNode.AsGroup()
	noneNode, _ := anyGroup.Child(0)
	noneGroup, _ := noneNode.AsGroup()
	leaf, _ := noneGroup.Child(0)
	path := leaf.Path()
	if got := path.String(); got != "root.allOf[0].anyOf[0].noneOf[0].between" {
		t.Fatalf("leaf path = %q", got)
	}
	if path.SegmentCount() != 7 {
		t.Fatalf("leaf segment count = %d, want 7", path.SegmentCount())
	}
	for _, index := range []int{-1, path.SegmentCount()} {
		segment, ok := path.Segment(index)
		if ok || segment != (PathSegment{}) {
			t.Fatalf("path Segment(%d) = (%#v, %t), want zero and false", index, segment, ok)
		}
	}

	rootSegment, _ := path.Segment(0)
	if rootSegment.Kind() != PathSegmentRoot ||
		rootSegment.NodeKind() != KindGroup ||
		rootSegment.Logic() != LogicAllOf ||
		rootSegment.Operator() != 0 {
		t.Fatalf("root segment = %#v", rootSegment)
	}
	for offset, wantIndex := range []int{0, 0, 0} {
		childSegment, _ := path.Segment(1 + offset*2)
		index, ok := childSegment.ChildIndex()
		if !ok || index != wantIndex || childSegment.NodeKind() != 0 {
			t.Fatalf("child segment %d = %#v", offset, childSegment)
		}
	}
	leafSegment, _ := path.Segment(6)
	if leafSegment.Kind() != PathSegmentNode ||
		leafSegment.NodeKind() != KindRange ||
		leafSegment.Operator() != OperatorBetween ||
		leafSegment.Logic() != 0 {
		t.Fatalf("leaf segment = %#v", leafSegment)
	}
}

func TestNodePathAccessorsDoNotExposeStoredBacking(t *testing.T) {
	builder := newConstructionBuilder()
	builder.EQ("field", 1)
	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v, want nil", err)
	}
	root, _ := predicate.Root().AsGroup()
	leaf, _ := root.Child(0)

	first := leaf.Path()
	first.segments[0] = PathSegment{}
	first.segments[1].childIndex = 99
	second := leaf.Path()
	if got := second.String(); got != "root.allOf[0].eq" {
		t.Fatalf("stored path after caller mutation = %q", got)
	}
	if first.segments == nil || second.segments == nil || &first.segments[0] == &second.segments[0] {
		t.Fatal("Path() calls shared segment backing storage")
	}
}

func TestNodeViewRejectsNodeOwnedByAnotherPredicate(t *testing.T) {
	firstBuilder := newConstructionBuilder()
	firstBuilder.EQ("first", 1)
	first, err := firstBuilder.Predicate()
	if err != nil {
		t.Fatalf("first Predicate() error = %v", err)
	}
	secondBuilder := newConstructionBuilder()
	secondBuilder.EQ("second", 2)
	second, err := secondBuilder.Predicate()
	if err != nil {
		t.Fatalf("second Predicate() error = %v", err)
	}

	secondRoot, _ := second.Root().AsGroup()
	secondChild, _ := secondRoot.Child(0)
	foreign := NodeView[constructionCondition, constructionExpression]{
		state: first.state,
		node:  secondChild.node,
	}
	if foreign.Valid() || foreign.Kind() != 0 {
		t.Fatalf("foreign NodeView = %#v, want invalid", foreign)
	}
}

func requireViewChild[C, E any](
	t *testing.T,
	group GroupView[C, E],
	index int,
	wantKind Kind,
) NodeView[C, E] {
	t.Helper()
	child, ok := group.Child(index)
	if !ok {
		t.Fatalf("Child(%d) failed within bounds", index)
	}
	if !child.Valid() || child.Kind() != wantKind {
		t.Fatalf("Child(%d) kind = %v, want %v", index, child.Kind(), wantKind)
	}
	return child
}
