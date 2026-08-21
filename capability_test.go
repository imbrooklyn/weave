package weave_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/imbrooklyn/weave"
)

func TestNewOperatorSet(t *testing.T) {
	tests := []struct {
		name   string
		values []weave.Operator
		want   []weave.Operator
	}{
		{name: "empty"},
		{
			name:   "deduplicates",
			values: []weave.Operator{weave.OperatorEQ, weave.OperatorEQ, weave.OperatorIn},
			want:   []weave.Operator{weave.OperatorEQ, weave.OperatorIn},
		},
		{
			name: "uses declaration order",
			values: []weave.Operator{
				weave.OperatorHasSuffix,
				weave.OperatorBetween,
				weave.OperatorEQ,
				weave.OperatorNotIn,
				weave.OperatorContains,
			},
			want: []weave.Operator{
				weave.OperatorEQ,
				weave.OperatorNotIn,
				weave.OperatorBetween,
				weave.OperatorContains,
				weave.OperatorHasSuffix,
			},
		},
		{
			name: "covers every declared operator",
			values: []weave.Operator{
				weave.OperatorHasSuffix,
				weave.OperatorHasPrefix,
				weave.OperatorContains,
				weave.OperatorNotNull,
				weave.OperatorIsNull,
				weave.OperatorBetween,
				weave.OperatorNotIn,
				weave.OperatorIn,
				weave.OperatorGTE,
				weave.OperatorGT,
				weave.OperatorLTE,
				weave.OperatorLT,
				weave.OperatorNEQ,
				weave.OperatorEQ,
			},
			want: []weave.Operator{
				weave.OperatorEQ,
				weave.OperatorNEQ,
				weave.OperatorLT,
				weave.OperatorLTE,
				weave.OperatorGT,
				weave.OperatorGTE,
				weave.OperatorIn,
				weave.OperatorNotIn,
				weave.OperatorBetween,
				weave.OperatorIsNull,
				weave.OperatorNotNull,
				weave.OperatorContains,
				weave.OperatorHasPrefix,
				weave.OperatorHasSuffix,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set := weave.NewOperatorSet(test.values...)
			assertOperatorSet(t, set, test.want)
			for _, value := range test.want {
				if !set.Has(value) {
					t.Errorf("Has(%v) = false, want true", value)
				}
			}
		})
	}
}

func TestOperatorSetRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		value weave.Operator
	}{
		{name: "zero", value: weave.Operator(0)},
		{name: "unknown", value: weave.Operator(65535)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requirePanic(t, func() {
				weave.NewOperatorSet(test.value)
			})

			if weave.NewOperatorSet(weave.OperatorEQ).Has(test.value) {
				t.Fatalf("Has(%v) = true for invalid value", test.value)
			}
		})
	}
}

func TestOperatorSetRelations(t *testing.T) {
	tests := []struct {
		name         string
		available    weave.OperatorSet
		required     weave.OperatorSet
		wantContains bool
		wantMissing  []weave.Operator
	}{
		{
			name:         "empty requirement",
			available:    weave.NewOperatorSet(weave.OperatorEQ),
			wantContains: true,
		},
		{
			name:         "all available",
			available:    weave.NewOperatorSet(weave.OperatorEQ, weave.OperatorIn),
			required:     weave.NewOperatorSet(weave.OperatorIn),
			wantContains: true,
		},
		{
			name:        "some missing",
			available:   weave.NewOperatorSet(weave.OperatorEQ, weave.OperatorIn),
			required:    weave.NewOperatorSet(weave.OperatorHasSuffix, weave.OperatorEQ, weave.OperatorBetween),
			wantMissing: []weave.Operator{weave.OperatorBetween, weave.OperatorHasSuffix},
		},
		{
			name:        "all missing from zero",
			required:    weave.NewOperatorSet(weave.OperatorNotNull, weave.OperatorContains),
			wantMissing: []weave.Operator{weave.OperatorNotNull, weave.OperatorContains},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.available.ContainsAll(test.required); got != test.wantContains {
				t.Fatalf("ContainsAll() = %v, want %v", got, test.wantContains)
			}
			assertOperatorSet(t, test.available.Missing(test.required), test.wantMissing)
		})
	}
}

