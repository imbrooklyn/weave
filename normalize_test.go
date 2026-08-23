package weave

import (
	"errors"
	"reflect"
	"testing"
)

type normalizationFieldState uint8

const (
	normalizationValue normalizationFieldState = iota + 1
	normalizationNull
	normalizationMissing
)

type normalizationRecord struct {
	name  string
	state normalizationFieldState
	value int
}

var normalizationRecords = []normalizationRecord{
	{name: "value one", state: normalizationValue, value: 1},
	{name: "value two", state: normalizationValue, value: 2},
	{name: "value three", state: normalizationValue, value: 3},
	{name: "null", state: normalizationNull},
	{name: "missing", state: normalizationMissing},
}

func TestEmptyMembershipNormalizationStructureAndMatchSets(t *testing.T) {
	t.Run("In becomes false", func(t *testing.T) {
		builder := newConstructionBuilder()
		builder.In("field", []int{})
		predicate := requireNormalizedPredicate(t, builder)
		root, _ := predicate.Root().AsGroup()
		node := requireViewChild(t, root, 0, KindConstant)
		constant, _ := node.AsConstant()
		if constant.Value() {
			t.Fatal("empty In normalized to true")
		}
		if node.Origin() != (Origin{Sequence: 1, Operator: OperatorIn}) {
			t.Fatalf("empty In origin = %#v", node.Origin())
		}
		if got := node.Path().String(); got != "root.allOf[0].constant" {
			t.Fatalf("empty In path = %q", got)
		}
		assertNormalizationMatchSet(t, predicate, []bool{false, false, false, false, false})
		assertRequirementsEqual(t, predicate.Requirements(), Requirements{})
	})

	t.Run("NotIn becomes true and root removes its identity", func(t *testing.T) {
		builder := newConstructionBuilder()
		builder.NotIn("field", []int{})
		predicate := requireNormalizedPredicate(t, builder)
		root, _ := predicate.Root().AsGroup()
		if root.ChildCount() != 0 {
			t.Fatalf("empty NotIn root child count = %d, want 0", root.ChildCount())
		}
		assertNormalizationMatchSet(t, predicate, []bool{true, true, true, true, true})
		assertRequirementsEqual(t, predicate.Requirements(), Requirements{})
	})

	t.Run("NotIn true remains visible when it is an annihilator", func(t *testing.T) {
		builder := newConstructionBuilder()
		builder.AnyOf(func(group *Group[constructionExpression]) {
			group.NotIn("field", []int{})
			group.EQ("field", 1)
		})
		predicate := requireNormalizedPredicate(t, builder)
		root, _ := predicate.Root().AsGroup()
		groupNode := requireViewChild(t, root, 0, KindGroup)
		group, _ := groupNode.AsGroup()
		if group.Logic() != LogicAnyOf || group.ChildCount() != 2 {
			t.Fatalf("AnyOf shape = %v with %d children", group.Logic(), group.ChildCount())
		}
		constantNode := requireViewChild(t, group, 0, KindConstant)
		constant, _ := constantNode.AsConstant()
		if !constant.Value() ||
			constantNode.Origin() != (Origin{Sequence: 2, Operator: OperatorNotIn}) {
			t.Fatalf("empty NotIn constant = %#v", constantNode)
		}
		requireViewChild(t, group, 1, KindComparison)
		assertNormalizationMatchSet(t, predicate, []bool{true, true, true, true, true})
		assertRequirementsEqual(t, predicate.Requirements(), Requirements{
			Operators: NewOperatorSet(OperatorEQ),
		})
	})
}

