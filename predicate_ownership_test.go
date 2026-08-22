package weave

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

type ownershipField struct {
	metadata map[string]int
}

type ownershipValue struct {
	metadata map[string]int
}

type ownershipMember struct {
	name     string
	metadata map[string]int
}

type ownershipNativeItem struct {
	name     string
	metadata map[string]int
}

type ownershipCondition []ownershipNativeItem

type ownershipExpression []int

type shallowOwnershipBox struct {
	value int
}

type shallowOwnershipMember struct {
	box *shallowOwnershipBox
}

func TestPredicateSnapshotOwnershipAndBorrowedPayloads(t *testing.T) {
	builder := newBuilder[ownershipCondition, ownershipExpression]()
	field := &ownershipField{metadata: map[string]int{"value": 1}}
	byteValue := []byte{1, 2}
	nestedValue := ownershipValue{metadata: map[string]int{"value": 2}}
	firstMemberMetadata := map[string]int{"value": 3}
	secondMemberMetadata := map[string]int{"value": 4}
	members := []ownershipMember{
		{name: "first", metadata: firstMemberMetadata},
		{name: "second", metadata: secondMemberMetadata},
	}
	firstNativeMetadata := map[string]int{"value": 5}
	secondNativeMetadata := map[string]int{"value": 6}
	native := ownershipCondition{
		{name: "first", metadata: firstNativeMetadata},
		{name: "second", metadata: secondNativeMetadata},
	}
	expression := ownershipExpression{7, 8}

	builder.EQ(field, byteValue)
	builder.EQ("nested", nestedValue)
	builder.In("members", members)
	builder.Native(native)
	builder.Expr(expression)

	byteValue[0] = 101
	members[0] = ownershipMember{name: "replaced before snapshot"}
	native[0] = ownershipNativeItem{name: "replaced before snapshot"}
	expression[0] = 107
	field.metadata["value"] = 11
	nestedValue.metadata["value"] = 12
	firstMemberMetadata["value"] = 13
	firstNativeMetadata["value"] = 14

	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v, want nil", err)
	}

	byteValue[1] = 102
	members[1] = ownershipMember{name: "replaced after snapshot"}
	native[1] = ownershipNativeItem{name: "replaced after snapshot"}
	expression[1] = 108
	field.metadata["value"] = 21
	nestedValue.metadata["value"] = 22
	firstMemberMetadata["value"] = 23
	firstNativeMetadata["value"] = 24

	root, _ := predicate.Root().AsGroup()
	byteNode := requireViewChild(t, root, 0, KindComparison)
	byteComparison, _ := byteNode.AsComparison()
	gotBytes, ok := byteComparison.Value().([]byte)
	if !ok || !reflect.DeepEqual(gotBytes, []byte{1, 2}) {
		t.Fatalf("byte value = %#v, want [1 2]", byteComparison.Value())
	}
	if byteComparison.ValueType() != reflect.TypeFor[[]byte]() {
		t.Fatalf("byte value type = %v, want []byte", byteComparison.ValueType())
	}
	if gotField := byteComparison.Field().(*ownershipField); gotField.metadata["value"] != 21 {
		t.Fatalf("borrowed field metadata = %#v, want value 21", gotField.metadata)
	}

	nestedNode := requireViewChild(t, root, 1, KindComparison)
	nestedComparison, _ := nestedNode.AsComparison()
	gotNested := nestedComparison.Value().(ownershipValue)
	if gotNested.metadata["value"] != 22 {
		t.Fatalf("borrowed value metadata = %#v, want value 22", gotNested.metadata)
	}

	membershipNodeView := requireViewChild(t, root, 2, KindMembership)
	membership, _ := membershipNodeView.AsMembership()
	firstMemberValue, ok := membership.Value(0)
	if !ok {
		t.Fatal("membership Value(0) failed")
	}
	firstMember := firstMemberValue.(ownershipMember)
	if firstMember.name != "first" || firstMember.metadata["value"] != 23 {
		t.Fatalf("first membership value = %#v", firstMember)
	}
	secondMemberValue, _ := membership.Value(1)
	if secondMemberValue.(ownershipMember).name != "second" {
		t.Fatalf("second membership value = %#v, want original top-level value", secondMemberValue)
	}

	nativeNode := requireViewChild(t, root, 3, KindNativeCondition)
	nativeView, _ := nativeNode.AsNativeCondition()
	gotNative := nativeView.Condition()
	if len(gotNative) != 2 ||
		gotNative[0].name != "first" ||
		gotNative[1].name != "second" ||
		gotNative[0].metadata["value"] != 24 {
		t.Fatalf("native condition = %#v", gotNative)
	}

	expressionNode := requireViewChild(t, root, 4, KindNativeExpression)
	expressionView, _ := expressionNode.AsNativeExpression()
	if got := expressionView.Expression(); !reflect.DeepEqual(got, ownershipExpression{107, 108}) {
		t.Fatalf("opaque expression = %#v, want [107 108]", got)
	}

	gotBytes[0] = 201
	if got := byteComparison.Value().([]byte); !reflect.DeepEqual(got, []byte{1, 2}) {
		t.Fatalf("byte accessor leaked snapshot backing: %#v", got)
	}
	gotNative[0] = ownershipNativeItem{name: "replaced through view"}
	if got := nativeView.Condition(); got[0].name != "first" {
		t.Fatalf("native accessor leaked snapshot backing: %#v", got)
	}

	borrowedNative := nativeView.Condition()
	borrowedNative[0].metadata["value"] = 34
	if got := nativeView.Condition()[0].metadata["value"]; got != 34 {
		t.Fatalf("native nested reference = %d, want borrowed value 34", got)
	}
	firstMember.metadata["value"] = 33
	firstAgain, _ := membership.Value(0)
	if got := firstAgain.(ownershipMember).metadata["value"]; got != 33 {
		t.Fatalf("membership nested reference = %d, want borrowed value 33", got)
	}
	borrowedExpression := expressionView.Expression()
	borrowedExpression[0] = 207
	if got := expressionView.Expression()[0]; got != 207 {
		t.Fatalf("opaque expression reference = %d, want borrowed value 207", got)
	}

	builder.state.root.children[0].(*comparisonNode).value.([]byte)[0] = 251
	builder.state.root.children[2].(*membershipNode).values[0] = ownershipMember{name: "builder mutation"}
	builder.state.root.children[3].(*nativeConditionNode[ownershipCondition]).condition[0] = ownershipNativeItem{name: "builder mutation"}
	if got := byteComparison.Value().([]byte)[0]; got != 1 {
		t.Fatalf("snapshot byte value after Builder mutation = %d, want 1", got)
	}
	firstAgain, _ = membership.Value(0)
	if got := firstAgain.(ownershipMember).name; got != "first" {
		t.Fatalf("snapshot membership after Builder mutation = %q, want first", got)
	}
	if got := nativeView.Condition()[0].name; got != "first" {
		t.Fatalf("snapshot Native after Builder mutation = %q, want first", got)
	}
}

