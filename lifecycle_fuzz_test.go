package weave

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

type lifecycleFuzzCompiler struct {
	capabilities      Capabilities
	capabilitiesCalls int
	compileCalls      int
}

func (c *lifecycleFuzzCompiler) Compile(
	predicate Predicate[constructionCondition, constructionExpression],
) (constructionCondition, error) {
	c.compileCalls++
	if !predicate.Root().Valid() {
		return nil, ErrInvalidPredicate
	}
	return constructionCondition{"compiled"}, nil
}

func (c *lifecycleFuzzCompiler) Capabilities() Capabilities {
	c.capabilitiesCalls++
	return c.capabilities
}

type lifecycleFuzzCursor struct {
	data  []byte
	index int
}

func (c *lifecycleFuzzCursor) take() (byte, bool) {
	if c.index >= len(c.data) {
		return 0, false
	}
	value := c.data[c.index]
	c.index++
	return value, true
}

type lifecycleMissing struct {
	code     ErrorCode
	path     string
	origin   Origin
	operator Operator
	feature  Feature
}

func FuzzEndToEndLifecycle(f *testing.F) {
	f.Add([]byte{}, uint32(0))
	f.Add([]byte{2, 3, 12, 28, 44, 60}, uint32(0))
	f.Add([]byte{5, 6, 11, 0, 1, 13, 14}, uint32(0xffff))
	f.Add([]byte{139, 11, 27, 43, 59, 75, 91}, uint32(0x15555))
	f.Add([]byte{15, 31, 47, 63, 79, 95}, uint32(0x2aaaa))

	f.Fuzz(func(t *testing.T, data []byte, capabilityMask uint32) {
		if len(data) > 96 {
			data = data[:96]
		}
		initialCapabilities := lifecycleCapabilities(capabilityMask)
		compiler := &lifecycleFuzzCompiler{
			capabilities: initialCapabilities,
		}
		factory := NewFactory[constructionCondition, constructionExpression](
			compiler,
		)
		compiler.capabilities = lifecycleCapabilities(^uint32(0))
		builder := factory.New()
		cursor := lifecycleFuzzCursor{data: data}
		for cursor.index < len(cursor.data) {
			addLifecycleBuilderNode(builder, &cursor, 0)
		}

		first, firstError := builder.Predicate()
		second, secondError := builder.Predicate()
		firstErrorFingerprint := predicateErrorFingerprint(firstError)
		if secondFingerprint := predicateErrorFingerprint(secondError); secondFingerprint != firstErrorFingerprint {
			t.Fatalf(
				"repeated Predicate errors differ: first %q, second %q",
				firstErrorFingerprint,
				secondFingerprint,
			)
		}
		if compiler.capabilitiesCalls != 1 {
			t.Fatalf(
				"Capabilities calls = %d, want 1",
				compiler.capabilitiesCalls,
			)
		}
		if factory.Capabilities() != initialCapabilities {
			t.Fatal("Factory capability snapshot changed after Compiler mutation")
		}

		if firstError != nil {
			if first.Root().Valid() || second.Root().Valid() {
				t.Fatal("failed Predicate returned a valid root")
			}
			built, buildError := builder.Build()
			if built != nil {
				t.Fatalf("failed Build result = %#v, want zero", built)
			}
			if fingerprint := predicateErrorFingerprint(buildError); fingerprint != firstErrorFingerprint {
				t.Fatalf(
					"Build error = %q, want Predicate error %q",
					fingerprint,
					firstErrorFingerprint,
				)
			}
			if compiler.compileCalls != 0 {
				t.Fatalf(
					"Compile calls after Predicate failure = %d, want 0",
					compiler.compileCalls,
				)
			}
			return
		}

		firstFingerprint, ok := predicateViewFingerprint(t, first.Root())
		if !ok {
			t.Fatal("first Predicate contains an invalid view")
		}
		secondFingerprint, ok := predicateViewFingerprint(t, second.Root())
		if !ok || secondFingerprint != firstFingerprint {
			t.Fatalf(
				"repeated Predicate differs: first %q, second %q",
				firstFingerprint,
				secondFingerprint,
			)
		}
		if first.state == second.state || first.state.root == second.state.root {
			t.Fatal("repeated Predicate reused snapshot topology")
		}

		independentRequirements := requirementsFromNodeViews(t, first.Root())
		assertRequirementsEqual(
			t,
			first.Requirements(),
			independentRequirements,
		)
		assertRequirementsEqual(
			t,
			second.Requirements(),
			independentRequirements,
		)

		renormalizedState, normalizeError := normalizePredicateState[constructionCondition, constructionExpression](
			first.state,
			first.state.domain,
		)
		if normalizeError != nil {
			t.Fatalf("second normalization failed: %v", normalizeError)
		}
		renormalized := Predicate[constructionCondition, constructionExpression]{
			state: renormalizedState,
		}
		renormalizedFingerprint, ok := predicateViewFingerprint(
			t,
			renormalized.Root(),
		)
		if !ok || renormalizedFingerprint != firstFingerprint {
			t.Fatalf(
				"normalization is not idempotent: first %q, second %q",
				firstFingerprint,
				renormalizedFingerprint,
			)
		}
		assertRequirementsEqual(
			t,
			renormalized.Requirements(),
			independentRequirements,
		)

		expectedMissing := firstLifecycleMissing(
			t,
			first.Root(),
			initialCapabilities,
		)
		firstCompiled, firstCompileError := factory.Compile(first)
		secondCompiled, secondCompileError := factory.Compile(first)
		built, buildError := builder.Build()
		if expectedMissing != nil {
			if firstCompiled != nil || secondCompiled != nil || built != nil {
				t.Fatal("preflight failure returned a nonzero condition")
			}
			assertLifecycleMissing(t, firstCompileError, *expectedMissing)
			assertLifecycleMissing(t, secondCompileError, *expectedMissing)
			assertLifecycleMissing(t, buildError, *expectedMissing)
			firstCompileFingerprint := predicateErrorFingerprint(firstCompileError)
			if secondCompileFingerprint := predicateErrorFingerprint(secondCompileError); secondCompileFingerprint != firstCompileFingerprint {
				t.Fatalf(
					"repeated Compile errors differ: first %q, second %q",
					firstCompileFingerprint,
					secondCompileFingerprint,
				)
			}
			if buildFingerprint := predicateErrorFingerprint(buildError); buildFingerprint != firstCompileFingerprint {
				t.Fatalf(
					"Build error = %q, want Compile error %q",
					buildFingerprint,
					firstCompileFingerprint,
				)
			}
			if compiler.compileCalls != 0 {
				t.Fatalf(
					"Compiler calls after preflight failure = %d, want 0",
					compiler.compileCalls,
				)
			}
			return
		}

		for _, result := range []struct {
			name      string
			condition constructionCondition
		}{
			{name: "first Compile", condition: firstCompiled},
			{name: "second Compile", condition: secondCompiled},
			{name: "Build", condition: built},
		} {
			if !reflect.DeepEqual(
				result.condition,
				constructionCondition{"compiled"},
			) {
				t.Fatalf(
					"%s result = %#v, want compiled",
					result.name,
					result.condition,
				)
			}
		}
		if firstCompileError != nil || secondCompileError != nil || buildError != nil {
			t.Fatal("supported Predicate returned a compile error")
		}
		if compiler.compileCalls != 3 {
			t.Fatalf("Compiler calls = %d, want 3", compiler.compileCalls)
		}
	})
}