func TestOperatorSetAtBounds(t *testing.T) {
	tests := []struct {
		name  string
		set   weave.OperatorSet
		index int
	}{
		{name: "negative", set: weave.NewOperatorSet(weave.OperatorEQ), index: -1},
		{name: "empty", index: 0},
		{name: "equal to count", set: weave.NewOperatorSet(weave.OperatorEQ), index: 1},
		{name: "greater than count", set: weave.NewOperatorSet(weave.OperatorEQ), index: 10},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := test.set.At(test.index)
			if ok || got != 0 {
				t.Fatalf("At(%d) = (%v, %v), want (operator(0), false)", test.index, got, ok)
			}
		})
	}
}

func TestOperatorSetIsImmutableByValue(t *testing.T) {
	input := []weave.Operator{weave.OperatorEQ, weave.OperatorLT}
	set := weave.NewOperatorSet(input...)
	input[0] = weave.OperatorHasSuffix
	input = append(input, weave.OperatorBetween)

	assertOperatorSet(t, set, []weave.Operator{weave.OperatorEQ, weave.OperatorLT})

	required := weave.NewOperatorSet(weave.OperatorLT, weave.OperatorIn)
	setBefore := operatorSetValues(t, set)
	requiredBefore := operatorSetValues(t, required)
	_ = set.Missing(required)
	assertOperatorSet(t, set, setBefore)
	assertOperatorSet(t, required, requiredBefore)

	copyOfSet := set
	copyOfSet = weave.NewOperatorSet(weave.OperatorHasSuffix)
	assertOperatorSet(t, set, []weave.Operator{weave.OperatorEQ, weave.OperatorLT})
	assertOperatorSet(t, copyOfSet, []weave.Operator{weave.OperatorHasSuffix})
}

func TestNewFeatureSet(t *testing.T) {
	tests := []struct {
		name   string
		values []weave.Feature
		want   []weave.Feature
	}{
		{name: "empty"},
		{
			name: "deduplicates and uses declaration order",
			values: []weave.Feature{
				weave.FeatureNativeExpression,
				weave.FeatureNativeCondition,
				weave.FeatureNativeExpression,
			},
			want: []weave.Feature{
				weave.FeatureNativeCondition,
				weave.FeatureNativeExpression,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set := weave.NewFeatureSet(test.values...)
			assertFeatureSet(t, set, test.want)
			for _, value := range test.want {
				if !set.Has(value) {
					t.Errorf("Has(%v) = false, want true", value)
				}
			}
		})
	}
}

func TestFeatureSetRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		value weave.Feature
	}{
		{name: "zero", value: weave.Feature(0)},
		{name: "unknown", value: weave.Feature(65535)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requirePanic(t, func() {
				weave.NewFeatureSet(test.value)
			})

			if weave.NewFeatureSet(weave.FeatureNativeCondition).Has(test.value) {
				t.Fatalf("Has(%v) = true for invalid value", test.value)
			}
		})
	}
}

func TestFeatureSetRelations(t *testing.T) {
	tests := []struct {
		name         string
		available    weave.FeatureSet
		required     weave.FeatureSet
		wantContains bool
		wantMissing  []weave.Feature
	}{
		{
			name:         "empty requirement",
			available:    weave.NewFeatureSet(weave.FeatureNativeCondition),
			wantContains: true,
		},
		{
			name:         "all available",
			available:    weave.NewFeatureSet(weave.FeatureNativeCondition, weave.FeatureNativeExpression),
			required:     weave.NewFeatureSet(weave.FeatureNativeExpression),
			wantContains: true,
		},
		{
			name:        "missing feature",
			available:   weave.NewFeatureSet(weave.FeatureNativeCondition),
			required:    weave.NewFeatureSet(weave.FeatureNativeExpression, weave.FeatureNativeCondition),
			wantMissing: []weave.Feature{weave.FeatureNativeExpression},
		},
		{
			name:        "all missing from zero",
			required:    weave.NewFeatureSet(weave.FeatureNativeCondition, weave.FeatureNativeExpression),
			wantMissing: []weave.Feature{weave.FeatureNativeCondition, weave.FeatureNativeExpression},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.available.ContainsAll(test.required); got != test.wantContains {
				t.Fatalf("ContainsAll() = %v, want %v", got, test.wantContains)
			}
			assertFeatureSet(t, test.available.Missing(test.required), test.wantMissing)
		})
	}
}