func TestNullableInNormalizationStructureAndMatchSets(t *testing.T) {
	t.Run("mixed values lower in stable order", func(t *testing.T) {
		one, two := 1, 2
		values := []*int{&two, nil, &one, &two, nil}
		builder := newConstructionBuilder()
		builder.In("field", values)

		predicate := requireNormalizedPredicate(t, builder)
		root, _ := predicate.Root().AsGroup()
		loweredNode := requireViewChild(t, root, 0, KindGroup)
		lowered, _ := loweredNode.AsGroup()
		if lowered.Logic() != LogicAnyOf || lowered.ChildCount() != 2 {
			t.Fatalf("nullable In shape = %v with %d children", lowered.Logic(), lowered.ChildCount())
		}

		wantOrigin := Origin{Sequence: 1, Operator: OperatorIn}
		membershipNode := requireViewChild(t, lowered, 0, KindMembership)
		membership, _ := membershipNode.AsMembership()
		if membershipNode.Origin() != wantOrigin || loweredNode.Origin() != wantOrigin {
			t.Fatalf("nullable In origins = (%#v, %#v)", loweredNode.Origin(), membershipNode.Origin())
		}
		if membership.InputSliceType() != reflect.TypeFor[[]*int]() ||
			membership.InputElementType() != reflect.TypeFor[*int]() ||
			membership.ElementType() != reflect.TypeFor[int]() {
			t.Fatalf("nullable In types = (%v, %v, %v)", membership.InputSliceType(), membership.InputElementType(), membership.ElementType())
		}
		wantValues := []int{2, 1, 2}
		if membership.ValueCount() != len(wantValues) {
			t.Fatalf("nullable In value count = %d, want %d", membership.ValueCount(), len(wantValues))
		}
		for index, want := range wantValues {
			got, ok := membership.Value(index)
			if !ok || got != want {
				t.Fatalf("nullable In value %d = (%#v, %t), want %d", index, got, ok, want)
			}
		}
		nullNode := requireViewChild(t, lowered, 1, KindNull)
		nullView, _ := nullNode.AsNull()
		if nullView.Operator() != OperatorIsNull || nullNode.Origin() != wantOrigin {
			t.Fatalf("nullable In null child = %#v", nullNode)
		}
		if membershipNode.Path().String() != "root.allOf[0].anyOf[0].in" ||
			nullNode.Path().String() != "root.allOf[0].anyOf[1].is_null" {
			t.Fatalf("nullable In paths = (%q, %q)", membershipNode.Path().String(), nullNode.Path().String())
		}

		assertNormalizationMatchSet(t, predicate, []bool{true, true, false, true, false})
		assertRequirementsEqual(t, predicate.Requirements(), Requirements{
			Operators: NewOperatorSet(OperatorIn, OperatorIsNull),
		})
	})

	t.Run("all nil lowers to IsNull", func(t *testing.T) {
		builder := newConstructionBuilder()
		builder.In("field", []*int{nil, nil})
		predicate := requireNormalizedPredicate(t, builder)
		root, _ := predicate.Root().AsGroup()
		node := requireViewChild(t, root, 0, KindNull)
		view, _ := node.AsNull()
		if view.Operator() != OperatorIsNull ||
			node.Origin() != (Origin{Sequence: 1, Operator: OperatorIn}) {
			t.Fatalf("all-nil In node = %#v", node)
		}
		assertNormalizationMatchSet(t, predicate, []bool{false, false, false, true, false})
		assertRequirementsEqual(t, predicate.Requirements(), Requirements{
			Operators: NewOperatorSet(OperatorIsNull),
		})
	})

	t.Run("non-null pointers dereference without reordering", func(t *testing.T) {
		one, two := 1, 2
		builder := newConstructionBuilder()
		builder.NotIn("field", []*int{&two, &one, &two})
		one = 101
		two = 102

		predicate := requireNormalizedPredicate(t, builder)
		root, _ := predicate.Root().AsGroup()
		node := requireViewChild(t, root, 0, KindMembership)
		view, _ := node.AsMembership()
		if node.Origin() != (Origin{Sequence: 1, Operator: OperatorNotIn}) ||
			view.InputSliceType() != reflect.TypeFor[[]*int]() ||
			view.InputElementType() != reflect.TypeFor[*int]() ||
			view.ElementType() != reflect.TypeFor[int]() {
			t.Fatalf("NotIn metadata = origin %#v, types (%v, %v, %v)", node.Origin(), view.InputSliceType(), view.InputElementType(), view.ElementType())
		}
		wantValues := []int{2, 1, 2}
		for index, want := range wantValues {
			got, ok := view.Value(index)
			if !ok || got != want {
				t.Fatalf("NotIn value %d = (%#v, %t), want %d", index, got, ok, want)
			}
		}
		assertNormalizationMatchSet(t, predicate, []bool{false, false, true, false, false})
		assertRequirementsEqual(t, predicate.Requirements(), Requirements{
			Operators: NewOperatorSet(OperatorNotIn),
		})
	})

	t.Run("interface slices never use runtime pointer lowering", func(t *testing.T) {
		one, two := 1, 2
		builder := newConstructionBuilder()
		builder.In("field", []any{&one, &two})
		predicate := requireNormalizedPredicate(t, builder)
		root, _ := predicate.Root().AsGroup()
		node := requireViewChild(t, root, 0, KindMembership)
		view, _ := node.AsMembership()
		if view.InputElementType() != reflect.TypeFor[any]() ||
			view.ElementType() != reflect.TypeFor[any]() ||
			view.ValueCount() != 2 {
			t.Fatalf("[]any membership metadata = (%v, %v, %d)", view.InputElementType(), view.ElementType(), view.ValueCount())
		}
		first, _ := view.Value(0)
		second, _ := view.Value(1)
		if first != &one || second != &two {
			t.Fatalf("[]any values = (%#v, %#v), want original pointers", first, second)
		}
	})

	t.Run("nil-like interface elements never lower", func(t *testing.T) {
		tests := []struct {
			name   string
			values []any
		}{
			{name: "nil interface", values: []any{nil}},
			{name: "typed nil pointer", values: []any{(*int)(nil)}},
			{name: "typed nil map", values: []any{map[string]int(nil)}},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				builder := newConstructionBuilder()
				builder.In("field", test.values)
				predicate, err := builder.Predicate()
				if !errors.Is(err, ErrInvalidValue) {
					t.Fatalf("Predicate() error = %v, want ErrInvalidValue", err)
				}
				if predicate.Root().Valid() {
					t.Fatal("nil-like []any input returned a valid Predicate")
				}
			})
		}
	})

	t.Run("NotIn nil and nested pointers remain construction errors", func(t *testing.T) {
		one := 1
		tests := []func(*Builder[constructionCondition, constructionExpression]){
			func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.NotIn("field", []*int{&one, nil})
			},
			func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.In("field", []**int{})
			},
		}
		for index, add := range tests {
			builder := newConstructionBuilder()
			add(builder)
			_, err := builder.Predicate()
			if !errors.Is(err, ErrInvalidValue) {
				t.Fatalf("case %d error = %v, want ErrInvalidValue", index, err)
			}
			var diagnostic *Error
			if !errors.As(err, &diagnostic) || diagnostic.Phase != PhaseConstruct {
				t.Fatalf("case %d diagnostic = %#v, want construct phase", index, diagnostic)
			}
		}
	})
}

