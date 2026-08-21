package weave

import "strconv"

// Operator identifies a standard query operation.
//
// The zero value is invalid. An Operator's underlying integer is an
// implementation detail, not a persistence, serialization, or interchange
// protocol.
type Operator uint16

const (
	// OperatorEQ matches values equal to an operand.
	OperatorEQ Operator = iota + 1
	// OperatorNEQ matches non-null, present values unequal to an operand.
	OperatorNEQ
	// OperatorLT matches values less than an operand.
	OperatorLT
	// OperatorLTE matches values less than or equal to an operand.
	OperatorLTE
	// OperatorGT matches values greater than an operand.
	OperatorGT
	// OperatorGTE matches values greater than or equal to an operand.
	OperatorGTE
	// OperatorIn matches values contained in a collection.
	OperatorIn
	// OperatorNotIn matches non-null, present values absent from a collection.
	OperatorNotIn
	// OperatorBetween matches values inside an inclusive range.
	OperatorBetween
	// OperatorIsNull matches fields that are present with an explicit null value.
	OperatorIsNull
	// OperatorNotNull matches fields that are present with a non-null value.
	OperatorNotNull
	// OperatorContains matches fields containing literal text.
	OperatorContains
	// OperatorHasPrefix matches fields beginning with literal text.
	OperatorHasPrefix
	// OperatorHasSuffix matches fields ending with literal text.
	OperatorHasSuffix
)

// String returns the stable English diagnostic identifier for o. It returns
// operator(n), with n in decimal, for zero or an unrecognized value. The
// result is intended for diagnostics, not serialization.
func (o Operator) String() string {
	switch o {
	case OperatorEQ:
		return "eq"
	case OperatorNEQ:
		return "neq"
	case OperatorLT:
		return "lt"
	case OperatorLTE:
		return "lte"
	case OperatorGT:
		return "gt"
	case OperatorGTE:
		return "gte"
	case OperatorIn:
		return "in"
	case OperatorNotIn:
		return "not_in"
	case OperatorBetween:
		return "between"
	case OperatorIsNull:
		return "is_null"
	case OperatorNotNull:
		return "not_null"
	case OperatorContains:
		return "contains"
	case OperatorHasPrefix:
		return "has_prefix"
	case OperatorHasSuffix:
		return "has_suffix"
	default:
		return unknownEnumString("operator", uint64(o))
	}
}

// Kind identifies the structural family of a predicate node.
//
// The zero value represents an invalid node view. A Kind's underlying integer
// is an implementation detail, not a persistence, serialization, or
// interchange protocol.
type Kind uint8

const (
	// KindConstant identifies a Boolean constant node.
	KindConstant Kind = iota + 1
	// KindComparison identifies a comparison node.
	KindComparison
	// KindMembership identifies a collection-membership node.
	KindMembership
	// KindRange identifies an inclusive range node.
	KindRange
	// KindNull identifies an explicit-null test node.
	KindNull
	// KindText identifies a literal-text operation node.
	KindText
	// KindGroup identifies a Boolean group node.
	KindGroup
	// KindNativeCondition identifies a root native-condition node.
	KindNativeCondition
	// KindNativeExpression identifies a native-expression node.
	KindNativeExpression
)

// String returns the stable English diagnostic identifier for k. It returns
// kind(n), with n in decimal, for zero or an unrecognized value. The result is
// intended for diagnostics, not serialization.
func (k Kind) String() string {
	switch k {
	case KindConstant:
		return "constant"
	case KindComparison:
		return "comparison"
	case KindMembership:
		return "membership"
	case KindRange:
		return "range"
	case KindNull:
		return "null"
	case KindText:
		return "text"
	case KindGroup:
		return "group"
	case KindNativeCondition:
		return "native_condition"
	case KindNativeExpression:
		return "native_expression"
	default:
		return unknownEnumString("kind", uint64(k))
	}
}

// Logic identifies how a predicate group combines its children.
//
// The zero value is invalid. A Logic's underlying integer is an implementation
// detail, not a persistence, serialization, or interchange protocol.
type Logic uint8

const (
	// LogicAllOf requires every child to match.
	LogicAllOf Logic = iota + 1
	// LogicAnyOf requires at least one child to match.
	LogicAnyOf
	// LogicNoneOf requires no child to match.
	LogicNoneOf
	// LogicNotAllOf requires at least one child not to match.
	LogicNotAllOf
)

// String returns the stable English diagnostic identifier for l. It returns
// logic(n), with n in decimal, for zero or an unrecognized value. The result is
// intended for diagnostics, not serialization.
func (l Logic) String() string {
	switch l {
	case LogicAllOf:
		return "all_of"
	case LogicAnyOf:
		return "any_of"
	case LogicNoneOf:
		return "none_of"
	case LogicNotAllOf:
		return "not_all_of"
	default:
		return unknownEnumString("logic", uint64(l))
	}
}

// Feature identifies an optional compiler capability that is not represented
// by an Operator.
//
// The zero value is invalid. A Feature's underlying integer is an
// implementation detail, not a persistence, serialization, or interchange
// protocol.
type Feature uint16

const (
	// FeatureNativeCondition permits native conditions at the predicate root.
	FeatureNativeCondition Feature = iota + 1
	// FeatureNativeExpression permits native expressions in Boolean groups.
	FeatureNativeExpression
)

// String returns the stable English diagnostic identifier for f. It returns
// feature(n), with n in decimal, for zero or an unrecognized value. The result
// is intended for diagnostics, not serialization.
func (f Feature) String() string {
	switch f {
	case FeatureNativeCondition:
		return "native_condition"
	case FeatureNativeExpression:
		return "native_expression"
	default:
		return unknownEnumString("feature", uint64(f))
	}
}

func unknownEnumString(name string, value uint64) string {
	return name + "(" + strconv.FormatUint(value, 10) + ")"
}
