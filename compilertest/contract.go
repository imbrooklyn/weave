package compilertest

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/imbrooklyn/weave"
)

var allOperators = []weave.Operator{
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
}

var numericOperators = []weave.Operator{
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
}

var textOperators = []weave.Operator{
	weave.OperatorEQ,
	weave.OperatorNEQ,
	weave.OperatorIn,
	weave.OperatorNotIn,
	weave.OperatorIsNull,
	weave.OperatorNotNull,
	weave.OperatorContains,
	weave.OperatorHasPrefix,
	weave.OperatorHasSuffix,
}

var equalityOperators = []weave.Operator{
	weave.OperatorEQ,
	weave.OperatorNEQ,
	weave.OperatorIn,
	weave.OperatorNotIn,
	weave.OperatorIsNull,
	weave.OperatorNotNull,
}

func runCompilerContract[C, E any](t *testing.T, harness Harness[C, E]) {
	t.Helper()
	t.Run("capabilities", func(t *testing.T) {
		assertGlobalCapabilities(t, harness.Factory.Capabilities())
		assertFieldCapabilities(t, harness)
	})
	t.Run("invalid field error", func(t *testing.T) {
		assertCompileError(
			t,
			harness,
			func(builder *weave.Builder[C, E]) {
				builder.EQ("compilertest-secret-field", int64(2))
			},
			weave.ErrInvalidField,
			weave.CodeInvalidField,
			weave.OperatorEQ,
			0,
			[]string{"compilertest-secret-field"},
		)
	})
	t.Run("invalid value error", func(t *testing.T) {
		assertCompileError(
			t,
			harness,
			func(builder *weave.Builder[C, E]) {
				builder.EQ(
					harness.Fields.Number,
					"compilertest-secret-query-value",
				)
			},
			weave.ErrInvalidValue,
			weave.CodeInvalidValue,
			weave.OperatorEQ,
			0,
			[]string{"compilertest-secret-query-value"},
		)
	})
	t.Run("operator not applicable error", func(t *testing.T) {
		assertCompileError(
			t,
			harness,
			func(builder *weave.Builder[C, E]) {
				builder.Contains(
					harness.Fields.EqualityOnlyText,
					"compilertest-secret-text-value",
				)
			},
			weave.ErrOperatorNotApplicable,
			weave.CodeOperatorNotApplicable,
			weave.OperatorContains,
			0,
			[]string{"compilertest-secret-text-value"},
		)
	})
	t.Run("stable first error", func(t *testing.T) {
		assertCompileError(
			t,
			harness,
			func(builder *weave.Builder[C, E]) {
				builder.EQ(
					harness.Fields.Number,
					"compilertest-first-secret",
				).EQ(
					"compilertest-second-secret-field",
					int64(2),
				)
			},
			weave.ErrInvalidValue,
			weave.CodeInvalidValue,
			weave.OperatorEQ,
			0,
			[]string{
				"compilertest-first-secret",
				"compilertest-second-secret-field",
			},
		)
	})
	if harness.NilLikeNativeCondition != nil {
		t.Run("nil-like native condition error", func(t *testing.T) {
			assertCompileError(
				t,
				harness,
				func(builder *weave.Builder[C, E]) {
					builder.Native(harness.NilLikeNativeCondition())
				},
				weave.ErrInvalidValue,
				weave.CodeInvalidValue,
				0,
				weave.FeatureNativeCondition,
				nil,
			)
		})
	}
	if harness.NilLikeNativeExpression != nil {
		t.Run("nil-like native expression error", func(t *testing.T) {
			assertCompileError(
				t,
				harness,
				func(builder *weave.Builder[C, E]) {
					builder.Expr(harness.NilLikeNativeExpression())
				},
				weave.ErrInvalidValue,
				weave.CodeInvalidValue,
				0,
				weave.FeatureNativeExpression,
				nil,
			)
		})
	}
	t.Run("repeated compile", func(t *testing.T) {
		predicate, wantIDs := stabilityPredicate(t, harness)
		for iteration := range 32 {
			condition, err := harness.Factory.Compile(predicate)
			if err != nil {
				t.Fatalf("Compile() iteration %d error = %v", iteration, err)
			}
			assertExecutionIDs(t, harness, condition, wantIDs)
		}
	})
	t.Run("concurrent compile", func(t *testing.T) {
		predicate, wantIDs := stabilityPredicate(t, harness)
		type result struct {
			ids []string
			err error
		}
		results := make(chan result, 32)
		for range 32 {
			go func() {
				condition, err := harness.Factory.Compile(predicate)
				if err != nil {
					results <- result{err: err}
					return
				}
				ids, err := harness.Execute(condition)
				results <- result{ids: ids, err: err}
			}()
		}
		want := canonicalIDs(wantIDs)
		for iteration := range 32 {
			result := <-results
			if result.err != nil {
				t.Fatalf("concurrent result %d error = %v", iteration, result.err)
			}
			if got := canonicalIDs(result.ids); !slices.Equal(got, want) {
				t.Fatalf(
					"concurrent result %d IDs = %v, want %v",
					iteration,
					got,
					want,
				)
			}
		}
	})
}