func FuzzCompilePreflightDeterminism(f *testing.F) {
	for mode := uint8(0); mode < 16; mode++ {
		f.Add(mode)
	}

	f.Fuzz(func(t *testing.T, rawMode uint8) {
		compiler := &factoryTestCompiler{
			capabilities: lifecycleCapabilities(^uint32(0)),
			result:       "compiled",
		}
		factory := NewFactory[string, string](compiler)
		predicate, err := factory.New().
			EQ("comparison", 1).
			AllOf(func(group *Group[string]) {
				group.In("membership", []int{1, 2})
			}).
			Expr("expression").
			Native("native").
			Predicate()
		if err != nil {
			t.Fatalf("Predicate failed: %v", err)
		}

		mode := rawMode % 16
		state := predicate.state
		comparison := state.root.children[0].(*comparisonNode)
		group := state.root.children[1].(*groupNode)
		membership := group.children[0].(*membershipNode)
		native := state.root.children[3].(*nativeConditionNode[string])
		switch mode {
		case 0:
		case 1:
			comparison.owner = nil
		case 2:
			comparison.path = NodePath{}
		case 3:
			comparison.origin = Origin{}
		case 4:
			comparison.operator = Operator(65535)
		case 5:
			membership.containsNull = true
		case 6:
			membership.values = nil
		case 7:
			group.children = nil
		case 8:
			group.logic = Logic(255)
		case 9:
			state.requirements = Requirements{}
		case 10:
			state.root.children = append(
				state.root.children,
				comparison,
			)
		case 11:
			state.root.children = state.root.children[:3]
			native.path = appendSnapshotPath(
				group.path,
				len(group.children),
				KindNativeCondition,
				0,
				0,
			)
			group.children = append(group.children, native)
		case 12:
			state.root.children[0] = (*comparisonNode)(nil)
		case 13:
			constant := &constantNode{
				nodeBase: nodeBase{
					origin: Origin{Sequence: 6, Operator: OperatorNotIn},
					owner:  state,
					path: appendSnapshotPath(
						state.root.path,
						len(state.root.children),
						KindConstant,
						0,
						0,
					),
				},
				value: true,
			}
			state.root.children = append(state.root.children, constant)
		case 14:
			comparison.field = (*int)(nil)
		case 15:
			group.children = append(group.children, group)
		}

		firstResult, firstError := factory.Compile(predicate)
		secondResult, secondError := factory.Compile(predicate)
		if mode == 0 {
			if firstResult != "compiled" || secondResult != "compiled" ||
				firstError != nil || secondError != nil {
				t.Fatal("unmodified Predicate did not compile deterministically")
			}
			if calls := compiler.compileCalls.Load(); calls != 2 {
				t.Fatalf("Compile calls = %d, want 2", calls)
			}
			return
		}

		if firstResult != "" || secondResult != "" {
			t.Fatal("invalid Predicate returned a nonzero condition")
		}
		if firstError == nil || secondError == nil ||
			!errors.Is(firstError, ErrCompile) ||
			!errors.Is(secondError, ErrCompile) {
			t.Fatal("invalid Predicate did not return compile-stage errors")
		}
		firstFingerprint := predicateErrorFingerprint(firstError)
		if secondFingerprint := predicateErrorFingerprint(secondError); secondFingerprint != firstFingerprint {
			t.Fatalf(
				"preflight errors differ: first %q, second %q",
				firstFingerprint,
				secondFingerprint,
			)
		}
		if calls := compiler.compileCalls.Load(); calls != 0 {
			t.Fatalf("Compiler calls = %d, want 0", calls)
		}
	})
}