func TestOwnershipBoundaryClonesOnlyPromisedTopLevelContainers(t *testing.T) {
	builder := newBuilder[[]*shallowOwnershipBox, []*shallowOwnershipBox]()
	field := []int{1}
	comparisonValue := []int{2}
	membershipBox := &shallowOwnershipBox{value: 3}
	membershipValues := []shallowOwnershipMember{{box: membershipBox}}
	nativeBox := &shallowOwnershipBox{value: 4}
	nativeValue := []*shallowOwnershipBox{nativeBox}
	expressionBox := &shallowOwnershipBox{value: 5}
	expressionValue := []*shallowOwnershipBox{expressionBox}

	builder.EQ(field, comparisonValue)
	builder.In("membership", membershipValues)
	builder.Native(nativeValue)
	builder.Expr(expressionValue)

	field[0] = 11
	comparisonValue[0] = 12
	membershipValues[0] = shallowOwnershipMember{box: &shallowOwnershipBox{value: 103}}
	membershipBox.value = 13
	nativeValue[0] = &shallowOwnershipBox{value: 104}
	nativeBox.value = 14
	expressionValue[0] = &shallowOwnershipBox{value: 15}

	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v, want nil", err)
	}

	field[0] = 21
	comparisonValue[0] = 22
	membershipValues[0] = shallowOwnershipMember{box: &shallowOwnershipBox{value: 203}}
	membershipBox.value = 23
	nativeValue[0] = &shallowOwnershipBox{value: 204}
	nativeBox.value = 24
	expressionValue[0] = &shallowOwnershipBox{value: 25}

	root, _ := predicate.Root().AsGroup()
	comparisonNodeView := requireViewChild(t, root, 0, KindComparison)
	comparison, _ := comparisonNodeView.AsComparison()
	gotField := comparison.Field().([]int)
	gotComparisonValue := comparison.Value().([]int)
	if gotField[0] != 21 || gotComparisonValue[0] != 22 {
		t.Fatalf(
			"borrowed field/value = (%#v, %#v), want ([21], [22])",
			gotField,
			gotComparisonValue,
		)
	}

	membershipNodeView := requireViewChild(t, root, 1, KindMembership)
	membership, _ := membershipNodeView.AsMembership()
	memberValue, ok := membership.Value(0)
	if !ok {
		t.Fatal("membership Value(0) failed")
	}
	member := memberValue.(shallowOwnershipMember)
	if member.box != membershipBox || member.box.value != 23 {
		t.Fatalf("membership value = %#v, want cloned top level and borrowed box", member)
	}

	nativeNodeView := requireViewChild(t, root, 2, KindNativeCondition)
	native, _ := nativeNodeView.AsNativeCondition()
	gotNative := native.Condition()
	if len(gotNative) != 1 || gotNative[0] != nativeBox || gotNative[0].value != 24 {
		t.Fatalf("native value = %#v, want cloned top level and borrowed box", gotNative)
	}

	expressionNodeView := requireViewChild(t, root, 3, KindNativeExpression)
	expression, _ := expressionNodeView.AsNativeExpression()
	gotExpression := expression.Expression()
	if len(gotExpression) != 1 || gotExpression[0].value != 25 {
		t.Fatalf("expression value = %#v, want fully borrowed slice", gotExpression)
	}

	gotField[0] = 31
	gotComparisonValue[0] = 32
	if comparison.Field().([]int)[0] != 31 || comparison.Value().([]int)[0] != 32 {
		t.Fatal("ordinary field or non-byte comparison slice was unexpectedly cloned by an accessor")
	}
	member.box.value = 33
	memberAgain, _ := membership.Value(0)
	if memberAgain.(shallowOwnershipMember).box.value != 33 {
		t.Fatal("membership element reference was unexpectedly deep-copied")
	}
	gotNative[0] = &shallowOwnershipBox{value: 304}
	if native.Condition()[0] != nativeBox {
		t.Fatal("Native Condition() exposed core-owned top-level slice backing")
	}
	nativeBox.value = 34
	if native.Condition()[0].value != 34 {
		t.Fatal("Native nested reference was unexpectedly deep-copied")
	}
	gotExpression[0] = &shallowOwnershipBox{value: 35}
	if expression.Expression()[0].value != 35 {
		t.Fatal("opaque Expr top-level slice was unexpectedly cloned")
	}
}