func TestEmptyGroupIdentitiesAndMatchSets(t *testing.T) {
	tests := []struct {
		name      string
		add       func(*Builder[constructionCondition, constructionExpression])
		wantValue bool
	}{
		{name: "AllOf", add: func(builder *Builder[constructionCondition, constructionExpression]) {
			builder.AllOf(func(*Group[constructionExpression]) {})
		}, wantValue: true},
		{name: "AnyOf", add: func(builder *Builder[constructionCondition, constructionExpression]) {
			builder.AnyOf(func(*Group[constructionExpression]) {})
		}},
		{name: "NoneOf", add: func(builder *Builder[constructionCondition, constructionExpression]) {
			builder.NoneOf(func(*Group[constructionExpression]) {})
		}, wantValue: true},
		{name: "NotAllOf", add: func(builder *Builder[constructionCondition, constructionExpression]) {
			builder.NotAllOf(func(*Group[constructionExpression]) {})
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			builder := newConstructionBuilder()
			test.add(builder)
			predicate := requireNormalizedPredicate(t, builder)
			root, _ := predicate.Root().AsGroup()
			if test.wantValue {
				if root.ChildCount() != 0 {
					t.Fatalf("true identity root child count = %d, want 0", root.ChildCount())
				}
			} else {
				node := requireViewChild(t, root, 0, KindConstant)
				constant, _ := node.AsConstant()
				if constant.Value() {
					t.Fatal("false identity normalized to true")
				}
				if node.Origin() != (Origin{Sequence: 1}) {
					t.Fatalf("false identity origin = %#v", node.Origin())
				}
			}
			want := make([]bool, len(normalizationRecords))
			for index := range want {
				want[index] = test.wantValue
			}
			assertNormalizationMatchSet(t, predicate, want)
			assertRequirementsEqual(t, predicate.Requirements(), Requirements{})
		})
	}
}