func addLifecycleBuilderNode(
	builder *Builder[constructionCondition, constructionExpression],
	cursor *lifecycleFuzzCursor,
	depth int,
) {
	raw, ok := cursor.take()
	if !ok {
		return
	}
	value := int(int8(raw))
	switch raw % 16 {
	case 0:
		builder.EQ("field", value)
	case 1:
		builder.NEQ("field", value)
	case 2:
		builder.In("field", []int{})
	case 3:
		builder.NotIn("field", []int{})
	case 4:
		builder.In("field", []int{value, value, value + 1})
	case 5:
		first, second := value, value+1
		builder.In("field", []*int{&second, nil, &first, &second})
	case 6:
		builder.In("field", []*int{nil, nil})
	case 7:
		pointer := value
		values := []*int{&pointer}
		if raw&0x80 != 0 {
			values = append(values, nil)
		}
		builder.NotIn("field", values)
	case 8:
		if raw&0x20 == 0 {
			builder.IsNull("field")
		} else {
			builder.NotNull("field")
		}
	case 9:
		builder.Between("field", value, value+1)
	case 10:
		switch raw & 0x60 {
		case 0:
			builder.Contains("field", fmt.Sprintf("value-%d", value))
		case 0x20:
			builder.HasPrefix("field", fmt.Sprintf("value-%d", value))
		default:
			builder.HasSuffix("field", fmt.Sprintf("value-%d", value))
		}
	case 11:
		if depth >= 6 {
			builder.EQ("field", value)
			return
		}
		addLifecycleBuilderGroup(builder, cursor, raw, depth)
	case 12:
		addLifecycleEmptyBuilderGroup(builder, raw)
	case 13:
		builder.Expr(constructionExpression{name: fmt.Sprintf("expression-%d", value)})
	case 14:
		builder.Native(constructionCondition{fmt.Sprintf("native-%d", value)})
	case 15:
		if raw&0x40 != 0 {
			builder.EQ(nil, value, func(int) bool { return false })
		} else if raw&0x20 != 0 {
			builder.EQ(nil, value)
		} else {
			builder.EQ("field", value, nil)
		}
	}
}

