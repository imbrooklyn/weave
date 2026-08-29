package compilertest

import (
	"errors"
	"slices"
	"testing"

	"github.com/imbrooklyn/weave"
)

// Fields contains the Adapter field payloads used by the shared fixture.
// NullableNumber and NullableText must preserve the three states encoded by
// Record when Harness.DistinguishesMissing is true.
type Fields struct {
	// Number identifies Record.Number.
	Number any
	// Text identifies Record.Text.
	Text any
	// NullableNumber identifies Record.NullableNumber and its presence state.
	NullableNumber any
	// NullableText identifies Record.NullableText and its presence state.
	NullableText any
	// EqualityOnlyText identifies Record.Text through a descriptor that supports
	// only equality, membership, and null operators.
	EqualityOnlyText any
}

// Harness connects the shared semantic suite to one Compiler and execution
// environment. Resolver exposes field-level capabilities. Execute applies a
// compiled condition to the fixture and returns matching stable Record IDs in
// any order and must be safe for concurrent calls. InspectCondition is an
// optional backend-owned structural and safety assertion; the suite never
// interprets C itself. NativeCondition and NativeExpression are optional
// constructors for legal Adapter-native filters that match the supplied IDs.
//
// A Harness that sets DistinguishesMissing to false must materialize missing
// nullable fixture values as explicit null. The suite then omits cases whose
// expected result specifically distinguishes those two states.
type Harness[C, E any] struct {
	// Factory owns the Compiler under test.
	Factory *weave.Factory[C, E]
	// Fields binds Adapter field payloads to the shared fixture.
	Fields Fields
	// Resolver exposes field-level capability discovery for the same immutable
	// Compiler configuration bound to Factory.
	Resolver weave.FieldCapabilityResolver
	// Execute runs a compiled condition and returns matching stable IDs.
	Execute func(C) ([]string, error)
	// InspectCondition optionally checks a compiled condition for one named
	// semantic case. It may validate backend-specific structure, parameter
	// binding, or other safety invariants without exposing those details to
	// compilertest. It must treat condition as read-only.
	InspectCondition func(caseName string, condition C) error
	// NativeCondition optionally constructs a root native condition matching IDs.
	NativeCondition func(ids []string) C
	// NativeExpression optionally constructs a nestable native expression matching IDs.
	NativeExpression func(ids []string) E
	// NilLikeNativeCondition optionally returns a typed nil-like C that the
	// Compiler must reject as an invalid Native payload.
	NilLikeNativeCondition func() C
	// NilLikeNativeExpression optionally returns a typed nil-like E that the
	// Compiler must reject as an invalid Expr payload.
	NilLikeNativeExpression func() E
	// DistinguishesMissing reports whether the harness preserves missing as a
	// state distinct from explicit null.
	DistinguishesMissing bool
}

type semanticCase[C, E any] struct {
	name                     string
	wantIDs                  []string
	missingCollapsedIDs      []string
	supportsMissingCollapsed bool
	requiresMissing          bool
	fieldOperators           []fieldOperator
	features                 []weave.Feature
	build                    func(*weave.Builder[C, E], Harness[C, E])
}

type fieldOperator struct {
	field    any
	operator weave.Operator
}

// Scenario is one read-only canonical semantic case. Scenario values are
// produced by Scenarios; the zero value is invalid. The predicate construction
// callback and expected match set remain owned by compilertest so executable
// examples cannot fork the Adapter contract cases.
type Scenario[C, E any] struct {
	name                    string
	expectedIDs             []string
	requiresDistinctMissing bool
	usesMissingCollapsed    bool
	build                   func(*weave.Builder[C, E])
}

// Name returns the stable human-readable scenario name.
func (scenario Scenario[C, E]) Name() string {
	return scenario.name
}

// ExpectedIDs returns an independent copy of the canonical record-ID match
// set selected for the Harness storage semantics passed to Scenarios. Callers
// may sort or otherwise modify the returned slice.
func (scenario Scenario[C, E]) ExpectedIDs() []string {
	return slices.Clone(scenario.expectedIDs)
}

