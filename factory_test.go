package weave

import (
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
)

type factoryTestCompiler struct {
	capabilities      Capabilities
	result            string
	err               error
	capabilitiesCalls atomic.Int64
	compileCalls      atomic.Int64
}

func (c *factoryTestCompiler) Compile(Predicate[string, string]) (string, error) {
	c.compileCalls.Add(1)
	return c.result, c.err
}

func (c *factoryTestCompiler) Capabilities() Capabilities {
	c.capabilitiesCalls.Add(1)
	return c.capabilities
}

var _ Compiler[string, string] = (*factoryTestCompiler)(nil)

func TestCompilerInterfaceShape(t *testing.T) {
	compilerType := reflect.TypeFor[Compiler[string, string]]()
	if compilerType.Kind() != reflect.Interface || compilerType.NumMethod() != 2 {
		t.Fatalf("Compiler shape = (%s, %d methods), want interface with 2 methods", compilerType.Kind(), compilerType.NumMethod())
	}
	for _, name := range []string{"Capabilities", "Compile"} {
		if _, present := compilerType.MethodByName(name); !present {
			t.Errorf("Compiler method %s is missing", name)
		}
	}
}

func TestNewFactoryRejectsNilCompilers(t *testing.T) {
	tests := []struct {
		name string
		new  func()
	}{
		{
			name: "nil interface",
			new: func() {
				NewFactory[string, string](nil)
			},
		},
		{
			name: "typed nil pointer",
			new: func() {
				var compiler *factoryTestCompiler
				NewFactory[string, string](compiler)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deferred := false
			func() {
				deferred = true
				defer func() {
					panicValue := recover()
					if panicValue != "weave: nil compiler" {
						t.Fatalf("panic = %#v, want nil compiler panic", panicValue)
					}
				}()
				test.new()
				deferred = false
			}()
			if !deferred {
				t.Fatal("NewFactory did not panic")
			}
		})
	}
}

func TestFactoryCapturesCapabilitiesExactlyOnce(t *testing.T) {
	want := Capabilities{
		Operators: NewOperatorSet(OperatorEQ),
		Features:  NewFeatureSet(FeatureNativeExpression),
	}
	compiler := &factoryTestCompiler{
		capabilities: want,
		result:       "compiled",
	}
	factory := NewFactory[string, string](compiler)
	compiler.capabilities = Capabilities{}

	if got := factory.Capabilities(); got != want {
		t.Fatalf("Factory capabilities = %#v, want captured %#v", got, want)
	}
	returned := factory.Capabilities()
	returned.Operators = NewOperatorSet(OperatorGT)
	returned.Features = FeatureSet{}
	if got := factory.Capabilities(); got != want {
		t.Fatalf("Factory capabilities changed through returned value: %#v", got)
	}

	predicate, err := factory.New().EQ("field", 1).Expr("expression").Predicate()
	if err != nil {
		t.Fatalf("Predicate failed: %v", err)
	}
	compiled, err := factory.Compile(predicate)
	if err != nil || compiled != "compiled" {
		t.Fatalf("Compile = (%q, %v), want (compiled, nil)", compiled, err)
	}
	if calls := compiler.capabilitiesCalls.Load(); calls != 1 {
		t.Fatalf("Capabilities calls = %d, want 1", calls)
	}
}

func TestBuilderBuildMatchesPredicateThenCompile(t *testing.T) {
	compiler := &factoryTestCompiler{
		capabilities: Capabilities{
			Operators: NewOperatorSet(OperatorEQ),
		},
		result: "compiled",
	}
	factory := NewFactory[string, string](compiler)

	builder := factory.New().EQ("field", 1)
	built, err := builder.Build()
	if err != nil || built != "compiled" {
		t.Fatalf("Build = (%q, %v), want (compiled, nil)", built, err)
	}
	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate failed: %v", err)
	}
	compiled, err := factory.Compile(predicate)
	if err != nil || compiled != built {
		t.Fatalf("explicit Compile = (%q, %v), want (%q, nil)", compiled, err, built)
	}
	if calls := compiler.compileCalls.Load(); calls != 2 {
		t.Fatalf("Compile calls = %d, want 2", calls)
	}
}