func TestPredicateAndNodeViewsSupportConcurrentReadTraversal(t *testing.T) {
	builder := newConstructionBuilder()
	builder.
		EQ("eq", 1).
		In("in", constructionNumbers{1, 2, 3}).
		Between("between", 1, 5).
		NotNull("not-null").
		HasPrefix("prefix", "pre").
		AnyOf(func(group *Group[constructionExpression]) {
			group.EQ("nested", 2)
			group.Expr(constructionExpression{name: "nested"})
		}).
		Native(constructionCondition{"native"}).
		Expr(constructionExpression{name: "root"})

	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v, want nil", err)
	}

	const goroutineCount = 16
	const traversalCount = 250
	errorsFound := make(chan error, goroutineCount)
	var waitGroup sync.WaitGroup
	waitGroup.Add(goroutineCount)
	for worker := 0; worker < goroutineCount; worker++ {
		go func() {
			defer waitGroup.Done()
			for iteration := 0; iteration < traversalCount; iteration++ {
				count, err := inspectPredicateViews(predicate.Root())
				if err != nil {
					errorsFound <- err
					return
				}
				if count != 11 {
					errorsFound <- fmt.Errorf("node count = %d, want 11", count)
					return
				}
			}
		}()
	}
	waitGroup.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Error(err)
	}
}