// RequiresDistinctMissing reports whether this scenario relies on missing and
// explicit null remaining observably distinct.
func (scenario Scenario[C, E]) RequiresDistinctMissing() bool {
	return scenario.requiresDistinctMissing
}

// UsesMissingCollapsedMatchSet reports whether ExpectedIDs reflects a Harness
// that materializes missing fixture values as explicit null. Differential
// runners can use this metadata to distinguish a canonical storage adjustment
// from a semantic mismatch with a missing-aware reference backend.
func (scenario Scenario[C, E]) UsesMissingCollapsedMatchSet() bool {
	return scenario.usesMissingCollapsed
}

// Build creates and compiles this scenario through factory. It returns an
// error for a nil Factory or an invalid zero Scenario.
func (scenario Scenario[C, E]) Build(factory *weave.Factory[C, E]) (C, error) {
	var zero C
	if factory == nil {
		return zero, errors.New("compilertest: nil Factory")
	}
	if scenario.build == nil {
		return zero, errors.New("compilertest: invalid Scenario")
	}
	builder := factory.New()
	scenario.build(builder)
	return builder.Build()
}

// Scenarios returns the canonical backend-neutral semantic match-set cases for
// harness. When Harness.Factory is non-nil, cases outside its global
// capabilities are omitted. When Harness.Resolver is also non-nil, a case is
// omitted when any standard operator is not applicable to its bound field.
// Native and Expr cases additionally require their corresponding Harness
// constructors. For a Harness that materializes missing as explicit null,
// null-sensitive scenarios expose their canonical collapsed match sets and
// only a scenario that must identify missing remains unavailable. Every call
// returns independent Scenario metadata; callers cannot replace or mutate the
// package-owned definitions.
func Scenarios[C, E any](harness Harness[C, E]) []Scenario[C, E] {
	tests := semanticCases(harness)
	scenarios := make([]Scenario[C, E], 0, len(tests))
	for _, test := range tests {
		test := test
		if !semanticCaseApplicable(test, harness) {
			continue
		}
		expectedIDs := test.wantIDs
		requiresDistinctMissing := test.requiresMissing
		usesMissingCollapsed := false
		if !harness.DistinguishesMissing && test.supportsMissingCollapsed {
			expectedIDs = test.missingCollapsedIDs
			requiresDistinctMissing = false
			usesMissingCollapsed = true
		}
		scenarios = append(scenarios, Scenario[C, E]{
			name:                    test.name,
			expectedIDs:             slices.Clone(expectedIDs),
			requiresDistinctMissing: requiresDistinctMissing,
			usesMissingCollapsed:    usesMissingCollapsed,
			build: func(builder *weave.Builder[C, E]) {
				test.build(builder, harness)
			},
		})
	}
	return scenarios
}

func semanticCaseApplicable[C, E any](
	test semanticCase[C, E],
	harness Harness[C, E],
) bool {
	if harness.Factory == nil {
		return true
	}
	if !harness.Factory.Capabilities().Supports(semanticCaseRequirements(test)) {
		return false
	}
	if isNilLike(harness.Resolver) {
		return true
	}
	for _, use := range test.fieldOperators {
		capabilities, err := harness.Resolver.CapabilitiesFor(use.field)
		if err != nil {
			// Keep the case so executable callers observe the invalid field instead
			// of silently losing coverage. Run reports the resolver error in its
			// capability contract.
			continue
		}
		if !capabilities.Operators.Has(use.operator) {
			return false
		}
	}
	return true
}

func semanticCaseRequirements[C, E any](test semanticCase[C, E]) weave.Requirements {
	operators := make([]weave.Operator, 0, len(test.fieldOperators))
	for _, use := range test.fieldOperators {
		operators = append(operators, use.operator)
	}
	return weave.Requirements{
		Operators: weave.NewOperatorSet(operators...),
		Features:  weave.NewFeatureSet(test.features...),
	}
}