func TestBuilderBuildStopsBeforeCompilerWhenPredicateFails(t *testing.T) {
	compiler := &factoryTestCompiler{
		capabilities: Capabilities{Operators: NewOperatorSet(OperatorEQ)},
		result:       "must not escape",
	}
	factory := NewFactory[string, string](compiler)

	compiled, err := factory.New().EQ(nil, 1).Build()
	if compiled != "" {
		t.Fatalf("Build result = %q, want zero", compiled)
	}
	if !errors.Is(err, ErrInvalidField) || errors.Is(err, ErrCompile) {
		t.Fatalf("Build error classifications are incorrect")
	}
	if calls := compiler.compileCalls.Load(); calls != 0 {
		t.Fatalf("Compile calls = %d, want 0", calls)
	}
}

func TestUnboundBuilderBuildReturnsCompileInvalidState(t *testing.T) {
	var builder Builder[string, string]
	builder.EQ("field", 1)

	compiled, err := builder.Build()
	if compiled != "" {
		t.Fatalf("Build result = %q, want zero", compiled)
	}
	compileError := requireFactoryCompileError(
		t,
		err,
		CodeInvalidState,
		PhasePreflight,
		ErrInvalidState,
	)
	if compileError.Path.SegmentCount() != 0 || compileError.Origin != (Origin{}) {
		t.Fatalf("invalid-state location = (%s, %#v), want empty", compileError.Path.String(), compileError.Origin)
	}
}

func TestFactoryRejectsInvalidAndForeignPredicates(t *testing.T) {
	compiler := &factoryTestCompiler{
		capabilities: Capabilities{Operators: NewOperatorSet(OperatorEQ)},
		result:       "compiled",
	}
	factory := NewFactory[string, string](compiler)
	foreignFactory := NewFactory[string, string](&factoryTestCompiler{
		capabilities: compiler.capabilities,
	})
	foreign, err := foreignFactory.New().EQ("field", 1).Predicate()
	if err != nil {
		t.Fatalf("foreign Predicate failed: %v", err)
	}

	for _, test := range []struct {
		name      string
		predicate Predicate[string, string]
	}{
		{name: "zero", predicate: Predicate[string, string]{}},
		{name: "foreign", predicate: foreign},
	} {
		t.Run(test.name, func(t *testing.T) {
			compiled, compileErr := factory.Compile(test.predicate)
			if compiled != "" {
				t.Fatalf("Compile result = %q, want zero", compiled)
			}
			requireFactoryCompileError(
				t,
				compileErr,
				CodeInvalidPredicate,
				PhasePreflight,
				ErrInvalidPredicate,
			)
		})
	}
	if calls := compiler.compileCalls.Load(); calls != 0 {
		t.Fatalf("Compile calls = %d, want 0", calls)
	}
}

func TestZeroFactoryLifecycle(t *testing.T) {
	var factory Factory[string, string]
	if got := factory.Capabilities(); got != (Capabilities{}) {
		t.Fatalf("zero Factory capabilities = %#v, want zero", got)
	}
	compiled, err := factory.Compile(Predicate[string, string]{})
	if compiled != "" {
		t.Fatalf("Compile result = %q, want zero", compiled)
	}
	requireFactoryCompileError(
		t,
		err,
		CodeInvalidState,
		PhasePreflight,
		ErrInvalidState,
	)
}

func TestCapabilityPreflightUsesFirstMissingNodeInStableDFS(t *testing.T) {
	compiler := &factoryTestCompiler{result: "must not escape"}
	factory := NewFactory[string, string](compiler)
	predicate, err := factory.New().
		GT("first", 1).
		AllOf(func(group *Group[string]) {
			group.EQ("nested", 2)
		}).
		EQ("last", 3).
		Predicate()
	if err != nil {
		t.Fatalf("Predicate failed: %v", err)
	}

	compiled, err := factory.Compile(predicate)
	if compiled != "" {
		t.Fatalf("Compile result = %q, want zero", compiled)
	}
	compileError := requireFactoryCompileError(
		t,
		err,
		CodeUnsupportedOperator,
		PhasePreflight,
		ErrUnsupportedOperator,
	)
	if compileError.Operator != OperatorGT ||
		compileError.Feature != 0 ||
		compileError.Origin != (Origin{Sequence: 1, Operator: OperatorGT}) ||
		compileError.Path.String() != "root.allOf[0].gt" {
		t.Fatalf(
			"first missing node = (%s, %#v, %s, %s)",
			compileError.Path.String(),
			compileError.Origin,
			compileError.Operator,
			compileError.Feature,
		)
	}
	if calls := compiler.compileCalls.Load(); calls != 0 {
		t.Fatalf("Compile calls = %d, want 0", calls)
	}
}