func TestFeatureSetAtBounds(t *testing.T) {
	tests := []struct {
		name  string
		set   weave.FeatureSet
		index int
	}{
		{name: "negative", set: weave.NewFeatureSet(weave.FeatureNativeCondition), index: -1},
		{name: "empty", index: 0},
		{name: "equal to count", set: weave.NewFeatureSet(weave.FeatureNativeCondition), index: 1},
		{name: "greater than count", set: weave.NewFeatureSet(weave.FeatureNativeCondition), index: 10},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := test.set.At(test.index)
			if ok || got != 0 {
				t.Fatalf("At(%d) = (%v, %v), want (feature(0), false)", test.index, got, ok)
			}
		})
	}
}

func TestFeatureSetIsImmutableByValue(t *testing.T) {
	input := []weave.Feature{weave.FeatureNativeCondition}
	set := weave.NewFeatureSet(input...)
	input[0] = weave.FeatureNativeExpression

	assertFeatureSet(t, set, []weave.Feature{weave.FeatureNativeCondition})

	required := weave.NewFeatureSet(weave.FeatureNativeCondition, weave.FeatureNativeExpression)
	setBefore := featureSetValues(t, set)
	requiredBefore := featureSetValues(t, required)
	_ = set.Missing(required)
	assertFeatureSet(t, set, setBefore)
	assertFeatureSet(t, required, requiredBefore)

	copyOfSet := set
	copyOfSet = weave.NewFeatureSet(weave.FeatureNativeExpression)
	assertFeatureSet(t, set, []weave.Feature{weave.FeatureNativeCondition})
	assertFeatureSet(t, copyOfSet, []weave.Feature{weave.FeatureNativeExpression})
}

func TestCapabilities(t *testing.T) {
	available := weave.Capabilities{
		Operators: weave.NewOperatorSet(weave.OperatorEQ, weave.OperatorIn),
		Features:  weave.NewFeatureSet(weave.FeatureNativeCondition),
	}

	tests := []struct {
		name          string
		required      weave.Requirements
		wantSupported bool
		wantOperators []weave.Operator
		wantFeatures  []weave.Feature
	}{
		{name: "zero requirements", wantSupported: true},
		{
			name: "supported",
			required: weave.Requirements{
				Operators: weave.NewOperatorSet(weave.OperatorIn),
				Features:  weave.NewFeatureSet(weave.FeatureNativeCondition),
			},
			wantSupported: true,
		},
		{
			name: "missing operator",
			required: weave.Requirements{
				Operators: weave.NewOperatorSet(weave.OperatorEQ, weave.OperatorBetween),
			},
			wantOperators: []weave.Operator{weave.OperatorBetween},
		},
		{
			name: "missing feature",
			required: weave.Requirements{
				Features: weave.NewFeatureSet(weave.FeatureNativeExpression),
			},
			wantFeatures: []weave.Feature{weave.FeatureNativeExpression},
		},
		{
			name: "missing both",
			required: weave.Requirements{
				Operators: weave.NewOperatorSet(weave.OperatorHasSuffix, weave.OperatorIn),
				Features:  weave.NewFeatureSet(weave.FeatureNativeExpression),
			},
			wantOperators: []weave.Operator{weave.OperatorHasSuffix},
			wantFeatures:  []weave.Feature{weave.FeatureNativeExpression},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := available.Supports(test.required); got != test.wantSupported {
				t.Fatalf("Supports() = %v, want %v", got, test.wantSupported)
			}

			missing := available.Missing(test.required)
			assertOperatorSet(t, missing.Operators, test.wantOperators)
			assertFeatureSet(t, missing.Features, test.wantFeatures)
		})
	}
}