func uses(field any, operators ...weave.Operator) []fieldOperator {
	result := make([]fieldOperator, len(operators))
	for index, operator := range operators {
		result[index] = fieldOperator{field: field, operator: operator}
	}
	return result
}

// Run executes the shared Compiler semantic suite. Each case builds through
// Harness.Factory, invokes Harness.Execute, and compares canonical record-ID
// sets rather than backend output text. Cases are selected from the Factory's
// global capabilities and the Resolver's field applicability. Run also checks
// that every unsupported standard operator and native feature is rejected by
// Factory preflight with a zero condition and a structured error.
func Run[C, E any](t *testing.T, harness Harness[C, E]) {
	t.Helper()
	if harness.Factory == nil {
		t.Fatal("compilertest: nil Factory")
	}
	if harness.Execute == nil {
		t.Fatal("compilertest: nil Execute callback")
	}
	if isNilLike(harness.Resolver) {
		t.Fatal("compilertest: nil Resolver")
	}

	for _, scenario := range Scenarios(harness) {
		t.Run(scenario.Name(), func(t *testing.T) {
			if scenario.RequiresDistinctMissing() && !harness.DistinguishesMissing {
				t.Skip("backend profile does not distinguish missing from explicit null")
			}

			condition, err := scenario.Build(harness.Factory)
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if harness.InspectCondition != nil {
				if err := harness.InspectCondition(scenario.Name(), condition); err != nil {
					t.Fatalf("InspectCondition() error = %v", err)
				}
			}
			ids, err := harness.Execute(condition)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			got := canonicalIDs(ids)
			want := canonicalIDs(scenario.ExpectedIDs())
			if !slices.Equal(got, want) {
				t.Fatalf("matching IDs = %v, want %v", got, want)
			}
		})
	}
	runCompilerContract(t, harness)
}