func TestCapabilityPreflightUsesTreeOrderAcrossRequirementKinds(t *testing.T) {
	compiler := &factoryTestCompiler{result: "must not escape"}
	factory := NewFactory[string, string](compiler)
	predicate, err := factory.New().
		Expr("first").
		EQ("second", 1).
		Predicate()
	if err != nil {
		t.Fatalf("Predicate failed: %v", err)
	}

	compiled, err := factory.Compile(predicate)
	if compiled != "" {
		t.Fatalf("Compile result = %q, want zero", compiled)
	}
	compileError := requireFactoryCompileError(
		t,
		err,
		CodeUnsupportedFeature,
		PhasePreflight,
		ErrUnsupportedFeature,
	)
	if compileError.Feature != FeatureNativeExpression ||
		compileError.Operator != 0 ||
		compileError.Origin != (Origin{Sequence: 1}) ||
		compileError.Path.String() != "root.allOf[0].native_expression" {
		t.Fatal("preflight did not report the first missing AST node")
	}
	if calls := compiler.compileCalls.Load(); calls != 0 {
		t.Fatalf("Compile calls = %d, want 0", calls)
	}
}

func TestCapabilityPreflightUsesNormalizedConstantRequirements(t *testing.T) {
	compiler := &factoryTestCompiler{result: "compiled"}
	factory := NewFactory[string, string](compiler)
	tests := []struct {
		name string
		add  func(*Builder[string, string])
	}{
		{
			name: "empty In",
			add: func(builder *Builder[string, string]) {
				builder.In("field", []int{})
			},
		},
		{
			name: "empty NotIn",
			add: func(builder *Builder[string, string]) {
				builder.NotIn("field", []int{})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			builder := factory.New()
			test.add(builder)
			predicate, err := builder.Predicate()
			if err != nil {
				t.Fatalf("Predicate failed: %v", err)
			}
			if predicate.Requirements() != (Requirements{}) {
				t.Fatalf("constant Predicate requirements = %#v, want zero", predicate.Requirements())
			}
			compiled, err := factory.Compile(predicate)
			if err != nil || compiled != "compiled" {
				t.Fatalf("Compile = (%q, %v), want constant success", compiled, err)
			}
		})
	}
	if calls := compiler.compileCalls.Load(); calls != int64(len(tests)) {
		t.Fatalf("Compile calls = %d, want %d", calls, len(tests))
	}
}

func TestCapabilityPreflightLocatesNativeFeatures(t *testing.T) {
	tests := []struct {
		name       string
		add        func(*Builder[string, string])
		feature    Feature
		path       string
		sequence   uint64
		capability Capabilities
	}{
		{
			name: "root native condition",
			add: func(builder *Builder[string, string]) {
				builder.Native("native")
			},
			feature:  FeatureNativeCondition,
			path:     "root.allOf[0].native_condition",
			sequence: 1,
		},
		{
			name: "nested native expression",
			add: func(builder *Builder[string, string]) {
				builder.AllOf(func(group *Group[string]) {
					group.Expr("expression")
				})
			},
			feature:  FeatureNativeExpression,
			path:     "root.allOf[0].allOf[0].native_expression",
			sequence: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			compiler := &factoryTestCompiler{
				capabilities: test.capability,
				result:       "must not escape",
			}
			factory := NewFactory[string, string](compiler)
			builder := factory.New()
			test.add(builder)
			predicate, err := builder.Predicate()
			if err != nil {
				t.Fatalf("Predicate failed: %v", err)
			}

			compiled, err := factory.Compile(predicate)
			if compiled != "" {
				t.Fatalf("Compile result = %q, want zero", compiled)
			}
			compileError := requireFactoryCompileError(
				t,
				err,
				CodeUnsupportedFeature,
				PhasePreflight,
				ErrUnsupportedFeature,
			)
			if compileError.Feature != test.feature ||
				compileError.Operator != 0 ||
				compileError.Path.String() != test.path ||
				compileError.Origin.Sequence != test.sequence {
				t.Fatalf(
					"missing feature location = (%s, %#v, %s, %s)",
					compileError.Path.String(),
					compileError.Origin,
					compileError.Operator,
					compileError.Feature,
				)
			}
			if calls := compiler.compileCalls.Load(); calls != 0 {
				t.Fatalf("Compile calls = %d, want 0", calls)
			}
		})
	}
}

