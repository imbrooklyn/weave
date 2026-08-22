package weave

import (
	"errors"
	"testing"
)

func TestPredicateIdentityAndInvalidStateBoundaries(t *testing.T) {
	var zero Predicate[constructionCondition, constructionExpression]
	if root := zero.Root(); root.Valid() || root.Kind() != 0 {
		t.Fatalf("zero Predicate root = %#v, want invalid view", root)
	}
	if zero.statusFor(newPredicateDomain()) != predicateInvalid {
		t.Fatal("zero Predicate status is not invalid")
	}

	builder := newConstructionBuilder()
	builder.EQ("field", 1)
	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v, want nil", err)
	}
	if predicate.statusFor(builder.domain) != predicateValid {
		t.Fatal("Predicate status for its domain is not valid")
	}
	if predicate.statusFor(newPredicateDomain()) != predicateForeign {
		t.Fatal("Predicate status for another domain is not foreign")
	}
	if !predicate.Root().Valid() {
		t.Fatal("valid Predicate returned an invalid root")
	}

	validRootState := func() *predicateState {
		state := &predicateState{
			seal:   validPredicateSeal,
			domain: newPredicateDomain(),
		}
		state.root = &groupNode{
			nodeBase: nodeBase{
				owner: state,
				path: newNodePath(
					newRootPathSegment(LogicAllOf),
				),
			},
			logic: LogicAllOf,
		}
		return state
	}

	tests := []struct {
		name  string
		state func() *predicateState
	}{
		{name: "nil state", state: func() *predicateState { return nil }},
		{
			name: "foreign seal",
			state: func() *predicateState {
				state := validRootState()
				state.seal = &predicateSeal{marker: 1}
				return state
			},
		},
		{
			name: "nil domain",
			state: func() *predicateState {
				state := validRootState()
				state.domain = nil
				return state
			},
		},
		{
			name: "nil root",
			state: func() *predicateState {
				state := validRootState()
				state.root = nil
				return state
			},
		},
		{
			name: "foreign root owner",
			state: func() *predicateState {
				state := validRootState()
				state.root.owner = validRootState()
				return state
			},
		},
		{
			name: "non-AllOf root",
			state: func() *predicateState {
				state := validRootState()
				state.root.logic = LogicAnyOf
				return state
			},
		},
		{
			name: "nonzero root origin",
			state: func() *predicateState {
				state := validRootState()
				state.root.origin = Origin{Sequence: 1}
				return state
			},
		},
		{
			name: "invalid root path",
			state: func() *predicateState {
				state := validRootState()
				state.root.path = NodePath{}
				return state
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			predicate := Predicate[constructionCondition, constructionExpression]{
				state: test.state(),
			}
			root := predicate.Root()
			if root.Valid() || root.Kind() != 0 || root.Path().String() != "" {
				t.Fatalf("invalid Predicate root = %#v, want invalid view", root)
			}
			if predicate.statusFor(newPredicateDomain()) != predicateInvalid {
				t.Fatal("malformed Predicate status is not invalid")
			}
		})
	}
}

func TestPredicateRootIsAlwaysImplicitAllOf(t *testing.T) {
	builder := newConstructionBuilder()
	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v, want nil", err)
	}

	root := predicate.Root()
	if !root.Valid() || root.Kind() != KindGroup {
		t.Fatalf("root = %#v, want valid group", root)
	}
	if root.Origin() != (Origin{}) {
		t.Fatalf("root origin = %#v, want zero", root.Origin())
	}
	if root.Path().String() != "root.allOf" {
		t.Fatalf("root path = %q, want root.allOf", root.Path().String())
	}
	group, ok := root.AsGroup()
	if !ok || group.Logic() != LogicAllOf || group.ChildCount() != 0 {
		t.Fatalf("root group = %#v, want empty AllOf", group)
	}
}

