package weave

import "errors"

// Factory owns one Compiler and its immutable capability snapshot. Factory is
// request-stateless, and New, Capabilities, and Compile are safe for concurrent
// use when the bound Compiler follows the Compiler concurrency contract.
type Factory[C, E any] struct {
	compiler     Compiler[C, E]
	capabilities Capabilities
	domain       *predicateDomain
}

// NewFactory binds compiler and captures its capabilities exactly once. It
// panics when compiler is nil, including when its interface holds a typed nil.
func NewFactory[C, E any](compiler Compiler[C, E]) *Factory[C, E] {
	if isNilLike(compiler) {
		panic("weave: nil compiler")
	}
	return &Factory[C, E]{
		compiler:     compiler,
		capabilities: compiler.Capabilities(),
		domain:       newPredicateDomain(),
	}
}

// New returns a fresh mutable Builder bound to f. The returned Builder is not
// safe for concurrent use and must not be shared between requests.
func (f *Factory[C, E]) New() *Builder[C, E] {
	return newBuilderForFactory(f)
}

// Capabilities returns the value snapshot captured by NewFactory. It does not
// call the bound Compiler. The zero Factory returns zero Capabilities.
func (f *Factory[C, E]) Capabilities() Capabilities {
	if !validFactory(f) {
		return Capabilities{}
	}
	return f.capabilities
}

// Compile validates predicate identity and structure, performs capability
// preflight, and delegates to the bound Compiler. Every failure returns the
// zero value of C and a compile-stage Error.
func (f *Factory[C, E]) Compile(predicate Predicate[C, E]) (C, error) {
	var zero C
	if !validFactory(f) {
		return zero, newCompileError(
			CodeInvalidState,
			PhasePreflight,
			NodePath{},
			Origin{},
			0,
			0,
			nil,
		)
	}

	switch predicate.statusFor(f.domain) {
	case predicateInvalid, predicateForeign:
		return zero, newCompileError(
			CodeInvalidPredicate,
			PhasePreflight,
			NodePath{},
			Origin{},
			0,
			0,
			nil,
		)
	}

	if err := validateCompilePredicate[C, E](predicate.state); err != nil {
		return zero, err
	}
	if err := preflightCapabilities[C, E](
		predicate.state,
		f.capabilities,
	); err != nil {
		return zero, err
	}

	compiled, err := f.compiler.Compile(predicate)
	if err != nil {
		return zero, normalizeCompileError(err)
	}
	return compiled, nil
}

func validFactory[C, E any](factory *Factory[C, E]) bool {
	return factory != nil &&
		factory.domain != nil &&
		!isNilLike(factory.compiler)
}

func newCompileError(
	code ErrorCode,
	phase ErrorPhase,
	path NodePath,
	origin Origin,
	operator Operator,
	feature Feature,
	cause error,
) *Error {
	return &Error{
		Code:     code,
		Phase:    phase,
		Path:     newNodePath(path.segments...),
		Origin:   origin,
		Operator: operator,
		Feature:  feature,
		Cause:    cause,
	}
}

func normalizeCompileError(source error) *Error {
	var structured *Error
	if errors.As(source, &structured) && structured != nil {
		cloned := *structured
		cloned.Path = newNodePath(structured.Path.segments...)
		cloned.Code = normalizeCompileCode(cloned.Code, source)
		cloned.Phase = normalizeCompilePhase(cloned.Code, cloned.Phase)
		return &cloned
	}

	code := errorCodeForSentinel(source)
	phase := PhaseValidate
	if code == 0 {
		code = CodeCompileFailure
		phase = PhaseEmit
	}
	return newCompileError(
		code,
		phase,
		NodePath{},
		Origin{},
		0,
		0,
		source,
	)
}

func normalizeCompileCode(code ErrorCode, source error) ErrorCode {
	if validErrorCode(code) {
		return code
	}
	if classified := errorCodeForSentinel(source); classified != 0 {
		return classified
	}
	return CodeCompileFailure
}

func normalizeCompilePhase(code ErrorCode, phase ErrorPhase) ErrorPhase {
	if isCompilePhase(phase) {
		return phase
	}
	if code == CodeCompileFailure {
		return PhaseEmit
	}
	return PhaseValidate
}

func validErrorCode(code ErrorCode) bool {
	return code >= CodeInvalidPredicate && code <= CodeCompileFailure
}

func errorCodeForSentinel(err error) ErrorCode {
	for _, candidate := range [...]struct {
		sentinel error
		code     ErrorCode
	}{
		{ErrInvalidPredicate, CodeInvalidPredicate},
		{ErrInvalidField, CodeInvalidField},
		{ErrInvalidValue, CodeInvalidValue},
		{ErrInvalidRange, CodeInvalidRange},
		{ErrInvalidState, CodeInvalidState},
		{ErrOperatorNotApplicable, CodeOperatorNotApplicable},
		{ErrUnsupportedOperator, CodeUnsupportedOperator},
		{ErrUnsupportedFeature, CodeUnsupportedFeature},
		{ErrNonNestableNative, CodeNonNestableNative},
		{ErrDepthLimit, CodeDepthLimit},
	} {
		if errors.Is(err, candidate.sentinel) {
			return candidate.code
		}
	}
	return 0
}
