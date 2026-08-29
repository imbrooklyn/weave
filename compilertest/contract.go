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
	capabilities := harness.Factory.Capabilities()
	t.Run("capabilities", func(t *testing.T) {
		assertHarnessCapabilities(t, harness, capabilities)
		assertFieldCapabilities(t, harness, capabilities)
	})
	t.Run("unsupported capabilities", func(t *testing.T) {
		assertUnsupportedCapabilities(t, harness, capabilities)
	})
	if capabilities.Operators.Has(weave.OperatorEQ) {
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
	}
	if operator, ok := notApplicableOperator(harness, capabilities); ok {
		t.Run("operator not applicable error", func(t *testing.T) {
			assertCompileError(
				t,
				harness,
				func(builder *weave.Builder[C, E]) {
					addTextOperator(
						builder,
						harness.Fields.EqualityOnlyText,
						operator,
					)
				},
				weave.ErrOperatorNotApplicable,
				weave.CodeOperatorNotApplicable,
				operator,
				0,
				[]string{"compilertest-secret-text-value"},
			)
		})
	}
	if capabilities.Features.Has(weave.FeatureNativeCondition) &&
		harness.NilLikeNativeCondition != nil {
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
	if capabilities.Features.Has(weave.FeatureNativeExpression) &&
		harness.NilLikeNativeExpression != nil {
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

func assertHarnessCapabilities[C, E any](
	t *testing.T,
	harness Harness[C, E],
	capabilities weave.Capabilities,
) {
	t.Helper()
	if capabilities.Features.Has(weave.FeatureNativeCondition) &&
		harness.NativeCondition == nil {
		t.Error("global capabilities contain native_condition but Harness.NativeCondition is nil")
	}
	if capabilities.Features.Has(weave.FeatureNativeExpression) &&
		harness.NativeExpression == nil {
		t.Error("global capabilities contain native_expression but Harness.NativeExpression is nil")
	}
}

func assertFieldCapabilities[C, E any](
	t *testing.T,
	harness Harness[C, E],
	global weave.Capabilities,
) {
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
				if global.Operators.Has(operator) &&
					!capabilities.Operators.Has(operator) {
					t.Errorf(
						"field capabilities do not contain globally supported %s",
						operator,
					)
				}
			}
			for index := range capabilities.Operators.Count() {
				operator, ok := capabilities.Operators.At(index)
				if !ok {
					t.Fatalf("field capability index %d is invalid", index)
				}
				if !global.Operators.Has(operator) {
					t.Errorf("field capabilities contain globally unsupported %s", operator)
				}
			}
			if test.exactStandard {
				for _, operator := range allOperators {
					if got, want := capabilities.Operators.Has(operator),
						global.Operators.Has(operator) &&
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

func assertUnsupportedCapabilities[C, E any](
	t *testing.T,
	harness Harness[C, E],
	capabilities weave.Capabilities,
) {
	t.Helper()
	for _, operator := range allOperators {
		if capabilities.Operators.Has(operator) {
			continue
		}
		operator := operator
		t.Run("operator "+operator.String(), func(t *testing.T) {
			builder := harness.Factory.New()
			addOperator(builder, harness.Fields, operator)
			assertUnsupportedError(
				t,
				harness.Factory,
				builder,
				weave.ErrUnsupportedOperator,
				weave.CodeUnsupportedOperator,
				operator,
				0,
			)
		})
	}
	for _, feature := range []weave.Feature{
		weave.FeatureNativeCondition,
		weave.FeatureNativeExpression,
	} {
		if capabilities.Features.Has(feature) {
			continue
		}
		feature := feature
		t.Run("feature "+feature.String(), func(t *testing.T) {
			builder := harness.Factory.New()
			switch feature {
			case weave.FeatureNativeCondition:
				var condition C
				if harness.NativeCondition != nil {
					condition = harness.NativeCondition([]string{"r01"})
				}
				builder.Native(condition)
			case weave.FeatureNativeExpression:
				var expression E
				if harness.NativeExpression != nil {
					expression = harness.NativeExpression([]string{"r01"})
				}
				builder.Expr(expression)
			}
			assertUnsupportedError(
				t,
				harness.Factory,
				builder,
				weave.ErrUnsupportedFeature,
				weave.CodeUnsupportedFeature,
				0,
				feature,
			)
		})
	}
}

func addOperator[C, E any](
	builder *weave.Builder[C, E],
	fields Fields,
	operator weave.Operator,
) {
	switch operator {
	case weave.OperatorEQ:
		builder.EQ(fields.Number, int64(2))
	case weave.OperatorNEQ:
		builder.NEQ(fields.Number, int64(2))
	case weave.OperatorLT:
		builder.LT(fields.Number, int64(2))
	case weave.OperatorLTE:
		builder.LTE(fields.Number, int64(2))
	case weave.OperatorGT:
		builder.GT(fields.Number, int64(2))
	case weave.OperatorGTE:
		builder.GTE(fields.Number, int64(2))
	case weave.OperatorIn:
		builder.In(fields.Number, []int64{2})
	case weave.OperatorNotIn:
		builder.NotIn(fields.Number, []int64{2})
	case weave.OperatorBetween:
		builder.Between(fields.Number, int64(1), int64(2))
	case weave.OperatorIsNull:
		builder.IsNull(fields.NullableNumber)
	case weave.OperatorNotNull:
		builder.NotNull(fields.NullableNumber)
	case weave.OperatorContains:
		builder.Contains(fields.Text, "compilertest-secret-text-value")
	case weave.OperatorHasPrefix:
		builder.HasPrefix(fields.Text, "compilertest-secret-text-value")
	case weave.OperatorHasSuffix:
		builder.HasSuffix(fields.Text, "compilertest-secret-text-value")
	}
}

func assertUnsupportedError[C, E any](
	t *testing.T,
	factory *weave.Factory[C, E],
	builder *weave.Builder[C, E],
	wantSentinel error,
	wantCode weave.ErrorCode,
	wantOperator weave.Operator,
	wantFeature weave.Feature,
) {
	t.Helper()
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

	condition, err := factory.Compile(predicate)
	if !isZero(condition) {
		t.Fatal("Compile() returned a nonzero condition for an unsupported capability")
	}
	if !errors.Is(err, wantSentinel) || !errors.Is(err, weave.ErrCompile) {
		t.Fatalf("Compile() error = %v, want %v and ErrCompile", err, wantSentinel)
	}
	var detail *weave.Error
	if !errors.As(err, &detail) {
		t.Fatalf("Compile() error type = %T, want *weave.Error", err)
	}
	if detail.Code != wantCode ||
		detail.Phase != weave.PhasePreflight ||
		detail.Operator != wantOperator ||
		detail.Feature != wantFeature {
		t.Fatalf(
			"error detail = (code=%s, phase=%s, operator=%s, feature=%s), want (%s, preflight, %s, %s)",
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
}

func notApplicableOperator[C, E any](
	harness Harness[C, E],
	global weave.Capabilities,
) (weave.Operator, bool) {
	capabilities, err := harness.Resolver.CapabilitiesFor(harness.Fields.EqualityOnlyText)
	if err != nil {
		return 0, false
	}
	for _, operator := range []weave.Operator{
		weave.OperatorContains,
		weave.OperatorHasPrefix,
		weave.OperatorHasSuffix,
		weave.OperatorLT,
		weave.OperatorLTE,
		weave.OperatorGT,
		weave.OperatorGTE,
	} {
		if global.Operators.Has(operator) && !capabilities.Operators.Has(operator) {
			return operator, true
		}
	}
	return 0, false
}

func addTextOperator[C, E any](
	builder *weave.Builder[C, E],
	field any,
	operator weave.Operator,
) {
	const value = "compilertest-secret-text-value"
	switch operator {
	case weave.OperatorContains:
		builder.Contains(field, value)
	case weave.OperatorHasPrefix:
		builder.HasPrefix(field, value)
	case weave.OperatorHasSuffix:
		builder.HasSuffix(field, value)
	case weave.OperatorLT:
		builder.LT(field, value)
	case weave.OperatorLTE:
		builder.LTE(field, value)
	case weave.OperatorGT:
		builder.GT(field, value)
	case weave.OperatorGTE:
		builder.GTE(field, value)
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
	preferred := semanticCase[C, E]{
		name:    "contract stability",
		wantIDs: []string{"r02", "r03", "r04", "r06"},
		fieldOperators: append(
			uses(harness.Fields.Number, weave.OperatorGTE, weave.OperatorIn),
			uses(harness.Fields.Text, weave.OperatorContains)...,
		),
		build: func(builder *weave.Builder[C, E], _ Harness[C, E]) {
			builder.GTE(harness.Fields.Number, int64(2)).
				AnyOf(func(group *weave.Group[E]) {
					group.Contains(harness.Fields.Text, "prefix").
						In(harness.Fields.Number, []int64{2, 6})
				})
		},
	}
	selected := preferred
	if !semanticCaseApplicable(preferred, harness) {
		selected = semanticCase[C, E]{}
		for _, candidate := range semanticCases(harness) {
			if !semanticCaseApplicable(candidate, harness) ||
				(candidate.requiresMissing &&
					!candidate.supportsMissingCollapsed &&
					!harness.DistinguishesMissing) {
				continue
			}
			if selected.build == nil || len(candidate.fieldOperators) != 0 ||
				len(candidate.features) != 0 {
				selected = candidate
			}
			if len(candidate.fieldOperators) != 0 || len(candidate.features) != 0 {
				break
			}
		}
	}
	if selected.build == nil {
		t.Fatal("compilertest: no applicable stability scenario")
	}
	builder := harness.Factory.New()
	selected.build(builder, harness)
	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v", err)
	}
	wantIDs := selected.wantIDs
	if !harness.DistinguishesMissing && selected.supportsMissingCollapsed {
		wantIDs = selected.missingCollapsedIDs
	}
	return predicate, wantIDs
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