func TestPredicateSnapshotPreservesOriginsAcrossOmittedCalls(t *testing.T) {
	builder := newConstructionBuilder()
	builder.EQ("omitted comparison", 1, func(int) bool { return false })
	builder.IsNull("omitted null", false)
	builder.GT("included", 2)

	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v, want nil", err)
	}
	root, _ := predicate.Root().AsGroup()
	if root.ChildCount() != 1 {
		t.Fatalf("root child count = %d, want 1", root.ChildCount())
	}
	child, _ := root.Child(0)
	wantOrigin := Origin{Sequence: 3, Operator: OperatorGT}
	if child.Origin() != wantOrigin {
		t.Fatalf("included child origin = %#v, want %#v", child.Origin(), wantOrigin)
	}
	if got := child.Path().String(); got != "root.allOf[0].gt" {
		t.Fatalf("included child path = %q, want root.allOf[0].gt", got)
	}
}

func TestPredicateSnapshotsAreTopologicallyIndependent(t *testing.T) {
	builder := newConstructionBuilder()
	builder.EQ("first", 1)

	first, err := builder.Predicate()
	if err != nil {
		t.Fatalf("first Predicate() error = %v, want nil", err)
	}
	firstRoot, _ := first.Root().AsGroup()
	firstChild, _ := firstRoot.Child(0)

	builder.GT("second", 2)
	second, err := builder.Predicate()
	if err != nil {
		t.Fatalf("second Predicate() error = %v, want nil", err)
	}
	secondRoot, _ := second.Root().AsGroup()

	if firstRoot.ChildCount() != 1 {
		t.Fatalf("first snapshot child count = %d, want 1", firstRoot.ChildCount())
	}
	if secondRoot.ChildCount() != 2 {
		t.Fatalf("second snapshot child count = %d, want 2", secondRoot.ChildCount())
	}
	if first.state == second.state || first.state.root == second.state.root {
		t.Fatal("repeated Predicate calls shared snapshot topology")
	}
	if first.state.root == builder.state.root {
		t.Fatal("Predicate root aliases Builder root")
	}
	if firstChild.node == builder.state.root.children[0] {
		t.Fatal("Predicate child aliases Builder child")
	}

	builder.state.root.children[0].(*comparisonNode).field = "changed"
	comparison, _ := firstChild.AsComparison()
	if comparison.Field() != "first" {
		t.Fatalf("first snapshot field = %v, want first", comparison.Field())
	}
}

func TestPredicateReturnsZeroForStableConstructionDiagnostics(t *testing.T) {
	builder := newConstructionBuilder()
	var field *int
	var value *int
	builder.EQ(field, value)
	builder.Between("range", 2, 1)

	predicate, err := builder.Predicate()
	if err == nil {
		t.Fatal("Predicate() error = nil, want construction diagnostics")
	}
	if predicate.Root().Valid() {
		t.Fatal("failed Predicate() returned a valid Predicate")
	}
	if !errors.Is(err, ErrInvalidField) ||
		!errors.Is(err, ErrInvalidValue) ||
		!errors.Is(err, ErrInvalidRange) {
		t.Fatalf("Predicate() error = %v, want all construction categories", err)
	}

	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("Predicate() error type = %T, want joined error", err)
	}
	unwrapped := joined.Unwrap()
	if len(unwrapped) != 3 {
		t.Fatalf("joined error count = %d, want 3", len(unwrapped))
	}
	wantCodes := []ErrorCode{CodeInvalidField, CodeInvalidValue, CodeInvalidRange}
	for index, want := range wantCodes {
		diagnostic, ok := unwrapped[index].(*Error)
		if !ok || diagnostic.Code != want {
			t.Fatalf("diagnostic %d = %#v, want code %v", index, unwrapped[index], want)
		}
	}

	single := newConstructionBuilder()
	single.Between("range", 2, 1)
	_, err = single.Predicate()
	diagnostic, ok := err.(*Error)
	if !ok {
		t.Fatalf("single Predicate() error type = %T, want *Error", err)
	}
	diagnostic.Code = CodeInvalidPredicate
	diagnostic.Origin = Origin{}
	_, err = single.Predicate()
	diagnostic, ok = err.(*Error)
	if !ok ||
		diagnostic.Code != CodeInvalidRange ||
		diagnostic.Origin != (Origin{Sequence: 1, Operator: OperatorBetween}) {
		t.Fatalf("repeated Predicate() error = %#v, want independent original diagnostic", err)
	}
}

