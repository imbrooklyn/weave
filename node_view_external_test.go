package weave_test

import (
	"reflect"
	"testing"

	"github.com/imbrooklyn/weave"
)

func TestPredicateAndNodeViewsAreSealedFromExternalConstruction(t *testing.T) {
	values := []any{
		weave.Predicate[int, string]{},
		weave.NodeView[int, string]{},
		weave.ConstantView{},
		weave.ComparisonView{},
		weave.MembershipView{},
		weave.RangeView{},
		weave.NullView{},
		weave.TextView{},
		weave.GroupView[int, string]{},
		weave.NativeConditionView[int]{},
		weave.NativeExpressionView[string]{},
	}

	for _, value := range values {
		typeOfValue := reflect.TypeOf(value)
		for index := 0; index < typeOfValue.NumField(); index++ {
			field := typeOfValue.Field(index)
			if field.PkgPath == "" {
				t.Errorf("%v field %q is exported", typeOfValue, field.Name)
			}
		}
	}

	var predicate weave.Predicate[int, string]
	root := predicate.Root()
	if root.Valid() || root.Kind() != 0 {
		t.Fatalf("externally constructible zero root = %#v, want invalid", root)
	}
	if _, ok := (weave.NodeView[int, string]{}).AsGroup(); ok {
		t.Fatal("externally constructible zero NodeView converted to GroupView")
	}
}

func TestPredicateReadOnlySPIIsUsableFromAnotherPackage(t *testing.T) {
	var builder weave.Builder[[]string, string]
	builder.
		EQ("field", 1).
		AnyOf(func(group *weave.Group[string]) {
			group.Contains("text", "literal")
			group.Expr("expression")
		}).
		Native([]string{"native"})

	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v, want nil", err)
	}
	root := predicate.Root()
	group, ok := root.AsGroup()
	if !root.Valid() || !ok || group.Logic() != weave.LogicAllOf {
		t.Fatalf("root = %#v, want valid AllOf group", root)
	}
	if group.ChildCount() != 3 {
		t.Fatalf("root child count = %d, want 3", group.ChildCount())
	}
	for index := 0; index < group.ChildCount(); index++ {
		child, ok := group.Child(index)
		if !ok || !child.Valid() || child.Path().SegmentCount() == 0 {
			t.Fatalf("Child(%d) = (%#v, %t), want valid located node", index, child, ok)
		}
	}
}