func TestGroupIdentityAndPureConstantFolding(t *testing.T) {
	t.Run("identity constants are removed without flattening groups", func(t *testing.T) {
		tests := []struct {
			name       string
			logic      Logic
			add        func(*Builder[constructionCondition, constructionExpression])
			wantMatch  []bool
			leafOrigin Origin
		}{
			{name: "AllOf", logic: LogicAllOf, add: func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.AllOf(func(group *Group[constructionExpression]) {
					group.AllOf(func(*Group[constructionExpression]) {})
					group.EQ("field", 1)
				})
			}, wantMatch: []bool{true, false, false, false, false}, leafOrigin: Origin{Sequence: 3, Operator: OperatorEQ}},
			{name: "AnyOf", logic: LogicAnyOf, add: func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.AnyOf(func(group *Group[constructionExpression]) {
					group.In("field", []int{})
					group.EQ("field", 1)
				})
			}, wantMatch: []bool{true, false, false, false, false}, leafOrigin: Origin{Sequence: 3, Operator: OperatorEQ}},
			{name: "NoneOf", logic: LogicNoneOf, add: func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.NoneOf(func(group *Group[constructionExpression]) {
					group.In("field", []int{})
					group.EQ("field", 1)
				})
			}, wantMatch: []bool{false, true, true, true, true}, leafOrigin: Origin{Sequence: 3, Operator: OperatorEQ}},
			{name: "NotAllOf", logic: LogicNotAllOf, add: func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.NotAllOf(func(group *Group[constructionExpression]) {
					group.AllOf(func(*Group[constructionExpression]) {})
					group.EQ("field", 1)
				})
			}, wantMatch: []bool{false, true, true, true, true}, leafOrigin: Origin{Sequence: 3, Operator: OperatorEQ}},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				builder := newConstructionBuilder()
				test.add(builder)
				predicate := requireNormalizedPredicate(t, builder)
				root, _ := predicate.Root().AsGroup()
				groupNode := requireViewChild(t, root, 0, KindGroup)
				group, _ := groupNode.AsGroup()
				if group.Logic() != test.logic || group.ChildCount() != 1 {
					t.Fatalf("normalized group = %v with %d children", group.Logic(), group.ChildCount())
				}
				leaf := requireViewChild(t, group, 0, KindComparison)
				if leaf.Origin() != test.leafOrigin {
					t.Fatalf("retained leaf origin = %#v, want %#v", leaf.Origin(), test.leafOrigin)
				}
				wantPath := "root.allOf[0]." + pathLogicString(test.logic) + "[0].eq"
				if got := leaf.Path().String(); got != wantPath {
					t.Fatalf("retained leaf path = %q, want %q", got, wantPath)
				}
				assertNormalizationMatchSet(t, predicate, test.wantMatch)
				assertRequirementsEqual(t, predicate.Requirements(), Requirements{
					Operators: NewOperatorSet(OperatorEQ),
				})
			})
		}
	})

	t.Run("groups containing only constants fold completely", func(t *testing.T) {
		tests := []struct {
			name      string
			add       func(*Builder[constructionCondition, constructionExpression])
			wantValue bool
		}{
			{name: "AllOf false and true", add: func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.AllOf(func(group *Group[constructionExpression]) {
					group.In("field", []int{})
					group.NotIn("field", []int{})
				})
			}},
			{name: "AnyOf false and true", add: func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.AnyOf(func(group *Group[constructionExpression]) {
					group.In("field", []int{})
					group.NotIn("field", []int{})
				})
			}, wantValue: true},
			{name: "NoneOf false and false", add: func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.NoneOf(func(group *Group[constructionExpression]) {
					group.In("field", []int{})
					group.In("field", []int{})
				})
			}, wantValue: true},
			{name: "NotAllOf true and true", add: func(builder *Builder[constructionCondition, constructionExpression]) {
				builder.NotAllOf(func(group *Group[constructionExpression]) {
					group.NotIn("field", []int{})
					group.NotIn("field", []int{})
				})
			}},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				builder := newConstructionBuilder()
				test.add(builder)
				predicate := requireNormalizedPredicate(t, builder)
				root, _ := predicate.Root().AsGroup()
				if test.wantValue {
					if root.ChildCount() != 0 {
						t.Fatalf("folded true root child count = %d, want 0", root.ChildCount())
					}
				} else {
					node := requireViewChild(t, root, 0, KindConstant)
					constant, _ := node.AsConstant()
					if constant.Value() || node.Origin() != (Origin{Sequence: 1}) {
						t.Fatalf("folded false node = %#v", node)
					}
				}
				want := make([]bool, len(normalizationRecords))
				for index := range want {
					want[index] = test.wantValue
				}
				assertNormalizationMatchSet(t, predicate, want)
				assertRequirementsEqual(t, predicate.Requirements(), Requirements{})
			})
		}
	})
}