func TestCompilePreflightValidatesStructureBeforeCapabilities(t *testing.T) {
	t.Run("node ownership and path", func(t *testing.T) {
		compiler := &factoryTestCompiler{
			capabilities: Capabilities{Operators: NewOperatorSet(OperatorEQ)},
			result:       "must not escape",
		}
		factory := NewFactory[string, string](compiler)
		predicate, err := factory.New().EQ("field", 1).Predicate()
		if err != nil {
			t.Fatalf("Predicate failed: %v", err)
		}
		comparison := predicate.state.root.children[0].(*comparisonNode)
		comparison.owner = &predicateState{}

		compiled, err := factory.Compile(predicate)
		if compiled != "" {
			t.Fatalf("Compile result = %q, want zero", compiled)
		}
		compileError := requireFactoryCompileError(
			t,
			err,
			CodeInvalidPredicate,
			PhasePreflight,
			ErrInvalidPredicate,
		)
		if compileError.Path.String() != "root.allOf[0].eq" ||
			compileError.Origin != (Origin{Sequence: 1, Operator: OperatorEQ}) ||
			compileError.Operator != OperatorEQ {
			t.Fatalf("structural error location is inaccurate")
		}
		if calls := compiler.compileCalls.Load(); calls != 0 {
			t.Fatalf("Compile calls = %d, want 0", calls)
		}
	})

	t.Run("requirements mismatch", func(t *testing.T) {
		compiler := &factoryTestCompiler{
			capabilities: Capabilities{Operators: NewOperatorSet(OperatorEQ)},
			result:       "must not escape",
		}
		factory := NewFactory[string, string](compiler)
		predicate, err := factory.New().EQ("field", 1).Predicate()
		if err != nil {
			t.Fatalf("Predicate failed: %v", err)
		}
		predicate.state.requirements = Requirements{}

		_, err = factory.Compile(predicate)
		compileError := requireFactoryCompileError(
			t,
			err,
			CodeInvalidPredicate,
			PhasePreflight,
			ErrInvalidPredicate,
		)
		if compileError.Path.String() != "root.allOf" {
			t.Fatalf("requirements error path = %s, want root.allOf", compileError.Path.String())
		}
		if calls := compiler.compileCalls.Load(); calls != 0 {
			t.Fatalf("Compile calls = %d, want 0", calls)
		}
	})
}

func TestCompilePreflightDepthBoundary(t *testing.T) {
	compiler := &factoryTestCompiler{
		capabilities: Capabilities{Operators: NewOperatorSet(OperatorEQ)},
		result:       "compiled",
	}
	factory := NewFactory[string, string](compiler)

	atLimit := makeDeepCompilePredicate(factory.domain, MaxPredicateDepth)
	compiled, err := factory.Compile(atLimit)
	if err != nil || compiled != "compiled" {
		t.Fatalf("depth %d Compile = (%q, %v), want success", MaxPredicateDepth, compiled, err)
	}

	overLimit := makeDeepCompilePredicate(factory.domain, MaxPredicateDepth+1)
	compiled, err = factory.Compile(overLimit)
	if compiled != "" {
		t.Fatalf("depth %d result = %q, want zero", MaxPredicateDepth+1, compiled)
	}
	compileError := requireFactoryCompileError(
		t,
		err,
		CodeDepthLimit,
		PhasePreflight,
		ErrDepthLimit,
	)
	if compileError.Operator != OperatorEQ ||
		compileError.Origin.Sequence != MaxPredicateDepth+1 {
		t.Fatalf("depth error metadata = (%s, %#v)", compileError.Operator, compileError.Origin)
	}
	if calls := compiler.compileCalls.Load(); calls != 1 {
		t.Fatalf("Compile calls = %d, want 1", calls)
	}
}

