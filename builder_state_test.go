package weave

import (
	"errors"
	"reflect"
	"testing"

	"github.com/imbrooklyn/weave/when"
)

func TestOriginSequenceIsGlobalAcrossRootAndNestedGroups(t *testing.T) {
	builder := newConstructionBuilder()
	builder.EQ("root", 1)
	builder.AllOf(func(group *Group[constructionExpression]) {
		group.NEQ("nested", 2)
		group.AnyOf(func(nested *Group[constructionExpression]) {
			nested.Expr(constructionExpression{name: "deep"})
		})
	})
	builder.Native(constructionCondition{"native"})

	rootChildren := builder.state.root.children
	if len(rootChildren) != 3 {
		t.Fatalf("root child count = %d, want 3", len(rootChildren))
	}
	requireNodeOrigin(t, rootChildren[0], Origin{Sequence: 1, Operator: OperatorEQ})

	outer, ok := rootChildren[1].(*groupNode)
	if !ok {
		t.Fatalf("root child 1 type = %T, want *groupNode", rootChildren[1])
	}
	requireNodeOrigin(t, outer, Origin{Sequence: 2})
	if len(outer.children) != 2 {
		t.Fatalf("outer child count = %d, want 2", len(outer.children))
	}
	requireNodeOrigin(t, outer.children[0], Origin{Sequence: 3, Operator: OperatorNEQ})

	inner, ok := outer.children[1].(*groupNode)
	if !ok {
		t.Fatalf("outer child 1 type = %T, want *groupNode", outer.children[1])
	}
	requireNodeOrigin(t, inner, Origin{Sequence: 4})
	if len(inner.children) != 1 {
		t.Fatalf("inner child count = %d, want 1", len(inner.children))
	}
	requireNodeOrigin(t, inner.children[0], Origin{Sequence: 5})
	requireNodeOrigin(t, rootChildren[2], Origin{Sequence: 6})

	if builder.state.sequence != 6 {
		t.Fatalf("last sequence = %d, want 6", builder.state.sequence)
	}
	requireNoConstructionErrors(t, builder)
}

func TestOmittedCallsConsumeOriginsWithoutValidatingPayloads(t *testing.T) {
	builder := newConstructionBuilder()
	predicateCalls := 0
	var nilValue *int
	scopeCalls := 0

	builder.EQ(nil, nilValue, func(*int) bool {
		predicateCalls++
		return false
	})
	builder.IsNull(nil, true, false, true)
	builder.AllOf(func(*Group[constructionExpression]) {
		scopeCalls++
	}, false)
	builder.Native(constructionCondition{"native"}, false)
	builder.Expr(constructionExpression{name: "expression"}, true, false)
	builder.EQ("included", 9)

	if predicateCalls != 1 {
		t.Fatalf("predicate calls = %d, want 1", predicateCalls)
	}
	if scopeCalls != 0 {
		t.Fatalf("disabled scope calls = %d, want 0", scopeCalls)
	}
	if builder.state.sequence != 6 {
		t.Fatalf("last sequence = %d, want 6", builder.state.sequence)
	}
	comparison := requireSingleRootChild[*comparisonNode](t, builder)
	requireNodeOrigin(t, comparison, Origin{Sequence: 6, Operator: OperatorEQ})
	requireNoConstructionErrors(t, builder)
}

func TestInclusionPredicatesEvaluateOnceInOrderAndShortCircuit(t *testing.T) {
	builder := newConstructionBuilder()
	var calls []string

	builder.EQ(
		nil,
		(*int)(nil),
		func(*int) bool {
			calls = append(calls, "first")
			return true
		},
		func(*int) bool {
			calls = append(calls, "second")
			return false
		},
		func(*int) bool {
			calls = append(calls, "third")
			return true
		},
	)

	if !reflect.DeepEqual(calls, []string{"first", "second"}) {
		t.Fatalf("predicate calls = %#v, want [first second]", calls)
	}
	if len(builder.state.root.children) != 0 {
		t.Fatalf("root child count = %d, want 0", len(builder.state.root.children))
	}
	requireNoConstructionErrors(t, builder)

	calledAfterNil := false
	builder.EQ(
		"field",
		1,
		func(int) bool {
			calls = append(calls, "before-nil")
			return true
		},
		nil,
		func(int) bool {
			calledAfterNil = true
			return true
		},
	)
	if calledAfterNil {
		t.Fatal("predicate after nil was evaluated")
	}
	if len(builder.state.errors) != 1 {
		t.Fatalf("construction error count = %d, want 1", len(builder.state.errors))
	}
	err := builder.state.errors[0]
	if !errors.Is(err, ErrInvalidPredicate) {
		t.Fatalf("error = %v, want ErrInvalidPredicate", err)
	}
	if err.Origin != (Origin{Sequence: 2, Operator: OperatorEQ}) {
		t.Fatalf("error origin = %#v, want sequence 2 EQ", err.Origin)
	}

	falseCalls := 0
	builder.EQ(
		"field",
		2,
		func(int) bool {
			falseCalls++
			return false
		},
		nil,
	)
	if falseCalls != 1 {
		t.Fatalf("false predicate calls = %d, want 1", falseCalls)
	}
	if len(builder.state.errors) != 1 {
		t.Fatalf("a nil predicate after false added an error: %#v", builder.state.errors)
	}
}