func addLifecycleGroupNode(
	group *Group[constructionExpression],
	cursor *lifecycleFuzzCursor,
	depth int,
) {
	raw, ok := cursor.take()
	if !ok {
		return
	}
	value := int(int8(raw))
	switch raw % 16 {
	case 0:
		group.EQ("field", value)
	case 1:
		group.NEQ("field", value)
	case 2:
		group.In("field", []int{})
	case 3:
		group.NotIn("field", []int{})
	case 4:
		group.In("field", []int{value, value, value + 1})
	case 5:
		first, second := value, value+1
		group.In("field", []*int{&second, nil, &first, &second})
	case 6:
		group.In("field", []*int{nil, nil})
	case 7:
		pointer := value
		values := []*int{&pointer}
		if raw&0x80 != 0 {
			values = append(values, nil)
		}
		group.NotIn("field", values)
	case 8:
		if raw&0x20 == 0 {
			group.IsNull("field")
		} else {
			group.NotNull("field")
		}
	case 9:
		group.Between("field", value, value+1)
	case 10:
		switch raw & 0x60 {
		case 0:
			group.Contains("field", fmt.Sprintf("value-%d", value))
		case 0x20:
			group.HasPrefix("field", fmt.Sprintf("value-%d", value))
		default:
			group.HasSuffix("field", fmt.Sprintf("value-%d", value))
		}
	case 11:
		if depth >= 6 {
			group.EQ("field", value)
			return
		}
		addLifecycleNestedGroup(group, cursor, raw, depth)
	case 12:
		addLifecycleEmptyNestedGroup(group, raw)
	case 13, 14:
		group.Expr(constructionExpression{name: fmt.Sprintf("expression-%d", value)})
	case 15:
		if raw&0x40 != 0 {
			group.EQ(nil, value, func(int) bool { return false })
		} else if raw&0x20 != 0 {
			group.EQ(nil, value)
		} else {
			group.EQ("field", value, nil)
		}
	}
}