func TestCompilePreflightRejectsNestedNativeCondition(t *testing.T) {
	compiler := &factoryTestCompiler{
		capabilities: Capabilities{
			Features: NewFeatureSet(FeatureNativeCondition),
		},
		result: "must not escape",
	}
	factory := NewFactory[string, string](compiler)
	predicate := makeNestedNativeCompilePredicate(factory.domain)

	compiled, err := factory.Compile(predicate)
	if compiled != "" {
		t.Fatalf("Compile result = %q, want zero", compiled)
	}
	compileError := requireFactoryCompileError(
		t,
		err,
		CodeNonNestableNative,
		PhasePreflight,
		ErrNonNestableNative,
	)
	if compileError.Path.String() != "root.allOf[0].allOf[0].native_condition" ||
		compileError.Origin.Sequence != 2 ||
		compileError.Feature != FeatureNativeCondition ||
		compileError.Operator != 0 {
		t.Fatalf("nested native error metadata is inaccurate")
	}
	if calls := compiler.compileCalls.Load(); calls != 0 {
		t.Fatalf("Compile calls = %d, want 0", calls)
	}
}

func makeDeepCompilePredicate(
	domain *predicateDomain,
	depth int,
) Predicate[string, string] {
	state := &predicateState{
		seal:         validPredicateSeal,
		domain:       domain,
		requirements: Requirements{Operators: NewOperatorSet(OperatorEQ)},
	}
	rootPath := newNodePath(newRootPathSegment(LogicAllOf))
	root := &groupNode{
		nodeBase: nodeBase{owner: state, path: rootPath},
		logic:    LogicAllOf,
	}
	state.root = root
	parent := root
	parentPath := rootPath
	for currentDepth := 1; currentDepth < depth; currentDepth++ {
		origin := Origin{Sequence: uint64(currentDepth)}
		path := appendSnapshotPath(
			parentPath,
			0,
			KindGroup,
			LogicAllOf,
			0,
		)
		group := &groupNode{
			nodeBase: nodeBase{origin: origin, owner: state, path: path},
			logic:    LogicAllOf,
		}
		parent.children = []node{group}
		parent = group
		parentPath = path
	}
	origin := Origin{Sequence: uint64(depth), Operator: OperatorEQ}
	path := appendSnapshotPath(
		parentPath,
		0,
		KindComparison,
		0,
		OperatorEQ,
	)
	parent.children = []node{&comparisonNode{
		nodeBase:  nodeBase{origin: origin, owner: state, path: path},
		operator:  OperatorEQ,
		field:     "field",
		value:     1,
		valueType: reflect.TypeFor[int](),
	}}
	return Predicate[string, string]{state: state}
}

func makeNestedNativeCompilePredicate(
	domain *predicateDomain,
) Predicate[string, string] {
	state := &predicateState{
		seal:   validPredicateSeal,
		domain: domain,
		requirements: Requirements{
			Features: NewFeatureSet(FeatureNativeCondition),
		},
	}
	rootPath := newNodePath(newRootPathSegment(LogicAllOf))
	root := &groupNode{
		nodeBase: nodeBase{owner: state, path: rootPath},
		logic:    LogicAllOf,
	}
	state.root = root
	groupPath := appendSnapshotPath(rootPath, 0, KindGroup, LogicAllOf, 0)
	group := &groupNode{
		nodeBase: nodeBase{
			origin: Origin{Sequence: 1},
			owner:  state,
			path:   groupPath,
		},
		logic: LogicAllOf,
	}
	root.children = []node{group}
	nativePath := appendSnapshotPath(
		groupPath,
		0,
		KindNativeCondition,
		0,
		0,
	)
	group.children = []node{&nativeConditionNode[string]{
		nodeBase: nodeBase{
			origin: Origin{Sequence: 2},
			owner:  state,
			path:   nativePath,
		},
		condition: "native",
	}}
	return Predicate[string, string]{state: state}
}

func requireFactoryCompileError(
	t *testing.T,
	err error,
	code ErrorCode,
	phase ErrorPhase,
	classification error,
) *Error {
	t.Helper()
	if err == nil {
		t.Fatal("Compile error is nil")
	}
	if !errors.Is(err, ErrCompile) {
		t.Fatal("error does not match ErrCompile")
	}
	if classification != nil && !errors.Is(err, classification) {
		t.Fatalf("error does not match classification %q", classification.Error())
	}
	var compileError *Error
	if !errors.As(err, &compileError) || compileError == nil {
		t.Fatalf("error type = %T, want *Error", err)
	}
	if compileError.Code != code || compileError.Phase != phase {
		t.Fatalf(
			"error code/phase = (%s, %s), want (%s, %s)",
			compileError.Code,
			compileError.Phase,
			code,
			phase,
		)
	}
	return compileError
}