func assertGlobalCapabilities(t *testing.T, capabilities weave.Capabilities) {
	t.Helper()
	for _, operator := range allOperators {
		if !capabilities.Operators.Has(operator) {
			t.Errorf("global capabilities do not contain %s", operator)
		}
	}
	for _, feature := range []weave.Feature{
		weave.FeatureNativeCondition,
		weave.FeatureNativeExpression,
	} {
		if !capabilities.Features.Has(feature) {
			t.Errorf("global capabilities do not contain %s", feature)
		}
	}
}

func assertFieldCapabilities[C, E any](t *testing.T, harness Harness[C, E]) {
	t.Helper()
	tests := []struct {
		name          string
		field         any
		operators     []weave.Operator
		exactStandard bool
	}{
		{name: "number", field: harness.Fields.Number, operators: numericOperators},
		{name: "text", field: harness.Fields.Text, operators: textOperators},
		{name: "nullable number", field: harness.Fields.NullableNumber, operators: numericOperators},
		{name: "nullable text", field: harness.Fields.NullableText, operators: textOperators},
		{
			name:          "equality only text",
			field:         harness.Fields.EqualityOnlyText,
			operators:     equalityOperators,
			exactStandard: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capabilities, err := harness.Resolver.CapabilitiesFor(test.field)
			if err != nil {
				t.Fatalf("CapabilitiesFor() error = %v", err)
			}
			for _, operator := range test.operators {
				if !capabilities.Operators.Has(operator) {
					t.Errorf("field capabilities do not contain %s", operator)
				}
			}
			if test.exactStandard {
				for _, operator := range allOperators {
					if got, want := capabilities.Operators.Has(operator),
						slices.Contains(test.operators, operator); got != want {
						t.Errorf(
							"field capability for %s = %v, want %v",
							operator,
							got,
							want,
						)
					}
				}
			}
		})
	}
	if _, err := harness.Resolver.CapabilitiesFor(
		"compilertest-secret-invalid-field",
	); !errors.Is(err, weave.ErrInvalidField) {
		t.Fatalf("CapabilitiesFor(invalid field) error = %v, want ErrInvalidField", err)
	}
}

func assertCompileError[C, E any](
	t *testing.T,
	harness Harness[C, E],
	build func(*weave.Builder[C, E]),
	wantSentinel error,
	wantCode weave.ErrorCode,
	wantOperator weave.Operator,
	wantFeature weave.Feature,
	secrets []string,
) {
	t.Helper()
	builder := harness.Factory.New()
	build(builder)
	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v", err)
	}
	root, ok := predicate.Root().AsGroup()
	if !ok {
		t.Fatal("Predicate root is not a group")
	}
	first, ok := root.Child(0)
	if !ok {
		t.Fatal("Predicate root has no first child")
	}

	condition, err := harness.Factory.Compile(predicate)
	if !isZero(condition) {
		t.Fatal("Compile() returned a nonzero condition on failure")
	}
	if !errors.Is(err, wantSentinel) || !errors.Is(err, weave.ErrCompile) {
		t.Fatalf("Compile() error = %v, want %v and ErrCompile", err, wantSentinel)
	}
	var detail *weave.Error
	if !errors.As(err, &detail) {
		t.Fatalf("Compile() error type = %T, want *weave.Error", err)
	}
	if detail.Code != wantCode ||
		detail.Phase != weave.PhaseValidate ||
		detail.Operator != wantOperator ||
		detail.Feature != wantFeature {
		t.Fatalf(
			"error detail = (code=%s, phase=%s, operator=%s, feature=%s), want (%s, validate, %s, %s)",
			detail.Code,
			detail.Phase,
			detail.Operator,
			detail.Feature,
			wantCode,
			wantOperator,
			wantFeature,
		)
	}
	if detail.Path.String() != first.Path().String() || detail.Origin != first.Origin() {
		t.Fatalf(
			"error location = (%s, %+v), want (%s, %+v)",
			detail.Path,
			detail.Origin,
			first.Path(),
			first.Origin(),
		)
	}
	for _, secret := range secrets {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("Compile() error disclosed %q: %q", secret, err)
		}
	}
}

func stabilityPredicate[C, E any](
	t *testing.T,
	harness Harness[C, E],
) (weave.Predicate[C, E], []string) {
	t.Helper()
	predicate, err := harness.Factory.New().
		GTE(harness.Fields.Number, int64(2)).
		AnyOf(func(group *weave.Group[E]) {
			group.Contains(harness.Fields.Text, "prefix").
				In(harness.Fields.Number, []int64{2, 6})
		}).
		Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v", err)
	}
	return predicate, []string{"r02", "r03", "r04", "r06"}
}

func assertExecutionIDs[C, E any](
	t *testing.T,
	harness Harness[C, E],
	condition C,
	wantIDs []string,
) {
	t.Helper()
	ids, err := harness.Execute(condition)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	got := canonicalIDs(ids)
	want := canonicalIDs(wantIDs)
	if !slices.Equal(got, want) {
		t.Fatalf("matching IDs = %v, want %v", got, want)
	}
}

func isZero[T any](value T) bool {
	var zero T
	return reflect.DeepEqual(value, zero)
}

func isNilLike(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
