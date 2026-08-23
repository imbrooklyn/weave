package weave

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFactoryDiscardsCompilerResultAndNormalizesStructuredError(t *testing.T) {
	cause := errors.New("secret query payload")
	compiler := &factoryTestCompiler{
		capabilities: Capabilities{Operators: NewOperatorSet(OperatorEQ)},
		result:       "must be discarded",
	}
	factory := NewFactory[string, string](compiler)
	predicate, err := factory.New().EQ("field", 1).Predicate()
	if err != nil {
		t.Fatalf("Predicate failed: %v", err)
	}
	nodePath := predicate.state.root.children[0].nodePath()
	wantPath := nodePath.String()
	source := &Error{
		Code:      CodeInvalidField,
		Phase:     PhaseConstruct,
		Path:      nodePath,
		Origin:    Origin{Sequence: 1, Operator: OperatorEQ},
		Operator:  OperatorEQ,
		FieldType: reflect.TypeFor[string](),
		ValueType: reflect.TypeFor[int](),
		Cause:     cause,
	}
	compiler.err = source

	compiled, err := factory.Compile(predicate)
	if compiled != "" {
		t.Fatalf("Compile result = %q, want zero", compiled)
	}
	compileError := requireFactoryCompileError(
		t,
		err,
		CodeInvalidField,
		PhaseValidate,
		ErrInvalidField,
	)
	if compileError == source {
		t.Fatal("Factory returned the Compiler-owned Error pointer")
	}
	if compileError.Path.String() != wantPath ||
		compileError.Origin != source.Origin ||
		compileError.Operator != source.Operator ||
		compileError.Feature != source.Feature ||
		compileError.FieldType != source.FieldType ||
		compileError.ValueType != source.ValueType ||
		compileError.Cause != cause {
		t.Fatal("Factory did not preserve structured error metadata")
	}

	source.Path.segments[0] = PathSegment{}
	if compileError.Path.String() != wantPath {
		t.Fatal("normalized error shares mutable path storage with Compiler error")
	}
	diagnostic := compileError.Error()
	if strings.Contains(diagnostic, "secret") || strings.Contains(diagnostic, "payload") {
		t.Fatalf("diagnostic exposed Cause text: %q", diagnostic)
	}
	if errors.Unwrap(compileError) != cause {
		t.Fatal("normalized error did not retain the redacted Cause")
	}
}

func TestFactoryNormalizesCompilerErrorKindsAndPhases(t *testing.T) {
	tests := []struct {
		name           string
		source         error
		code           ErrorCode
		phase          ErrorPhase
		classification error
	}{
		{
			name: "plain sentinel",
			source: fmt.Errorf(
				"adapter detail with private operand: %w",
				ErrInvalidValue,
			),
			code:           CodeInvalidValue,
			phase:          PhaseValidate,
			classification: ErrInvalidValue,
		},
		{
			name:           "plain compile sentinel",
			source:         ErrCompile,
			code:           CodeCompileFailure,
			phase:          PhaseEmit,
			classification: nil,
		},
		{
			name:           "unknown error",
			source:         errors.New("password equals private-value"),
			code:           CodeCompileFailure,
			phase:          PhaseEmit,
			classification: nil,
		},
		{
			name: "structured validation category with construction phase",
			source: &Error{
				Code:  CodeOperatorNotApplicable,
				Phase: PhaseConstruct,
			},
			code:           CodeOperatorNotApplicable,
			phase:          PhaseValidate,
			classification: ErrOperatorNotApplicable,
		},
		{
			name: "structured compile failure with normalization phase",
			source: &Error{
				Code:  CodeCompileFailure,
				Phase: PhaseNormalize,
			},
			code:           CodeCompileFailure,
			phase:          PhaseEmit,
			classification: nil,
		},
		{
			name: "structured compile phase is preserved",
			source: &Error{
				Code:  CodeInvalidRange,
				Phase: PhasePreflight,
			},
			code:           CodeInvalidRange,
			phase:          PhasePreflight,
			classification: ErrInvalidRange,
		},
		{
			name: "invalid code derives category from cause",
			source: &Error{
				Code:  ErrorCode(65535),
				Phase: ErrorPhase(255),
				Cause: ErrDepthLimit,
			},
			code:           CodeDepthLimit,
			phase:          PhaseValidate,
			classification: ErrDepthLimit,
		},
		{
			name: "invalid code without category becomes emit failure",
			source: &Error{
				Code:  ErrorCode(65535),
				Phase: ErrorPhase(255),
			},
			code:           CodeCompileFailure,
			phase:          PhaseEmit,
			classification: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			compiler := &factoryTestCompiler{
				capabilities: Capabilities{Operators: NewOperatorSet(OperatorEQ)},
				result:       "must be discarded",
				err:          test.source,
			}
			factory := NewFactory[string, string](compiler)
			predicate, err := factory.New().EQ("field", 1).Predicate()
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
				test.code,
				test.phase,
				test.classification,
			)
			diagnostic := compileError.Error()
			for _, secret := range []string{"private operand", "password", "private-value"} {
				if strings.Contains(diagnostic, secret) {
					t.Fatalf("diagnostic exposed Compiler error text: %q", diagnostic)
				}
			}
			if calls := compiler.compileCalls.Load(); calls != 1 {
				t.Fatalf("Compile calls = %d, want 1", calls)
			}
		})
	}
}

