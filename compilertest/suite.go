package compilertest

import (
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
	name            string
	wantIDs         []string
	requiresMissing bool
	build           func(*weave.Builder[C, E], Harness[C, E])
}

// Run executes the shared Compiler semantic suite. Each case builds through
// Harness.Factory, invokes Harness.Execute, and compares canonical record-ID
// sets rather than backend output text. Native and Expr cases run only when
// their corresponding optional constructor is present.
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

	for _, test := range semanticCases(harness) {
		t.Run(test.name, func(t *testing.T) {
			if test.requiresMissing && !harness.DistinguishesMissing {
				t.Skip("backend profile does not distinguish missing from explicit null")
			}

			builder := harness.Factory.New()
			test.build(builder, harness)
			condition, err := builder.Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if harness.InspectCondition != nil {
				if err := harness.InspectCondition(test.name, condition); err != nil {
					t.Fatalf("InspectCondition() error = %v", err)
				}
			}
			ids, err := harness.Execute(condition)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			got := canonicalIDs(ids)
			want := canonicalIDs(test.wantIDs)
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
			name:    "constant false empty any",
			wantIDs: nil,
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.AnyOf(func(*weave.Group[E]) {})
			},
		},
		{
			name:    "scalar equality",
			wantIDs: []string{"r03"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.EQ(fields.Number, int64(3))
			},
		},
		{
			name:    "scalar literal text",
			wantIDs: []string{"r03", "r04"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.Contains(fields.Text, "prefix")
			},
		},
		{
			name:    "equality",
			wantIDs: []string{"r02", "r06"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.EQ(fields.NullableNumber, int64(2))
			},
		},
		{
			name:    "inequality excludes null and missing",
			wantIDs: []string{"r01", "r05"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.NEQ(fields.NullableNumber, int64(2))
			},
		},
		{
			name:    "less than",
			wantIDs: []string{"r01"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.LT(fields.NullableNumber, int64(2))
			},
		},
		{
			name:    "less than or equal",
			wantIDs: []string{"r01", "r02", "r06"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.LTE(fields.NullableNumber, int64(2))
			},
		},
		{
			name:    "greater than",
			wantIDs: []string{"r05"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.GT(fields.NullableNumber, int64(2))
			},
		},
		{
			name:    "greater than or equal",
			wantIDs: []string{"r02", "r05", "r06"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.GTE(fields.NullableNumber, int64(2))
			},
		},
		{
			name:    "membership preserves set semantics",
			wantIDs: []string{"r02", "r05", "r06"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.In(fields.NullableNumber, []int64{2, 2, 5})
			},
		},
		{
			name:    "negative membership excludes null and missing",
			wantIDs: []string{"r01"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.NotIn(fields.NullableNumber, []int64{2, 2, 5})
			},
		},
		{
			name:    "inclusive range",
			wantIDs: []string{"r02", "r06"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.Between(fields.NullableNumber, int64(2), int64(4))
			},
		},
		{
			name:            "explicit null only",
			wantIDs:         []string{"r03"},
			requiresMissing: true,
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.IsNull(fields.NullableNumber)
			},
		},
		{
			name:    "not null means value",
			wantIDs: []string{"r01", "r02", "r05", "r06"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.NotNull(fields.NullableNumber)
			},
		},
		{
			name:    "literal contains special characters",
			wantIDs: []string{"r02"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.Contains(fields.NullableText, LiteralSpecialText)
			},
		},
		{
			name:    "literal prefix",
			wantIDs: []string{"r06"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.HasPrefix(fields.NullableText, ".*")
			},
		},
		{
			name:    "literal suffix",
			wantIDs: []string{"r02"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.HasSuffix(fields.NullableText, "\u4e16\u754c\nend")
			},
		},
		{
			name:    "root conjunction",
			wantIDs: []string{"r02", "r03", "r04"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.GTE(fields.Number, int64(2)).
					LTE(fields.Number, int64(4))
			},
		},
		{
			name:    "all of",
			wantIDs: []string{"r02", "r03", "r04"},
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
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.NoneOf(func(group *weave.Group[E]) {
					group.EQ(fields.NullableNumber, int64(2))
				})
			},
		},
		{
			name:    "not all of is match-set complement",
			wantIDs: []string{"r01", "r06"},
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				builder.NotAllOf(func(group *weave.Group[E]) {
					group.GTE(fields.Number, int64(2)).
						LTE(fields.Number, int64(5))
				})
			},
		},
		{
			name:            "nullable membership",
			wantIDs:         []string{"r02", "r03", "r06"},
			requiresMissing: true,
			build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
				two := int64(2)
				builder.In(fields.NullableNumber, []*int64{&two, nil, &two})
			},
		},
		{
			name:            "missing state",
			wantIDs:         []string{"r04"},
			requiresMissing: true,
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
