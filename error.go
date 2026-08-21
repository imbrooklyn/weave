package weave

import (
	"errors"
	"reflect"
	"strings"
)

var (
	// ErrInvalidPredicate classifies an invalid or unusable predicate.
	ErrInvalidPredicate = errors.New("weave: invalid predicate")
	// ErrInvalidField classifies an unrecognized field.
	ErrInvalidField = errors.New("weave: invalid field")
	// ErrInvalidValue classifies a value that is invalid for an operation.
	ErrInvalidValue = errors.New("weave: invalid value")
	// ErrInvalidRange classifies an invalid range.
	ErrInvalidRange = errors.New("weave: invalid range")
	// ErrInvalidState classifies an invalid object lifecycle state.
	ErrInvalidState = errors.New("weave: invalid state")
	// ErrOperatorNotApplicable classifies an operator that is not applicable to
	// an otherwise valid field.
	ErrOperatorNotApplicable = errors.New("weave: operator not applicable")
	// ErrUnsupportedOperator classifies an operator unsupported by a compiler.
	ErrUnsupportedOperator = errors.New("weave: unsupported operator")
	// ErrUnsupportedFeature classifies an optional feature unsupported by a
	// compiler.
	ErrUnsupportedFeature = errors.New("weave: unsupported feature")
	// ErrNonNestableNative classifies a native condition found below the
	// predicate root.
	ErrNonNestableNative = errors.New("weave: non-nestable native condition")
	// ErrDepthLimit classifies a predicate that exceeds the supported depth.
	ErrDepthLimit = errors.New("weave: predicate depth limit")
	// ErrCompile classifies every error from predicate preflight, validation, or
	// emission.
	ErrCompile = errors.New("weave: compile predicate")
)

// ErrorCode identifies a stable category of Weave error.
//
// The zero value is invalid. An ErrorCode's underlying integer is an
// implementation detail, not a persistence, serialization, or interchange
// protocol.
type ErrorCode uint16

const (
	// CodeInvalidPredicate identifies an invalid predicate.
	CodeInvalidPredicate ErrorCode = iota + 1
	// CodeInvalidField identifies an unrecognized field.
	CodeInvalidField
	// CodeInvalidValue identifies an invalid value.
	CodeInvalidValue
	// CodeInvalidRange identifies an invalid range.
	CodeInvalidRange
	// CodeInvalidState identifies an invalid lifecycle state.
	CodeInvalidState
	// CodeOperatorNotApplicable identifies an operator that is not applicable to
	// a valid field.
	CodeOperatorNotApplicable
	// CodeUnsupportedOperator identifies an unsupported operator.
	CodeUnsupportedOperator
	// CodeUnsupportedFeature identifies an unsupported optional feature.
	CodeUnsupportedFeature
	// CodeNonNestableNative identifies a native condition below the predicate
	// root.
	CodeNonNestableNative
	// CodeDepthLimit identifies a predicate depth limit violation.
	CodeDepthLimit
	// CodeCompileFailure identifies a compiler failure with no more specific
	// Weave category.
	CodeCompileFailure
)

// String returns the stable English diagnostic identifier for c. It returns
// error_code(n), with n in decimal, for zero or an unrecognized value. The
// result is intended for diagnostics, not serialization.
func (c ErrorCode) String() string {
	switch c {
	case CodeInvalidPredicate:
		return "invalid_predicate"
	case CodeInvalidField:
		return "invalid_field"
	case CodeInvalidValue:
		return "invalid_value"
	case CodeInvalidRange:
		return "invalid_range"
	case CodeInvalidState:
		return "invalid_state"
	case CodeOperatorNotApplicable:
		return "operator_not_applicable"
	case CodeUnsupportedOperator:
		return "unsupported_operator"
	case CodeUnsupportedFeature:
		return "unsupported_feature"
	case CodeNonNestableNative:
		return "non_nestable_native"
	case CodeDepthLimit:
		return "depth_limit"
	case CodeCompileFailure:
		return "compile_failure"
	default:
		return unknownEnumString("error_code", uint64(c))
	}
}

// ErrorPhase identifies the stage at which a Weave error was detected.
//
// The zero value is invalid. An ErrorPhase's underlying integer is an
// implementation detail, not a persistence, serialization, or interchange
// protocol.
type ErrorPhase uint8