type panicTextError struct{}

func (panicTextError) Error() string {
	panic("Cause.Error must not be called")
}

func TestFactoryNeverFormatsCompilerCause(t *testing.T) {
	compiler := &factoryTestCompiler{
		capabilities: Capabilities{Operators: NewOperatorSet(OperatorEQ)},
		result:       "must be discarded",
		err:          panicTextError{},
	}
	factory := NewFactory[string, string](compiler)
	predicate, err := factory.New().EQ("field", 1).Predicate()
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
		CodeCompileFailure,
		PhaseEmit,
		nil,
	)
	if compileError.Cause == nil {
		t.Fatal("Compiler cause was not retained")
	}
	if diagnostic := compileError.Error(); strings.Contains(diagnostic, "Cause") {
		t.Fatalf("diagnostic unexpectedly contains Cause text: %q", diagnostic)
	}
}

type boundaryCompiler struct {
	compileCalls atomic.Int64
}

func (c *boundaryCompiler) Compile(Predicate[string, string]) (string, error) {
	c.compileCalls.Add(1)
	return "unchecked", nil
}

func (*boundaryCompiler) Capabilities() Capabilities {
	return Capabilities{}
}

func TestDirectCompilerCallBypassesFactorySafeguards(t *testing.T) {
	compiler := &boundaryCompiler{}
	factory := NewFactory[string, string](compiler)
	predicate, err := factory.New().EQ("field", 1).Predicate()
	if err != nil {
		t.Fatalf("Predicate failed: %v", err)
	}

	direct, err := compiler.Compile(predicate)
	if err != nil || direct != "unchecked" {
		t.Fatalf("direct Compile = (%q, %v), want unchecked success", direct, err)
	}
	compiled, err := factory.Compile(predicate)
	if compiled != "" {
		t.Fatalf("Factory Compile result = %q, want zero", compiled)
	}
	requireFactoryCompileError(
		t,
		err,
		CodeUnsupportedOperator,
		PhasePreflight,
		ErrUnsupportedOperator,
	)
	if calls := compiler.compileCalls.Load(); calls != 1 {
		t.Fatalf("Compiler calls = %d, want only the direct call", calls)
	}
}

type resolverValidationCompiler struct {
	compileCalls  atomic.Int64
	resolverCalls atomic.Int64
}

func (c *resolverValidationCompiler) Compile(Predicate[string, string]) (string, error) {
	c.compileCalls.Add(1)
	return "must be discarded", ErrOperatorNotApplicable
}

func (*resolverValidationCompiler) Capabilities() Capabilities {
	return Capabilities{Operators: NewOperatorSet(OperatorEQ)}
}

func (c *resolverValidationCompiler) CapabilitiesFor(any) (FieldCapabilities, error) {
	c.resolverCalls.Add(1)
	return FieldCapabilities{Operators: NewOperatorSet(OperatorEQ)}, nil
}

var _ FieldCapabilityResolver = (*resolverValidationCompiler)(nil)

func TestFieldCapabilityResolverDoesNotReplaceCompileValidation(t *testing.T) {
	compiler := &resolverValidationCompiler{}
	factory := NewFactory[string, string](compiler)
	predicate, err := factory.New().EQ("field", 1).Predicate()
	if err != nil {
		t.Fatalf("Predicate failed: %v", err)
	}

	compiled, err := factory.Compile(predicate)
	if compiled != "" {
		t.Fatalf("Compile result = %q, want zero", compiled)
	}
	requireFactoryCompileError(
		t,
		err,
		CodeOperatorNotApplicable,
		PhaseValidate,
		ErrOperatorNotApplicable,
	)
	if calls := compiler.compileCalls.Load(); calls != 1 {
		t.Fatalf("Compile calls = %d, want 1", calls)
	}
	if calls := compiler.resolverCalls.Load(); calls != 0 {
		t.Fatalf("resolver calls = %d, want Factory not to call resolver", calls)
	}
}