func TestAnnihilatorsNeverDiscardNonConstantChildren(t *testing.T) {
	tests := []struct {
		name             string
		logic            Logic
		addConstant      func(*Group[constructionExpression])
		wantConstant     bool
		wantOrigin       Origin
		wantGroupMatches bool
	}{
		{name: "AllOf false", logic: LogicAllOf, addConstant: func(group *Group[constructionExpression]) {
			group.In("field", []int{})
		}, wantOrigin: Origin{Sequence: 2, Operator: OperatorIn}},
		{name: "AnyOf true", logic: LogicAnyOf, addConstant: func(group *Group[constructionExpression]) {
			group.NotIn("field", []int{})
		}, wantConstant: true, wantOrigin: Origin{Sequence: 2, Operator: OperatorNotIn}, wantGroupMatches: true},
		{name: "NoneOf true", logic: LogicNoneOf, addConstant: func(group *Group[constructionExpression]) {
			group.NotIn("field", []int{})
		}, wantConstant: true, wantOrigin: Origin{Sequence: 2, Operator: OperatorNotIn}},
		{name: "NotAllOf false", logic: LogicNotAllOf, addConstant: func(group *Group[constructionExpression]) {
			group.In("field", []int{})
		}, wantOrigin: Origin{Sequence: 2, Operator: OperatorIn}, wantGroupMatches: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			builder := newConstructionBuilder()
			addGroupByLogic(builder, test.logic, func(group *Group[constructionExpression]) {
				test.addConstant(group)
				group.EQ("field", 1)
			})
			predicate := requireNormalizedPredicate(t, builder)
			root, _ := predicate.Root().AsGroup()
			groupNode := requireViewChild(t, root, 0, KindGroup)
			group, _ := groupNode.AsGroup()
			if group.Logic() != test.logic || group.ChildCount() != 2 {
				t.Fatalf("annihilator group = %v with %d children", group.Logic(), group.ChildCount())
			}
			constantNode := requireViewChild(t, group, 0, KindConstant)
			constant, _ := constantNode.AsConstant()
			if constant.Value() != test.wantConstant || constantNode.Origin() != test.wantOrigin {
				t.Fatalf("annihilator constant = %t, origin %#v", constant.Value(), constantNode.Origin())
			}
			leaf := requireViewChild(t, group, 1, KindComparison)
			if leaf.Origin() != (Origin{Sequence: 3, Operator: OperatorEQ}) {
				t.Fatalf("non-constant leaf origin = %#v", leaf.Origin())
			}
			want := make([]bool, len(normalizationRecords))
			for index := range want {
				want[index] = test.wantGroupMatches
			}
			assertNormalizationMatchSet(t, predicate, want)
			assertRequirementsEqual(t, predicate.Requirements(), Requirements{
				Operators: NewOperatorSet(OperatorEQ),
			})
		})
	}
}

