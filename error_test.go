package weave

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestSentinelErrorText(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "invalid predicate", err: ErrInvalidPredicate, want: "weave: invalid predicate"},
		{name: "invalid field", err: ErrInvalidField, want: "weave: invalid field"},
		{name: "invalid value", err: ErrInvalidValue, want: "weave: invalid value"},
		{name: "invalid range", err: ErrInvalidRange, want: "weave: invalid range"},
		{name: "invalid state", err: ErrInvalidState, want: "weave: invalid state"},
		{name: "operator not applicable", err: ErrOperatorNotApplicable, want: "weave: operator not applicable"},
		{name: "unsupported operator", err: ErrUnsupportedOperator, want: "weave: unsupported operator"},
		{name: "unsupported feature", err: ErrUnsupportedFeature, want: "weave: unsupported feature"},
		{name: "non-nestable native", err: ErrNonNestableNative, want: "weave: non-nestable native condition"},
		{name: "depth limit", err: ErrDepthLimit, want: "weave: predicate depth limit"},
		{name: "compile", err: ErrCompile, want: "weave: compile predicate"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.err.Error(); got != test.want {
				t.Fatalf("Error() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestErrorCodeString(t *testing.T) {
	tests := []struct {
		name  string
		value ErrorCode
		want  string
	}{
		{name: "invalid predicate", value: CodeInvalidPredicate, want: "invalid_predicate"},
		{name: "invalid field", value: CodeInvalidField, want: "invalid_field"},
		{name: "invalid value", value: CodeInvalidValue, want: "invalid_value"},
		{name: "invalid range", value: CodeInvalidRange, want: "invalid_range"},
		{name: "invalid state", value: CodeInvalidState, want: "invalid_state"},
		{name: "operator not applicable", value: CodeOperatorNotApplicable, want: "operator_not_applicable"},
		{name: "unsupported operator", value: CodeUnsupportedOperator, want: "unsupported_operator"},
		{name: "unsupported feature", value: CodeUnsupportedFeature, want: "unsupported_feature"},
		{name: "non-nestable native", value: CodeNonNestableNative, want: "non_nestable_native"},
		{name: "depth limit", value: CodeDepthLimit, want: "depth_limit"},
		{name: "compile failure", value: CodeCompileFailure, want: "compile_failure"},
		{name: "zero", value: ErrorCode(0), want: "error_code(0)"},
		{name: "unknown", value: ErrorCode(65535), want: "error_code(65535)"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.value.String(); got != test.want {
				t.Fatalf("String() = %q, want %q", got, test.want)
			}
			if got := test.value.String(); got != test.want {
				t.Fatalf("second String() = %q, want deterministic result %q", got, test.want)
			}
		})
	}
}

func TestErrorPhaseString(t *testing.T) {
	tests := []struct {
		name  string
		value ErrorPhase
		want  string
	}{
		{name: "construct", value: PhaseConstruct, want: "construct"},
		{name: "normalize", value: PhaseNormalize, want: "normalize"},
		{name: "preflight", value: PhasePreflight, want: "preflight"},
		{name: "validate", value: PhaseValidate, want: "validate"},
		{name: "emit", value: PhaseEmit, want: "emit"},
		{name: "zero", value: ErrorPhase(0), want: "error_phase(0)"},
		{name: "unknown", value: ErrorPhase(255), want: "error_phase(255)"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.value.String(); got != test.want {
				t.Fatalf("String() = %q, want %q", got, test.want)
			}
			if got := test.value.String(); got != test.want {
				t.Fatalf("second String() = %q, want deterministic result %q", got, test.want)
			}
		})
	}
}

func TestErrorMatchesCodeSentinel(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		sentinel error
	}{
		{name: "invalid predicate", code: CodeInvalidPredicate, sentinel: ErrInvalidPredicate},
		{name: "invalid field", code: CodeInvalidField, sentinel: ErrInvalidField},
		{name: "invalid value", code: CodeInvalidValue, sentinel: ErrInvalidValue},
		{name: "invalid range", code: CodeInvalidRange, sentinel: ErrInvalidRange},
		{name: "invalid state", code: CodeInvalidState, sentinel: ErrInvalidState},
		{name: "operator not applicable", code: CodeOperatorNotApplicable, sentinel: ErrOperatorNotApplicable},
		{name: "unsupported operator", code: CodeUnsupportedOperator, sentinel: ErrUnsupportedOperator},
		{name: "unsupported feature", code: CodeUnsupportedFeature, sentinel: ErrUnsupportedFeature},
		{name: "non-nestable native", code: CodeNonNestableNative, sentinel: ErrNonNestableNative},
		{name: "depth limit", code: CodeDepthLimit, sentinel: ErrDepthLimit},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := &Error{Code: test.code, Phase: PhaseConstruct}
			if !errors.Is(err, test.sentinel) {
				t.Fatalf("errors.Is(error, %v) = false, want true", test.sentinel)
			}
			if errors.Is(err, ErrCompile) {
				t.Fatal("construct error unexpectedly matched ErrCompile")
			}
		})
	}
}

func TestErrorCompileUmbrella(t *testing.T) {
	tests := []struct {
		name      string
		phase     ErrorPhase
		wantMatch bool
	}{
		{name: "zero"},
		{name: "construct", phase: PhaseConstruct},
		{name: "normalize", phase: PhaseNormalize},
		{name: "preflight", phase: PhasePreflight, wantMatch: true},
		{name: "validate", phase: PhaseValidate, wantMatch: true},
		{name: "emit", phase: PhaseEmit, wantMatch: true},
		{name: "unknown", phase: ErrorPhase(255)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := &Error{Code: CodeCompileFailure, Phase: test.phase}
			if got := errors.Is(err, ErrCompile); got != test.wantMatch {
				t.Fatalf("errors.Is(error, ErrCompile) = %v, want %v", got, test.wantMatch)
			}
		})
	}
}