func addLifecycleBuilderGroup(
	builder *Builder[constructionCondition, constructionExpression],
	cursor *lifecycleFuzzCursor,
	raw byte,
	depth int,
) {
	logic := Logic((raw/16)%4 + 1)
	childCount := int(raw>>6) + 1
	scope := func(group *Group[constructionExpression]) {
		for index := 0; index < childCount && cursor.index < len(cursor.data); index++ {
			addLifecycleGroupNode(group, cursor, depth+1)
		}
	}
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

func addLifecycleNestedGroup(
	group *Group[constructionExpression],
	cursor *lifecycleFuzzCursor,
	raw byte,
	depth int,
) {
	logic := Logic((raw/16)%4 + 1)
	childCount := int(raw>>6) + 1
	scope := func(nested *Group[constructionExpression]) {
		for index := 0; index < childCount && cursor.index < len(cursor.data); index++ {
			addLifecycleGroupNode(nested, cursor, depth+1)
		}
	}
	switch logic {
	case LogicAllOf:
		group.AllOf(scope)
	case LogicAnyOf:
		group.AnyOf(scope)
	case LogicNoneOf:
		group.NoneOf(scope)
	case LogicNotAllOf:
		group.NotAllOf(scope)
	}
}

func addLifecycleEmptyBuilderGroup(
	builder *Builder[constructionCondition, constructionExpression],
	raw byte,
) {
	logic := Logic((raw/16)%4 + 1)
	var scope Scope[constructionExpression] = func(*Group[constructionExpression]) {}
	if raw&0x80 != 0 {
		scope = nil
	}
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

func addLifecycleEmptyNestedGroup(
	group *Group[constructionExpression],
	raw byte,
) {
	logic := Logic((raw/16)%4 + 1)
	var scope Scope[constructionExpression] = func(*Group[constructionExpression]) {}
	if raw&0x80 != 0 {
		scope = nil
	}
	switch logic {
	case LogicAllOf:
		group.AllOf(scope)
	case LogicAnyOf:
		group.AnyOf(scope)
	case LogicNoneOf:
		group.NoneOf(scope)
	case LogicNotAllOf:
		group.NotAllOf(scope)
	}
}

func lifecycleCapabilities(mask uint32) Capabilities {
	operators := [...]Operator{
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
	}
	operatorValues := make([]Operator, 0, len(operators))
	for index, operator := range operators {
		if mask&(1<<index) != 0 {
			operatorValues = append(operatorValues, operator)
		}
	}
	featureValues := make([]Feature, 0, 2)
	if mask&(1<<14) != 0 {
		featureValues = append(featureValues, FeatureNativeCondition)
	}
	if mask&(1<<15) != 0 {
		featureValues = append(featureValues, FeatureNativeExpression)
	}
	return Capabilities{
		Operators: NewOperatorSet(operatorValues...),
		Features:  NewFeatureSet(featureValues...),
	}
}

func firstLifecycleMissing[C, E any](
	t *testing.T,
	root NodeView[C, E],
	capabilities Capabilities,
) *lifecycleMissing {
	t.Helper()
	stack := []NodeView[C, E]{root}
	for len(stack) != 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if group, ok := current.AsGroup(); ok {
			for index := group.ChildCount() - 1; index >= 0; index-- {
				child, childOK := group.Child(index)
				if !childOK {
					t.Fatalf("Child(%d) failed within bounds", index)
				}
				stack = append(stack, child)
			}
			continue
		}

		operator := Operator(0)
		switch {
		case current.Kind() == KindComparison:
			view, _ := current.AsComparison()
			operator = view.Operator()
		case current.Kind() == KindMembership:
			view, _ := current.AsMembership()
			operator = view.Operator()
		case current.Kind() == KindRange:
			view, _ := current.AsRange()
			operator = view.Operator()
		case current.Kind() == KindNull:
			view, _ := current.AsNull()
			operator = view.Operator()
		case current.Kind() == KindText:
			view, _ := current.AsText()
			operator = view.Operator()
		}
		if operator != 0 && !capabilities.Operators.Has(operator) {
			return &lifecycleMissing{
				code:     CodeUnsupportedOperator,
				path:     current.Path().String(),
				origin:   current.Origin(),
				operator: operator,
			}
		}
		if _, ok := current.AsNativeCondition(); ok &&
			!capabilities.Features.Has(FeatureNativeCondition) {
			return &lifecycleMissing{
				code:    CodeUnsupportedFeature,
				path:    current.Path().String(),
				origin:  current.Origin(),
				feature: FeatureNativeCondition,
			}
		}
		if _, ok := current.AsNativeExpression(); ok &&
			!capabilities.Features.Has(FeatureNativeExpression) {
			return &lifecycleMissing{
				code:    CodeUnsupportedFeature,
				path:    current.Path().String(),
				origin:  current.Origin(),
				feature: FeatureNativeExpression,
			}
		}
	}
	return nil
}

func assertLifecycleMissing(
	t *testing.T,
	err error,
	want lifecycleMissing,
) {
	t.Helper()
	if err == nil || !errors.Is(err, ErrCompile) {
		t.Fatal("preflight error does not match ErrCompile")
	}
	classification := ErrUnsupportedOperator
	if want.code == CodeUnsupportedFeature {
		classification = ErrUnsupportedFeature
	}
	if !errors.Is(err, classification) {
		t.Fatal("preflight error does not match its specific classification")
	}
	var compileError *Error
	if !errors.As(err, &compileError) || compileError == nil {
		t.Fatalf("preflight error type = %T, want *Error", err)
	}
	if compileError.Code != want.code ||
		compileError.Phase != PhasePreflight ||
		compileError.Path.String() != want.path ||
		compileError.Origin != want.origin ||
		compileError.Operator != want.operator ||
		compileError.Feature != want.feature {
		t.Fatalf(
			"preflight metadata = (%s, %s, %s, %#v, %s, %s), want (%s, %s, %s, %#v, %s, %s)",
			compileError.Code,
			compileError.Phase,
			compileError.Path.String(),
			compileError.Origin,
			compileError.Operator,
			compileError.Feature,
			want.code,
			PhasePreflight,
			want.path,
			want.origin,
			want.operator,
			want.feature,
		)
	}
}