const (
	// PhaseConstruct identifies predicate construction.
	PhaseConstruct ErrorPhase = iota + 1
	// PhaseNormalize identifies predicate normalization.
	PhaseNormalize
	// PhasePreflight identifies capability preflight.
	PhasePreflight
	// PhaseValidate identifies compiler validation.
	PhaseValidate
	// PhaseEmit identifies backend expression emission.
	PhaseEmit
)

// String returns the stable English diagnostic identifier for p. It returns
// error_phase(n), with n in decimal, for zero or an unrecognized value. The
// result is intended for diagnostics, not serialization.
func (p ErrorPhase) String() string {
	switch p {
	case PhaseConstruct:
		return "construct"
	case PhaseNormalize:
		return "normalize"
	case PhasePreflight:
		return "preflight"
	case PhaseValidate:
		return "validate"
	case PhaseEmit:
		return "emit"
	default:
		return unknownEnumString("error_phase", uint64(p))
	}
}

// Error is a structured, location-aware Weave error. Its Error method omits
// field values, query values, native payloads, expression payloads, and the
// text of Cause.
type Error struct {
	// Code identifies the error category.
	Code ErrorCode
	// Phase identifies the stage that detected the error.
	Phase ErrorPhase
	// Path locates the normalized predicate node, when one exists.
	Path NodePath
	// Origin identifies the builder call that produced the node or error.
	Origin Origin
	// Operator identifies the relevant standard operation, when one exists.
	Operator Operator
	// Feature identifies the relevant optional feature, when one exists.
	Feature Feature
	// FieldType records the field's Go type without recording its value.
	FieldType reflect.Type
	// ValueType records the query value's Go type without recording its value.
	ValueType reflect.Type
	// Cause retains a lower-level error for explicit unwrapping. Error does not
	// include Cause's text.
	Cause error
}

// Error returns a stable, English, redacted diagnostic string. A nil receiver
// returns "weave: <nil error>".
func (e *Error) Error() string {
	if e == nil {
		return "weave: <nil error>"
	}

	var builder strings.Builder
	builder.WriteString("weave: code=")
	builder.WriteString(e.Code.String())
	builder.WriteString(" phase=")
	builder.WriteString(e.Phase.String())
	if path := e.Path.String(); path != "" {
		builder.WriteString(" path=")
		builder.WriteString(path)
	}
	if e.Operator != 0 {
		builder.WriteString(" operator=")
		builder.WriteString(e.Operator.String())
	}
	if e.Feature != 0 {
		builder.WriteString(" feature=")
		builder.WriteString(e.Feature.String())
	}
	if e.FieldType != nil {
		builder.WriteString(" field_type=")
		builder.WriteString(e.FieldType.String())
	}
	if e.ValueType != nil {
		builder.WriteString(" value_type=")
		builder.WriteString(e.ValueType.String())
	}
	return builder.String()
}

// Is reports whether e belongs to the category represented by target.
// Preflight, validation, and emission errors also match ErrCompile.
func (e *Error) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	if target == ErrCompile && isCompilePhase(e.Phase) {
		return true
	}
	return target == sentinelForErrorCode(e.Code)
}

// Unwrap returns e's lower-level cause. It returns nil for a nil receiver or
// when no cause is present.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func isCompilePhase(phase ErrorPhase) bool {
	switch phase {
	case PhasePreflight, PhaseValidate, PhaseEmit:
		return true
	default:
		return false
	}
}

func sentinelForErrorCode(code ErrorCode) error {
	switch code {
	case CodeInvalidPredicate:
		return ErrInvalidPredicate
	case CodeInvalidField:
		return ErrInvalidField
	case CodeInvalidValue:
		return ErrInvalidValue
	case CodeInvalidRange:
		return ErrInvalidRange
	case CodeInvalidState:
		return ErrInvalidState
	case CodeOperatorNotApplicable:
		return ErrOperatorNotApplicable
	case CodeUnsupportedOperator:
		return ErrUnsupportedOperator
	case CodeUnsupportedFeature:
		return ErrUnsupportedFeature
	case CodeNonNestableNative:
		return ErrNonNestableNative
	case CodeDepthLimit:
		return ErrDepthLimit
	default:
		return nil
	}
}