func TestZeroCapabilitiesAndRequirements(t *testing.T) {
	var capabilities weave.Capabilities
	var requirements weave.Requirements

	if !capabilities.Supports(requirements) {
		t.Fatal("zero Capabilities should support zero Requirements")
	}
	missing := capabilities.Missing(requirements)
	assertOperatorSet(t, missing.Operators, nil)
	assertFeatureSet(t, missing.Features, nil)
}

func TestCapabilitiesAreImmutableByValue(t *testing.T) {
	original := weave.Capabilities{
		Operators: weave.NewOperatorSet(weave.OperatorEQ),
		Features:  weave.NewFeatureSet(weave.FeatureNativeCondition),
	}
	copyOfOriginal := original
	copyOfOriginal.Operators = weave.NewOperatorSet(weave.OperatorHasSuffix)
	copyOfOriginal.Features = weave.NewFeatureSet(weave.FeatureNativeExpression)

	assertOperatorSet(t, original.Operators, []weave.Operator{weave.OperatorEQ})
	assertFeatureSet(t, original.Features, []weave.Feature{weave.FeatureNativeCondition})
	assertOperatorSet(t, copyOfOriginal.Operators, []weave.Operator{weave.OperatorHasSuffix})
	assertFeatureSet(t, copyOfOriginal.Features, []weave.Feature{weave.FeatureNativeExpression})
}

var errUnknownField = errors.New("unknown field")

type staticFieldCapabilityResolver struct{}

func (staticFieldCapabilityResolver) CapabilitiesFor(field any) (weave.FieldCapabilities, error) {
	if field != "name" {
		return weave.FieldCapabilities{}, errUnknownField
	}
	return weave.FieldCapabilities{
		Operators: weave.NewOperatorSet(weave.OperatorEQ, weave.OperatorContains),
	}, nil
}

var _ weave.FieldCapabilityResolver = staticFieldCapabilityResolver{}

func TestFieldCapabilityResolver(t *testing.T) {
	resolver := staticFieldCapabilityResolver{}
	tests := []struct {
		name          string
		field         any
		wantOperators []weave.Operator
		wantErr       error
	}{
		{
			name:          "known field",
			field:         "name",
			wantOperators: []weave.Operator{weave.OperatorEQ, weave.OperatorContains},
		},
		{name: "unknown field", field: "other", wantErr: errUnknownField},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capabilities, err := resolver.CapabilitiesFor(test.field)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("CapabilitiesFor() error = %v, want %v", err, test.wantErr)
			}
			assertOperatorSet(t, capabilities.Operators, test.wantOperators)
		})
	}
}

func assertOperatorSet(t *testing.T, set weave.OperatorSet, want []weave.Operator) {
	t.Helper()
	if got := set.Count(); got != len(want) {
		t.Fatalf("Count() = %d, want %d", got, len(want))
	}
	if got := operatorSetValues(t, set); !slices.Equal(got, want) {
		t.Fatalf("operator iteration = %v, want %v", got, want)
	}
}

func operatorSetValues(t *testing.T, set weave.OperatorSet) []weave.Operator {
	t.Helper()
	values := make([]weave.Operator, 0, set.Count())
	for index := 0; index < set.Count(); index++ {
		value, ok := set.At(index)
		if !ok {
			t.Fatalf("At(%d) returned false below Count()", index)
		}
		values = append(values, value)
	}
	return values
}

func assertFeatureSet(t *testing.T, set weave.FeatureSet, want []weave.Feature) {
	t.Helper()
	if got := set.Count(); got != len(want) {
		t.Fatalf("Count() = %d, want %d", got, len(want))
	}
	if got := featureSetValues(t, set); !slices.Equal(got, want) {
		t.Fatalf("feature iteration = %v, want %v", got, want)
	}
}

func featureSetValues(t *testing.T, set weave.FeatureSet) []weave.Feature {
	t.Helper()
	values := make([]weave.Feature, 0, set.Count())
	for index := 0; index < set.Count(); index++ {
		value, ok := set.At(index)
		if !ok {
			t.Fatalf("At(%d) returned false below Count()", index)
		}
		values = append(values, value)
	}
	return values
}

func requirePanic(t *testing.T, function func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("function did not panic")
		}
	}()
	function()
}