func TestNormalizationDoesNotRewriteNEQAsNoneOfEQ(t *testing.T) {
	neqBuilder := newConstructionBuilder()
	neqBuilder.NEQ("field", 1)
	neq := requireNormalizedPredicate(t, neqBuilder)
	neqRoot, _ := neq.Root().AsGroup()
	neqNode := requireViewChild(t, neqRoot, 0, KindComparison)
	neqView, _ := neqNode.AsComparison()
	if neqView.Operator() != OperatorNEQ {
		t.Fatalf("NEQ operator = %v", neqView.Operator())
	}

	complementBuilder := newConstructionBuilder()
	complementBuilder.NoneOf(func(group *Group[constructionExpression]) {
		group.EQ("field", 1)
	})
	complement := requireNormalizedPredicate(t, complementBuilder)
	complementRoot, _ := complement.Root().AsGroup()
	groupNode := requireViewChild(t, complementRoot, 0, KindGroup)
	group, _ := groupNode.AsGroup()
	if group.Logic() != LogicNoneOf || group.ChildCount() != 1 {
		t.Fatalf("NoneOf(EQ) shape = %v with %d children", group.Logic(), group.ChildCount())
	}
	eqNode := requireViewChild(t, group, 0, KindComparison)
	eq, _ := eqNode.AsComparison()
	if eq.Operator() != OperatorEQ {
		t.Fatalf("NoneOf child operator = %v, want EQ", eq.Operator())
	}

	assertNormalizationMatchSet(t, neq, []bool{false, true, true, false, false})
	assertNormalizationMatchSet(t, complement, []bool{false, true, true, true, true})
	assertRequirementsEqual(t, neq.Requirements(), Requirements{
		Operators: NewOperatorSet(OperatorNEQ),
	})
	assertRequirementsEqual(t, complement.Requirements(), Requirements{
		Operators: NewOperatorSet(OperatorEQ),
	})
}

func TestNormalizationPreservesOrderDuplicatesAndExplicitStructure(t *testing.T) {
	builder := newConstructionBuilder()
	builder.GTE("field", 1)
	builder.LTE("field", 3)
	builder.EQ("field", 2)
	builder.EQ("field", 2)
	builder.AllOf(func(group *Group[constructionExpression]) {
		group.NoneOf(func(nested *Group[constructionExpression]) {
			nested.EQ("field", 3)
		})
	})

	predicate := requireNormalizedPredicate(t, builder)
	root, _ := predicate.Root().AsGroup()
	if root.ChildCount() != 5 {
		t.Fatalf("root child count = %d, want 5", root.ChildCount())
	}
	wantOperators := []Operator{OperatorGTE, OperatorLTE, OperatorEQ, OperatorEQ}
	for index, want := range wantOperators {
		node := requireViewChild(t, root, index, KindComparison)
		view, _ := node.AsComparison()
		if view.Operator() != want {
			t.Fatalf("child %d operator = %v, want %v", index, view.Operator(), want)
		}
	}
	outerNode := requireViewChild(t, root, 4, KindGroup)
	outer, _ := outerNode.AsGroup()
	innerNode := requireViewChild(t, outer, 0, KindGroup)
	inner, _ := innerNode.AsGroup()
	if outer.Logic() != LogicAllOf || inner.Logic() != LogicNoneOf {
		t.Fatalf("nested logic = (%v, %v)", outer.Logic(), inner.Logic())
	}
	requireViewChild(t, inner, 0, KindComparison)
	if predicate.Requirements().Features.Count() != 0 {
		t.Fatal("ordinary leaves unexpectedly require a native feature")
	}
}

func addGroupByLogic(
	builder *Builder[constructionCondition, constructionExpression],
	logic Logic,
	scope Scope[constructionExpression],
) {
	switch logic {
	case LogicAllOf:
		builder.AllOf(scope)
	case LogicAnyOf:
		builder.AnyOf(scope)
	case LogicNoneOf:
		builder.NoneOf(scope)
	case LogicNotAllOf:
		builder.NotAllOf(scope)
	}
}

func requireNormalizedPredicate(
	t *testing.T,
	builder *Builder[constructionCondition, constructionExpression],
) Predicate[constructionCondition, constructionExpression] {
	t.Helper()
	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v, want nil", err)
	}
	return predicate
}

