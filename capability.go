package weave

import "math/bits"

// OperatorSet is an immutable set of recognized Operator values. Its zero
// value is an empty set. Copies do not share mutable backing storage.
type OperatorSet struct {
	bits uint64
}

// NewOperatorSet returns a set containing values with duplicates removed. It
// panics if a value is zero or is not a recognized Operator.
func NewOperatorSet(values ...Operator) OperatorSet {
	var set OperatorSet
	for _, value := range values {
		bit, ok := operatorBit(value)
		if !ok {
			panic("weave: invalid operator " + value.String())
		}
		set.bits |= bit
	}
	return set
}

// Has reports whether value belongs to s. It reports false for zero and
// unrecognized values.
func (s OperatorSet) Has(value Operator) bool {
	bit, ok := operatorBit(value)
	return ok && s.bits&bit != 0
}

// ContainsAll reports whether s contains every operator in required. An empty
// required set is contained in every set.
func (s OperatorSet) ContainsAll(required OperatorSet) bool {
	return s.bits&required.bits == required.bits
}

// Missing returns the operators that belong to required but not to s. It does
// not modify either operand.
func (s OperatorSet) Missing(required OperatorSet) OperatorSet {
	return OperatorSet{bits: required.bits &^ s.bits}
}

// Count returns the number of operators in s.
func (s OperatorSet) Count() int {
	return bits.OnesCount64(s.bits)
}

// At returns the operator at index in stable declaration order. It returns the
// zero Operator and false when index is outside [0, Count()). Declaration order
// is a diagnostic iteration order, not an integer-value protocol.
func (s OperatorSet) At(index int) (Operator, bool) {
	if index < 0 {
		return 0, false
	}

	for _, value := range [...]Operator{
		OperatorEQ,
		OperatorNEQ,
		OperatorLT,
		OperatorLTE,
		OperatorGT,
		OperatorGTE,
		OperatorIn,
		OperatorNotIn,
		OperatorBetween,
		OperatorIsNull,
		OperatorNotNull,
		OperatorContains,
		OperatorHasPrefix,
		OperatorHasSuffix,
	} {
		if !s.Has(value) {
			continue
		}
		if index == 0 {
			return value, true
		}
		index--
	}

	return 0, false
}

func operatorBit(value Operator) (uint64, bool) {
	switch value {
	case OperatorEQ:
		return 1 << 0, true
	case OperatorNEQ:
		return 1 << 1, true
	case OperatorLT:
		return 1 << 2, true
	case OperatorLTE:
		return 1 << 3, true
	case OperatorGT:
		return 1 << 4, true
	case OperatorGTE:
		return 1 << 5, true
	case OperatorIn:
		return 1 << 6, true
	case OperatorNotIn:
		return 1 << 7, true
	case OperatorBetween:
		return 1 << 8, true
	case OperatorIsNull:
		return 1 << 9, true
	case OperatorNotNull:
		return 1 << 10, true
	case OperatorContains:
		return 1 << 11, true
	case OperatorHasPrefix:
		return 1 << 12, true
	case OperatorHasSuffix:
		return 1 << 13, true
	default:
		return 0, false
	}
}

// FeatureSet is an immutable set of recognized Feature values. Its zero value
// is an empty set. Copies do not share mutable backing storage.
type FeatureSet struct {
	bits uint64
}

// NewFeatureSet returns a set containing values with duplicates removed. It
// panics if a value is zero or is not a recognized Feature.
func NewFeatureSet(values ...Feature) FeatureSet {
	var set FeatureSet
	for _, value := range values {
		bit, ok := featureBit(value)
		if !ok {
			panic("weave: invalid feature " + value.String())
		}
		set.bits |= bit
	}
	return set
}

// Has reports whether value belongs to s. It reports false for zero and
// unrecognized values.
func (s FeatureSet) Has(value Feature) bool {
	bit, ok := featureBit(value)
	return ok && s.bits&bit != 0
}

// ContainsAll reports whether s contains every feature in required. An empty
// required set is contained in every set.
func (s FeatureSet) ContainsAll(required FeatureSet) bool {
	return s.bits&required.bits == required.bits
}

// Missing returns the features that belong to required but not to s. It does
// not modify either operand.
func (s FeatureSet) Missing(required FeatureSet) FeatureSet {
	return FeatureSet{bits: required.bits &^ s.bits}
}

// Count returns the number of features in s.
func (s FeatureSet) Count() int {
	return bits.OnesCount64(s.bits)
}

// At returns the feature at index in stable declaration order. It returns the
// zero Feature and false when index is outside [0, Count()). Declaration order
// is a diagnostic iteration order, not an integer-value protocol.
func (s FeatureSet) At(index int) (Feature, bool) {
	if index < 0 {
		return 0, false
	}

	for _, value := range [...]Feature{
		FeatureNativeCondition,
		FeatureNativeExpression,
	} {
		if !s.Has(value) {
			continue
		}
		if index == 0 {
			return value, true
		}
		index--
	}

	return 0, false
}

func featureBit(value Feature) (uint64, bool) {
	switch value {
	case FeatureNativeCondition:
		return 1 << 0, true
	case FeatureNativeExpression:
		return 1 << 1, true
	default:
		return 0, false
	}
}

// Capabilities describes the operators and optional features supported by a
// configured compiler.
type Capabilities struct {
	// Operators contains the compiler's supported standard operations.
	Operators OperatorSet
	// Features contains the compiler's supported optional capabilities.
	Features FeatureSet
}

// Supports reports whether c contains every operator and feature in required.
func (c Capabilities) Supports(required Requirements) bool {
	return c.Operators.ContainsAll(required.Operators) &&
		c.Features.ContainsAll(required.Features)
}

// Missing returns the operators and features in required that c does not
// support. It does not modify c or required.
func (c Capabilities) Missing(required Requirements) Requirements {
	return Requirements{
		Operators: c.Operators.Missing(required.Operators),
		Features:  c.Features.Missing(required.Features),
	}
}

// Requirements describes the operators and optional features needed to
// compile a normalized predicate. Its zero value has no requirements, and its
// value fields do not share mutable backing storage.
type Requirements struct {
	// Operators contains the required standard operations.
	Operators OperatorSet
	// Features contains the required optional capabilities.
	Features FeatureSet
}

// FieldCapabilities describes the standard operations applicable to one
// adapter field.
type FieldCapabilities struct {
	// Operators contains the standard operations applicable to the field.
	Operators OperatorSet
}

// FieldCapabilityResolver optionally exposes field-level operator discovery.
// A compiler's final validation remains authoritative even when it implements
// this interface.
type FieldCapabilityResolver interface {
	// CapabilitiesFor returns the standard operations applicable to field.
	CapabilitiesFor(field any) (FieldCapabilities, error)
}