func inspectPredicateViews[C, E any](root NodeView[C, E]) (int, error) {
	stack := []NodeView[C, E]{root}
	count := 0
	for len(stack) != 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if !current.Valid() || current.Path().SegmentCount() == 0 {
			return 0, fmt.Errorf("invalid node encountered at index %d", count)
		}
		_ = current.Path().String()
		_ = current.Origin()
		count++

		switch current.Kind() {
		case KindConstant:
			view, ok := current.AsConstant()
			if !ok {
				return 0, fmt.Errorf("AsConstant failed")
			}
			_ = view.Value()
		case KindComparison:
			view, ok := current.AsComparison()
			if !ok {
				return 0, fmt.Errorf("AsComparison failed")
			}
			_, _, _, _ = view.Operator(), view.Field(), view.Value(), view.ValueType()
		case KindMembership:
			view, ok := current.AsMembership()
			if !ok {
				return 0, fmt.Errorf("AsMembership failed")
			}
			for index := 0; index < view.ValueCount(); index++ {
				if _, ok := view.Value(index); !ok {
					return 0, fmt.Errorf("membership Value(%d) failed", index)
				}
			}
			_, _, _ = view.InputSliceType(), view.InputElementType(), view.ElementType()
		case KindRange:
			view, ok := current.AsRange()
			if !ok {
				return 0, fmt.Errorf("AsRange failed")
			}
			_, _, _, _, _ = view.Operator(), view.Field(), view.Lower(), view.Upper(), view.BoundType()
		case KindNull:
			view, ok := current.AsNull()
			if !ok {
				return 0, fmt.Errorf("AsNull failed")
			}
			_, _ = view.Operator(), view.Field()
		case KindText:
			view, ok := current.AsText()
			if !ok {
				return 0, fmt.Errorf("AsText failed")
			}
			_, _, _ = view.Operator(), view.Field(), view.Value()
		case KindGroup:
			view, ok := current.AsGroup()
			if !ok {
				return 0, fmt.Errorf("AsGroup failed")
			}
			_ = view.Logic()
			for index := view.ChildCount() - 1; index >= 0; index-- {
				child, ok := view.Child(index)
				if !ok {
					return 0, fmt.Errorf("group Child(%d) failed", index)
				}
				stack = append(stack, child)
			}
		case KindNativeCondition:
			view, ok := current.AsNativeCondition()
			if !ok {
				return 0, fmt.Errorf("AsNativeCondition failed")
			}
			_ = view.Condition()
		case KindNativeExpression:
			view, ok := current.AsNativeExpression()
			if !ok {
				return 0, fmt.Errorf("AsNativeExpression failed")
			}
			_ = view.Expression()
		default:
			return 0, fmt.Errorf("unknown Kind %v", current.Kind())
		}
	}
	return count, nil
}