func TestPredicateDepthBoundaryAndDefensiveTraversal(t *testing.T) {
	t.Run("depth 128 snapshots and traverses", func(t *testing.T) {
		builder := newConstructionBuilder()
		builder.AllOf(nestedConstructionScope(MaxPredicateDepth - 2))
		predicate, err := builder.Predicate()
		if err != nil {
			t.Fatalf("Predicate() error = %v, want nil", err)
		}
		if depth := maximumViewDepth(t, predicate.Root()); depth != MaxPredicateDepth {
			t.Fatalf("maximum depth = %d, want %d", depth, MaxPredicateDepth)
		}
	})

	t.Run("depth 129 construction fails without panic", func(t *testing.T) {
		builder := newConstructionBuilder()
		builder.AllOf(nestedConstructionScope(MaxPredicateDepth - 1))
		predicate, err := builder.Predicate()
		if !errors.Is(err, ErrDepthLimit) {
			t.Fatalf("Predicate() error = %v, want ErrDepthLimit", err)
		}
		if predicate.Root().Valid() {
			t.Fatal("depth failure returned a valid Predicate")
		}
	})

	t.Run("defensive snapshot rejects depth 129", func(t *testing.T) {
		builder := newConstructionBuilder()
		parent := builder.state.root
		for depth := 1; depth <= MaxPredicateDepth+1; depth++ {
			child := &groupNode{
				nodeBase: nodeBase{origin: Origin{Sequence: uint64(depth)}},
				logic:    LogicAllOf,
			}
			parent.children = append(parent.children, child)
			parent = child
		}

		predicate, err := builder.Predicate()
		if !errors.Is(err, ErrDepthLimit) {
			t.Fatalf("Predicate() error = %v, want ErrDepthLimit", err)
		}
		if predicate.Root().Valid() {
			t.Fatal("defensive depth failure returned a valid Predicate")
		}
	})

	t.Run("defensive snapshot rejects a cycle", func(t *testing.T) {
		builder := newConstructionBuilder()
		group := &groupNode{
			nodeBase: nodeBase{origin: Origin{Sequence: 1}},
			logic:    LogicAllOf,
		}
		group.children = append(group.children, group)
		builder.state.root.children = append(builder.state.root.children, group)

		predicate, err := builder.Predicate()
		if !errors.Is(err, ErrInvalidPredicate) {
			t.Fatalf("Predicate() error = %v, want ErrInvalidPredicate", err)
		}
		if predicate.Root().Valid() {
			t.Fatal("cyclic snapshot returned a valid Predicate")
		}
	})

	t.Run("defensive snapshot rejects nested Native", func(t *testing.T) {
		builder := newConstructionBuilder()
		group := &groupNode{
			nodeBase: nodeBase{origin: Origin{Sequence: 1}},
			logic:    LogicAnyOf,
			children: []node{
				&nativeConditionNode[constructionCondition]{
					nodeBase: nodeBase{origin: Origin{Sequence: 2}},
					condition: constructionCondition{
						"native",
					},
				},
			},
		}
		builder.state.root.children = append(builder.state.root.children, group)

		predicate, err := builder.Predicate()
		if !errors.Is(err, ErrNonNestableNative) {
			t.Fatalf("Predicate() error = %v, want ErrNonNestableNative", err)
		}
		if predicate.Root().Valid() {
			t.Fatal("nested Native failure returned a valid Predicate")
		}
	})
}

func maximumViewDepth[C, E any](t *testing.T, root NodeView[C, E]) int {
	t.Helper()
	type pendingNode struct {
		node  NodeView[C, E]
		depth int
	}
	stack := []pendingNode{{node: root}}
	maximum := 0
	for len(stack) != 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if !current.node.Valid() {
			t.Fatal("traversal encountered an invalid node")
		}
		if current.depth > maximum {
			maximum = current.depth
		}
		group, ok := current.node.AsGroup()
		if !ok {
			continue
		}
		for index := 0; index < group.ChildCount(); index++ {
			child, ok := group.Child(index)
			if !ok {
				t.Fatalf("Child(%d) failed within bounds", index)
			}
			stack = append(stack, pendingNode{
				node:  child,
				depth: current.depth + 1,
			})
		}
	}
	return maximum
}