func TestMembershipAndRangeInclusionPredicatesEvaluateOnce(t *testing.T) {
	builder := newConstructionBuilder()
	values := constructionNumbers{2, 3}
	membershipCalls := 0
	pairCalls := 0

	builder.In("field", values, func(got constructionNumbers) bool {
		membershipCalls++
		if !reflect.DeepEqual(got, values) {
			t.Fatalf("membership predicate received %#v, want %#v", got, values)
		}
		return true
	})
	builder.Between("field", 2, 3, func(lower, upper int) bool {
		pairCalls++
		if lower != 2 || upper != 3 {
			t.Fatalf("pair predicate received (%d, %d), want (2, 3)", lower, upper)
		}
		return true
	})

	if membershipCalls != 1 || pairCalls != 1 {
		t.Fatalf("predicate calls = (%d, %d), want (1, 1)", membershipCalls, pairCalls)
	}
	if len(builder.state.root.children) != 2 {
		t.Fatalf("root child count = %d, want 2", len(builder.state.root.children))
	}
	requireNoConstructionErrors(t, builder)
}

func TestPairInclusionPredicatesShortCircuitBeforeLaterNil(t *testing.T) {
	builder := newConstructionBuilder()
	falseCalls := 0
	builder.Between(
		"field",
		3,
		1,
		func(int, int) bool {
			falseCalls++
			return false
		},
		nil,
	)
	if falseCalls != 1 {
		t.Fatalf("false pair predicate calls = %d, want 1", falseCalls)
	}
	if len(builder.state.root.children) != 0 || len(builder.state.errors) != 0 {
		t.Fatalf(
			"omitted pair state = %d children, %d errors",
			len(builder.state.root.children),
			len(builder.state.errors),
		)
	}

	afterNil := false
	builder.Between(
		"field",
		1,
		2,
		func(int, int) bool { return true },
		nil,
		func(int, int) bool {
			afterNil = true
			return true
		},
	)
	if afterNil {
		t.Fatal("pair predicate after nil was evaluated")
	}
	if len(builder.state.errors) != 1 || !errors.Is(builder.state.errors[0], ErrInvalidPredicate) {
		t.Fatalf("pair predicate errors = %#v, want ErrInvalidPredicate", builder.state.errors)
	}
	if builder.state.errors[0].Origin != (Origin{Sequence: 2, Operator: OperatorBetween}) {
		t.Fatalf("pair predicate error origin = %#v", builder.state.errors[0].Origin)
	}
}

func TestInclusionPredicatePanicPropagatesAfterConsumingOrigin(t *testing.T) {
	builder := newConstructionBuilder()
	token := &struct{ name string }{name: "predicate panic"}

	recovered := recoverValue(func() {
		builder.EQ("field", 1, func(int) bool {
			panic(token)
		})
	})
	if recovered != token {
		t.Fatalf("recovered value = %#v, want token", recovered)
	}
	if builder.state.sequence != 1 {
		t.Fatalf("last sequence = %d, want 1", builder.state.sequence)
	}
	if len(builder.state.root.children) != 0 || len(builder.state.errors) != 0 {
		t.Fatalf(
			"state after panic = %d children, %d errors",
			len(builder.state.root.children),
			len(builder.state.errors),
		)
	}
}

func TestNullableMembershipConstructionDefersLoweringWithOneOrigin(t *testing.T) {
	builder := newConstructionBuilder()
	one, two := 1, 2
	input := []*int{&one, nil, &two}
	builder.In("field", input)

	membership := requireSingleRootChild[*membershipNode](t, builder)
	wantOrigin := Origin{Sequence: 1, Operator: OperatorIn}
	requireNodeOrigin(t, membership, wantOrigin)
	if membership.operator != OperatorIn || !membership.containsNull {
		t.Fatalf("membership = %#v, want nullable In", membership)
	}
	if !reflect.DeepEqual(membership.values, []any{1, 2}) {
		t.Fatalf("construction values = %#v, want [1 2]", membership.values)
	}
	if membership.inputSliceType != reflect.TypeFor[[]*int]() ||
		membership.inputElementType != reflect.TypeFor[*int]() ||
		membership.elementType != reflect.TypeFor[int]() {
		t.Fatalf(
			"lowered membership types = (%v, %v, %v)",
			membership.inputSliceType,
			membership.inputElementType,
			membership.elementType,
		)
	}
	requireNoConstructionErrors(t, builder)
}