func TestErrorCanMatchSpecificAndCompileSentinels(t *testing.T) {
	err := &Error{Code: CodeUnsupportedOperator, Phase: PhasePreflight}

	if !errors.Is(err, ErrUnsupportedOperator) {
		t.Fatal("errors.Is(error, ErrUnsupportedOperator) = false, want true")
	}
	if !errors.Is(err, ErrCompile) {
		t.Fatal("errors.Is(error, ErrCompile) = false, want true")
	}
}

func TestErrorStringIsStableAndRedacted(t *testing.T) {
	path := newNodePath(
		newRootPathSegment(LogicAllOf),
		newChildPathSegment(2),
		newNodePathSegment(KindText, 0, OperatorContains),
	)
	err := &Error{
		Code:      CodeUnsupportedOperator,
		Phase:     PhaseValidate,
		Path:      path,
		Origin:    Origin{Sequence: 987654321, Operator: OperatorContains},
		Operator:  OperatorContains,
		Feature:   FeatureNativeExpression,
		FieldType: reflect.TypeOf(""),
		ValueType: reflect.TypeOf(int64(0)),
		Cause:     errors.New("secret-query-value"),
	}
	want := "weave: code=unsupported_operator phase=validate path=root.allOf[2].contains operator=contains feature=native_expression field_type=string value_type=int64"

	if got := err.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if got := err.Error(); got != want {
		t.Fatalf("second Error() = %q, want deterministic result %q", got, want)
	}
	for _, forbidden := range []string{"secret-query-value", "987654321"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("Error() exposed redacted content %q", forbidden)
		}
	}
}

func TestErrorStringDoesNotInspectCause(t *testing.T) {
	err := &Error{
		Code:  CodeCompileFailure,
		Phase: PhaseEmit,
		Cause: causeThatMustNotBeFormatted{},
	}

	if got, want := err.Error(), "weave: code=compile_failure phase=emit"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestErrorStringHandlesZeroUnknownAndNil(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{name: "nil receiver", want: "weave: <nil error>"},
		{name: "zero", err: &Error{}, want: "weave: code=error_code(0) phase=error_phase(0)"},
		{
			name: "unknown enums",
			err: &Error{
				Code:     ErrorCode(65535),
				Phase:    ErrorPhase(255),
				Operator: Operator(65535),
				Feature:  Feature(65535),
			},
			want: "weave: code=error_code(65535) phase=error_phase(255) operator=operator(65535) feature=feature(65535)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.err.Error(); got != test.want {
				t.Fatalf("Error() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestErrorUnwrapAndAs(t *testing.T) {
	cause := &classifiedCause{}
	structured := &Error{
		Code:  CodeInvalidValue,
		Phase: PhaseValidate,
		Cause: cause,
	}
	wrapped := fmt.Errorf("outer classification: %w", structured)

	if got := structured.Unwrap(); got != cause {
		t.Fatalf("Unwrap() = %v, want cause", got)
	}
	if !errors.Is(structured, cause) {
		t.Fatal("errors.Is(error, cause) = false, want true")
	}
	var got *Error
	if !errors.As(wrapped, &got) {
		t.Fatal("errors.As() = false, want true")
	}
	if got != structured {
		t.Fatalf("errors.As() returned %p, want %p", got, structured)
	}
	var gotCause *classifiedCause
	if !errors.As(structured, &gotCause) {
		t.Fatal("errors.As(error, cause) = false, want true")
	}
	if gotCause != cause {
		t.Fatalf("errors.As(error, cause) returned %p, want %p", gotCause, cause)
	}

	var nilError *Error
	if nilError.Unwrap() != nil {
		t.Fatal("nil Error.Unwrap() returned a non-nil cause")
	}
	if nilError.Is(ErrInvalidValue) {
		t.Fatal("nil Error.Is() returned true")
	}
}

func TestErrorIsHandlesNonComparableTarget(t *testing.T) {
	err := &Error{Code: CodeInvalidValue, Phase: PhaseConstruct}
	target := nonComparableTarget{values: []string{"value"}}

	if errors.Is(err, target) {
		t.Fatal("errors.Is(error, non-comparable target) = true, want false")
	}
}

func TestUnknownAndCompileFailureCodesHaveNoSpecificSentinel(t *testing.T) {
	tests := []struct {
		name string
		code ErrorCode
	}{
		{name: "zero"},
		{name: "compile failure", code: CodeCompileFailure},
		{name: "unknown", code: ErrorCode(65535)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := &Error{Code: test.code, Phase: PhaseConstruct}
			for _, sentinel := range []error{
				ErrInvalidPredicate,
				ErrInvalidField,
				ErrInvalidValue,
				ErrInvalidRange,
				ErrInvalidState,
				ErrOperatorNotApplicable,
				ErrUnsupportedOperator,
				ErrUnsupportedFeature,
				ErrNonNestableNative,
				ErrDepthLimit,
				ErrCompile,
			} {
				if errors.Is(err, sentinel) {
					t.Fatalf("errors.Is(error, %v) = true, want false", sentinel)
				}
			}
		})
	}
}

type causeThatMustNotBeFormatted struct{}

func (causeThatMustNotBeFormatted) Error() string {
	panic("structured error formatted its cause")
}

type classifiedCause struct{}

func (*classifiedCause) Error() string {
	return "classified cause"
}

type nonComparableTarget struct {
	values []string
}

func (nonComparableTarget) Error() string {
	return "non-comparable target"
}