func semanticCases[C, E any](harness Harness[C, E]) []semanticCase[C, E] {
	fields := harness.Fields
	tests := []semanticCase[C, E]{
		{
			name:    "constant true root",
			wantIDs: allRecordIDs(),
			build:   func(*weave.Builder[C, E], Harness[C, E]) {},
		},
		{
			name:    "constant true empty all",
			wantIDs: allRecordIDs(),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.AllOf(func(*weave.Group[E]) {})
			},
		},
		{
			name:    "constant false empty any",
			wantIDs: nil,
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.AnyOf(func(*weave.Group[E]) {})
			},
		},
		{
			name:    "constant true empty none",
			wantIDs: allRecordIDs(),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.NoneOf(func(*weave.Group[E]) {})
			},
		},
		{
			name:    "constant false empty not all",
			wantIDs: nil,
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.NotAllOf(func(*weave.Group[E]) {})
			},
		},
		{
			name:    "scalar equality",
			wantIDs: []string{"r03"},
			fieldOperators: uses(
				fields.Number,
				weave.OperatorEQ,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.EQ(fields.Number, int64(3))
			},
		},
		{
			name:    "scalar literal text",
			wantIDs: []string{"r03", "r04"},
			fieldOperators: uses(
				fields.Text,
				weave.OperatorContains,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.Contains(fields.Text, "prefix")
			},
		},
		{
			name:    "equality",
			wantIDs: []string{"r02", "r06"},
			fieldOperators: uses(
				fields.NullableNumber,
				weave.OperatorEQ,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.EQ(fields.NullableNumber, int64(2))
			},
		},
		{
			name:    "inequality excludes null and missing",
			wantIDs: []string{"r01", "r05"},
			fieldOperators: uses(
				fields.NullableNumber,
				weave.OperatorNEQ,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.NEQ(fields.NullableNumber, int64(2))
			},
		},
		{
			name:    "less than",
			wantIDs: []string{"r01"},
			fieldOperators: uses(
				fields.NullableNumber,
				weave.OperatorLT,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.LT(fields.NullableNumber, int64(2))
			},
		},
		{
			name:    "less than or equal",
			wantIDs: []string{"r01", "r02", "r06"},
			fieldOperators: uses(
				fields.NullableNumber,
				weave.OperatorLTE,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.LTE(fields.NullableNumber, int64(2))
			},
		},
		{
			name:    "greater than",
			wantIDs: []string{"r05"},
			fieldOperators: uses(
				fields.NullableNumber,
				weave.OperatorGT,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.GT(fields.NullableNumber, int64(2))
			},
		},
		{
			name:    "greater than or equal",
			wantIDs: []string{"r02", "r05", "r06"},
			fieldOperators: uses(
				fields.NullableNumber,
				weave.OperatorGTE,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.GTE(fields.NullableNumber, int64(2))
			},
		},
		{
			name:    "membership preserves set semantics",
			wantIDs: []string{"r02", "r05", "r06"},
			fieldOperators: uses(
				fields.NullableNumber,
				weave.OperatorIn,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.In(fields.NullableNumber, []int64{2, 2, 5})
			},
		},
		{
			name:    "negative membership excludes null and missing",
			wantIDs: []string{"r01"},
			fieldOperators: uses(
				fields.NullableNumber,
				weave.OperatorNotIn,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.NotIn(fields.NullableNumber, []int64{2, 2, 5})
			},
		},
		{
			name:    "inclusive range",
			wantIDs: []string{"r02", "r06"},
			fieldOperators: uses(
				fields.NullableNumber,
				weave.OperatorBetween,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.Between(fields.NullableNumber, int64(2), int64(4))
			},
		},
		{
			name:                     "explicit null only",
			wantIDs:                  []string{"r03"},
			missingCollapsedIDs:      []string{"r03", "r04"},
			supportsMissingCollapsed: true,
			requiresMissing:          true,
			fieldOperators: uses(
				fields.NullableNumber,
				weave.OperatorIsNull,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.IsNull(fields.NullableNumber)
			},
		},
		{
			name:    "not null means value",
			wantIDs: []string{"r01", "r02", "r05", "r06"},
			fieldOperators: uses(
				fields.NullableNumber,
				weave.OperatorNotNull,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.NotNull(fields.NullableNumber)
			},
		},
		{
			name:    "literal contains special characters",
			wantIDs: []string{"r02"},
			fieldOperators: uses(
				fields.NullableText,
				weave.OperatorContains,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.Contains(fields.NullableText, LiteralSpecialText)
			},
		},
		{
			name:    "literal prefix",
			wantIDs: []string{"r06"},
			fieldOperators: uses(
				fields.NullableText,
				weave.OperatorHasPrefix,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.HasPrefix(fields.NullableText, ".*")
			},
		},
		{
			name:    "literal suffix",
			wantIDs: []string{"r02"},
			fieldOperators: uses(
				fields.NullableText,
				weave.OperatorHasSuffix,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.HasSuffix(fields.NullableText, "\u4e16\u754c\nend")
			},
		},
		{
			name:    "root conjunction",
			wantIDs: []string{"r02", "r03", "r04"},
			fieldOperators: uses(
				fields.Number,
				weave.OperatorGTE,
				weave.OperatorLTE,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.GTE(fields.Number, int64(2)).
					LTE(fields.Number, int64(4))
			},
		},
		{
			name:    "all of",
			wantIDs: []string{"r02", "r03", "r04"},
			fieldOperators: uses(
				fields.Number,
				weave.OperatorGTE,
				weave.OperatorLTE,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.AllOf(func(group *weave.Group[E]) {
					group.GTE(fields.Number, int64(2)).
						LTE(fields.Number, int64(4))
				})
			},
		},
		{
			name:    "any of",
			wantIDs: []string{"r01", "r06"},
			fieldOperators: uses(
				fields.Number,
				weave.OperatorEQ,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.AnyOf(func(group *weave.Group[E]) {
					group.EQ(fields.Number, int64(1)).
						EQ(fields.Number, int64(6))
				})
			},
		},
		{
			name:    "none of is match-set complement",
			wantIDs: []string{"r01", "r03", "r04", "r05"},
			fieldOperators: uses(
				fields.NullableNumber,
				weave.OperatorEQ,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.NoneOf(func(group *weave.Group[E]) {
					group.EQ(fields.NullableNumber, int64(2))
				})
			},
		},
		{
			name:    "not all of is match-set complement",
			wantIDs: []string{"r01", "r06"},
			fieldOperators: uses(
				fields.Number,
				weave.OperatorGTE,
				weave.OperatorLTE,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.NotAllOf(func(group *weave.Group[E]) {
					group.GTE(fields.Number, int64(2)).
						LTE(fields.Number, int64(5))
				})
			},
		},
		{
			name:                     "nullable membership",
			wantIDs:                  []string{"r02", "r03", "r06"},
			missingCollapsedIDs:      []string{"r02", "r03", "r04", "r06"},
			supportsMissingCollapsed: true,
			requiresMissing:          true,
			fieldOperators: uses(
				fields.NullableNumber,
				weave.OperatorIn,
				weave.OperatorIsNull,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				two := int64(2)
				builder.In(fields.NullableNumber, []*int64{&two, nil, &two})
			},
		},
		{
			name:            "missing state",
			wantIDs:         []string{"r04"},
			requiresMissing: true,
			fieldOperators: uses(
				fields.NullableNumber,
				weave.OperatorIsNull,
				weave.OperatorNotNull,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.NoneOf(func(group *weave.Group[E]) {
					group.IsNull(fields.NullableNumber).
						NotNull(fields.NullableNumber)
				})
			},
		},
		{
			name:    "three-level logic nesting",
			wantIDs: []string{"r03", "r04", "r05", "r06"},
			fieldOperators: uses(
				fields.Number,
				weave.OperatorEQ,
				weave.OperatorGTE,
			),
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.AllOf(func(levelOne *weave.Group[E]) {
					levelOne.AnyOf(func(levelTwo *weave.Group[E]) {
						levelTwo.NoneOf(func(levelThree *weave.Group[E]) {
							levelThree.EQ(fields.Number, int64(1)).
								EQ(fields.Number, int64(2))
						}).EQ(fields.Number, int64(6))
					}).GTE(fields.Number, int64(3))
				})
			},
		},
	}

	if harness.NativeCondition != nil {
		tests = append(tests, semanticCase[C, E]{
			name:    "native condition in root conjunction",
			wantIDs: []string{"r04"},
			fieldOperators: uses(
				fields.Number,
				weave.OperatorGTE,
			),
			features: []weave.Feature{weave.FeatureNativeCondition},
			build: func(builder *weave.Builder[C, E], harness Harness[C, E]) {
				builder.Native(harness.NativeCondition([]string{"r02", "r04"})).
					GTE(fields.Number, int64(3))
			},
		})
	}
	if harness.NativeExpression != nil {
		tests = append(tests, semanticCase[C, E]{
			name:    "native expression inside group",
			wantIDs: []string{"r01", "r03", "r06"},
			fieldOperators: uses(
				fields.Number,
				weave.OperatorEQ,
			),
			features: []weave.Feature{weave.FeatureNativeExpression},
			build: func(builder *weave.Builder[C, E], harness Harness[C, E]) {
				builder.AnyOf(func(group *weave.Group[E]) {
					group.Expr(harness.NativeExpression([]string{"r01", "r03"})).
						EQ(fields.Number, int64(6))
				})
			},
		})
	}
	return tests
}

func allRecordIDs() []string {
	records := Records()
	ids := make([]string, len(records))
	for index := range records {
		ids[index] = records[index].ID
	}
	return ids
}

func canonicalIDs(ids []string) []string {
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	canonical := make([]string, 0, len(set))
	for id := range set {
		canonical = append(canonical, id)
	}
	slices.Sort(canonical)
	return canonical
}