func assertNormalizationMatchSet(
	t *testing.T,
	predicate Predicate[constructionCondition, constructionExpression],
	want []bool,
) {
	t.Helper()
	if len(want) != len(normalizationRecords) {
		t.Fatalf("match-set expectation length = %d, want %d", len(want), len(normalizationRecords))
	}
	for index, record := range normalizationRecords {
		got := evaluateNormalizedNode(t, predicate.Root(), record)
		if got != want[index] {
			t.Errorf("record %q match = %t, want %t", record.name, got, want[index])
		}
	}
}

func evaluateNormalizedNode[C, E any](
	t *testing.T,
	node NodeView[C, E],
	record normalizationRecord,
) bool {
	t.Helper()
	if constant, ok := node.AsConstant(); ok {
		return constant.Value()
	}
	if comparison, ok := node.AsComparison(); ok {
		if record.state != normalizationValue {
			return false
		}
		value, ok := comparison.Value().(int)
		if !ok {
			t.Fatalf("comparison value type = %T, want int", comparison.Value())
		}
		switch comparison.Operator() {
		case OperatorEQ:
			return record.value == value
		case OperatorNEQ:
			return record.value != value
		case OperatorLT:
			return record.value < value
		case OperatorLTE:
			return record.value <= value
		case OperatorGT:
			return record.value > value
		case OperatorGTE:
			return record.value >= value
		}
	}
	if membership, ok := node.AsMembership(); ok {
		if record.state != normalizationValue {
			return false
		}
		found := false
		for index := 0; index < membership.ValueCount(); index++ {
			value, valueOK := membership.Value(index)
			if !valueOK {
				t.Fatalf("membership Value(%d) failed", index)
			}
			if value == record.value {
				found = true
			}
		}
		if membership.Operator() == OperatorIn {
			return found
		}
		return !found
	}
	if nullCheck, ok := node.AsNull(); ok {
		switch nullCheck.Operator() {
		case OperatorIsNull:
			return record.state == normalizationNull
		case OperatorNotNull:
			return record.state == normalizationValue
		}
	}
	if group, ok := node.AsGroup(); ok {
		switch group.Logic() {
		case LogicAllOf:
			for index := 0; index < group.ChildCount(); index++ {
				child, childOK := group.Child(index)
				if !childOK {
					t.Fatalf("AllOf Child(%d) failed", index)
				}
				if !evaluateNormalizedNode(t, child, record) {
					return false
				}
			}
			return true
		case LogicAnyOf:
			for index := 0; index < group.ChildCount(); index++ {
				child, childOK := group.Child(index)
				if !childOK {
					t.Fatalf("AnyOf Child(%d) failed", index)
				}
				if evaluateNormalizedNode(t, child, record) {
					return true
				}
			}
			return false
		case LogicNoneOf:
			return !evaluateGroupAsAnyOf(t, group, record)
		case LogicNotAllOf:
			return !evaluateGroupAsAllOf(t, group, record)
		}
	}
	t.Fatalf("unsupported normalized node kind %v", node.Kind())
	return false
}

func evaluateGroupAsAnyOf[C, E any](
	t *testing.T,
	group GroupView[C, E],
	record normalizationRecord,
) bool {
	t.Helper()
	for index := 0; index < group.ChildCount(); index++ {
		child, ok := group.Child(index)
		if !ok {
			t.Fatalf("AnyOf Child(%d) failed", index)
		}
		if evaluateNormalizedNode(t, child, record) {
			return true
		}
	}
	return false
}

func evaluateGroupAsAllOf[C, E any](
	t *testing.T,
	group GroupView[C, E],
	record normalizationRecord,
) bool {
	t.Helper()
	for index := 0; index < group.ChildCount(); index++ {
		child, ok := group.Child(index)
		if !ok {
			t.Fatalf("AllOf Child(%d) failed", index)
		}
		if !evaluateNormalizedNode(t, child, record) {
			return false
		}
	}
	return true
}

func assertRequirementsEqual(t *testing.T, got, want Requirements) {
	t.Helper()
	if got != want {
		t.Fatalf("Requirements = %#v, want %#v", got, want)
	}
}