func TestGroupLifecycleDisabledNilScopeAndFreeze(t *testing.T) {
	builder := newConstructionBuilder()
	scopeCalls := 0
	var leaked *Group[constructionExpression]

	builder.AllOf(func(*Group[constructionExpression]) {
		scopeCalls++
	}, false)
	builder.AnyOf(nil)
	builder.NoneOf(func(group *Group[constructionExpression]) {
		scopeCalls++
		leaked = group
		group.EQ("field", 1)
	})

	if scopeCalls != 1 {
		t.Fatalf("scope calls = %d, want 1", scopeCalls)
	}
	if leaked == nil {
		t.Fatal("scope did not expose its active group")
	}
	if leaked.control.lifecycle != groupFrozen {
		t.Fatalf("leaked group lifecycle = %v, want frozen", leaked.control.lifecycle)
	}
	group := requireSingleRootChild[*groupNode](t, builder)
	if group.logic != LogicNoneOf || len(group.children) != 1 {
		t.Fatalf("constructed group = %#v, want one-child none-of", group)
	}

	before := len(group.children)
	leaked.EQ(nil, (*int)(nil))
	if len(group.children) != before {
		t.Fatalf("frozen group child count changed from %d to %d", before, len(group.children))
	}
	if builder.state.sequence != 5 {
		t.Fatalf("last sequence = %d, want 5", builder.state.sequence)
	}
	if len(builder.state.errors) != 2 {
		t.Fatalf("construction error count = %d, want 2", len(builder.state.errors))
	}
	if !errors.Is(builder.state.errors[0], ErrInvalidPredicate) ||
		builder.state.errors[0].Origin.Sequence != 2 {
		t.Fatalf("nil scope error = %#v", builder.state.errors[0])
	}
	if !errors.Is(builder.state.errors[1], ErrInvalidState) ||
		builder.state.errors[1].Origin != (Origin{Sequence: 5, Operator: OperatorEQ}) {
		t.Fatalf("frozen state error = %#v", builder.state.errors[1])
	}
}

func TestScopePanicPropagatesAfterDeferredFreeze(t *testing.T) {
	builder := newConstructionBuilder()
	var leaked *Group[constructionExpression]
	token := &struct{ name string }{name: "scope panic"}

	recovered := recoverValue(func() {
		builder.NotAllOf(func(group *Group[constructionExpression]) {
			leaked = group
			group.Expr(constructionExpression{name: "before panic"})
			panic(token)
		})
	})
	if recovered != token {
		t.Fatalf("recovered value = %#v, want token", recovered)
	}
	if leaked == nil {
		t.Fatal("panicking scope did not expose its active group")
	}
	if leaked.control.lifecycle != groupFrozen {
		t.Fatalf("leaked group lifecycle = %v, want frozen", leaked.control.lifecycle)
	}
	group := requireSingleRootChild[*groupNode](t, builder)
	if group.logic != LogicNotAllOf || len(group.children) != 1 {
		t.Fatalf("partial group = %#v, want one-child not-all-of", group)
	}

	leaked.Expr(constructionExpression{name: "after panic"})
	if len(group.children) != 1 {
		t.Fatalf("frozen group child count = %d, want 1", len(group.children))
	}
	if len(builder.state.errors) != 1 || !errors.Is(builder.state.errors[0], ErrInvalidState) {
		t.Fatalf("post-panic errors = %#v, want ErrInvalidState", builder.state.errors)
	}
	if builder.state.errors[0].Origin.Sequence != 3 {
		t.Fatalf("post-panic error sequence = %d, want 3", builder.state.errors[0].Origin.Sequence)
	}
}

func TestCopiedGroupSharesTheScopeFreezeState(t *testing.T) {
	builder := newConstructionBuilder()
	var copied Group[constructionExpression]
	builder.AllOf(func(group *Group[constructionExpression]) {
		copied = *group
		group.EQ("field", 1)
	})

	(&copied).EQ("field", 2)
	group := requireSingleRootChild[*groupNode](t, builder)
	if len(group.children) != 1 {
		t.Fatalf("frozen copied group child count = %d, want 1", len(group.children))
	}
	if copied.control == nil || copied.control.lifecycle != groupFrozen {
		t.Fatalf("copied group control = %#v, want frozen", copied.control)
	}
	if len(builder.state.errors) != 1 || !errors.Is(builder.state.errors[0], ErrInvalidState) {
		t.Fatalf("copied group errors = %#v, want ErrInvalidState", builder.state.errors)
	}
	if builder.state.errors[0].Origin.Sequence != 3 {
		t.Fatalf("copied group error sequence = %d, want 3", builder.state.errors[0].Origin.Sequence)
	}
}

func TestWhenIfCanControlBuilderMethodsWithoutExtraEvaluation(t *testing.T) {
	builder := newConstructionBuilder()
	builder.EQ("field", 1, when.If[int](false))
	builder.Between("field", 1, 2, when.PairIf[int, int](false))
	if len(builder.state.root.children) != 0 || builder.state.sequence != 2 {
		t.Fatalf(
			"state = %d children, sequence %d; want 0 children, sequence 2",
			len(builder.state.root.children),
			builder.state.sequence,
		)
	}
	requireNoConstructionErrors(t, builder)
}

func requireNodeOrigin(t *testing.T, value node, want Origin) {
	t.Helper()
	if got := value.nodeOrigin(); got != want {
		t.Fatalf("origin = %#v, want %#v", got, want)
	}
}

func recoverValue(function func()) (recovered any) {
	defer func() {
		recovered = recover()
	}()
	function()
	return nil
}
