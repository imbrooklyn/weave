package weave_test

import (
	"testing"

	"github.com/imbrooklyn/weave"
)

type diagnosticStringer interface {
	String() string
}

func TestEnumString(t *testing.T) {
	tests := []struct {
		name  string
		value diagnosticStringer
		want  string
	}{
		{name: "operator eq", value: weave.OperatorEQ, want: "eq"},
		{name: "operator neq", value: weave.OperatorNEQ, want: "neq"},
		{name: "operator lt", value: weave.OperatorLT, want: "lt"},
		{name: "operator lte", value: weave.OperatorLTE, want: "lte"},
		{name: "operator gt", value: weave.OperatorGT, want: "gt"},
		{name: "operator gte", value: weave.OperatorGTE, want: "gte"},
		{name: "operator in", value: weave.OperatorIn, want: "in"},
		{name: "operator not in", value: weave.OperatorNotIn, want: "not_in"},
		{name: "operator between", value: weave.OperatorBetween, want: "between"},
		{name: "operator is null", value: weave.OperatorIsNull, want: "is_null"},
		{name: "operator not null", value: weave.OperatorNotNull, want: "not_null"},
		{name: "operator contains", value: weave.OperatorContains, want: "contains"},
		{name: "operator has prefix", value: weave.OperatorHasPrefix, want: "has_prefix"},
		{name: "operator has suffix", value: weave.OperatorHasSuffix, want: "has_suffix"},
		{name: "zero operator", value: weave.Operator(0), want: "operator(0)"},
		{name: "unknown operator", value: weave.Operator(65535), want: "operator(65535)"},
		{name: "kind constant", value: weave.KindConstant, want: "constant"},
		{name: "kind comparison", value: weave.KindComparison, want: "comparison"},
		{name: "kind membership", value: weave.KindMembership, want: "membership"},
		{name: "kind range", value: weave.KindRange, want: "range"},
		{name: "kind null", value: weave.KindNull, want: "null"},
		{name: "kind text", value: weave.KindText, want: "text"},
		{name: "kind group", value: weave.KindGroup, want: "group"},
		{name: "kind native condition", value: weave.KindNativeCondition, want: "native_condition"},
		{name: "kind native expression", value: weave.KindNativeExpression, want: "native_expression"},
		{name: "zero kind", value: weave.Kind(0), want: "kind(0)"},
		{name: "unknown kind", value: weave.Kind(255), want: "kind(255)"},
		{name: "logic all of", value: weave.LogicAllOf, want: "all_of"},
		{name: "logic any of", value: weave.LogicAnyOf, want: "any_of"},
		{name: "logic none of", value: weave.LogicNoneOf, want: "none_of"},
		{name: "logic not all of", value: weave.LogicNotAllOf, want: "not_all_of"},
		{name: "zero logic", value: weave.Logic(0), want: "logic(0)"},
		{name: "unknown logic", value: weave.Logic(255), want: "logic(255)"},
		{name: "feature native condition", value: weave.FeatureNativeCondition, want: "native_condition"},
		{name: "feature native expression", value: weave.FeatureNativeExpression, want: "native_expression"},
		{name: "zero feature", value: weave.Feature(0), want: "feature(0)"},
		{name: "unknown feature", value: weave.Feature(65535), want: "feature(65535)"},
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
