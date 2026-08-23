package weave_test

import (
	"errors"
	"fmt"

	"github.com/imbrooklyn/weave"
	"github.com/imbrooklyn/weave/when"
)

type exampleCondition string
type exampleExpression string

type exampleCompiler struct{}

func (exampleCompiler) Compile(
	predicate weave.Predicate[exampleCondition, exampleExpression],
) (exampleCondition, error) {
	root, ok := predicate.Root().AsGroup()
	if !ok {
		return "", weave.ErrInvalidPredicate
	}
	return exampleCondition(fmt.Sprintf(
		"%s with %d children",
		root.Logic(),
		root.ChildCount(),
	)), nil
}

func (exampleCompiler) Capabilities() weave.Capabilities {
	return weave.Capabilities{
		Operators: weave.NewOperatorSet(
			weave.OperatorEQ,
			weave.OperatorIn,
			weave.OperatorIsNull,
			weave.OperatorContains,
		),
	}
}

func Example() {
	factory := weave.NewFactory[exampleCondition, exampleExpression](exampleCompiler{})
	statuses := []int{1, 2}
	keyword := "ann"

	predicate, err := factory.New().
		EQ("tenant_id", 42).
		In("status", statuses, when.NotEmpty[[]int]).
		AnyOf(func(group *weave.Group[exampleExpression]) {
			group.Contains("name", keyword).
				Contains("email", keyword)
		}, when.NotBlank(keyword)).
		Predicate()
	if err != nil {
		fmt.Println(err)
		return
	}

	condition, err := factory.Compile(predicate)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(condition)
	fmt.Println(predicate.Requirements().Operators.Count(), "operators")

	// Output:
	// all_of with 3 children
	// 3 operators
}

func ExampleBuilder_AnyOf() {
	factory := weave.NewFactory[exampleCondition, exampleExpression](exampleCompiler{})
	keyword := ""

	active, err := factory.New().
		AnyOf(func(group *weave.Group[exampleExpression]) {
			group.Contains("name", keyword, when.NotBlank)
		}).
		Predicate()
	if err != nil {
		fmt.Println(err)
		return
	}
	activeRoot, _ := active.Root().AsGroup()
	activeNode, _ := activeRoot.Child(0)
	activeConstant, _ := activeNode.AsConstant()
	fmt.Println("active empty group:", activeNode.Kind(), activeConstant.Value())

	disabled, err := factory.New().
		AnyOf(func(group *weave.Group[exampleExpression]) {
			group.Contains("name", keyword)
		}, when.NotBlank(keyword)).
		Predicate()
	if err != nil {
		fmt.Println(err)
		return
	}
	disabledRoot, _ := disabled.Root().AsGroup()
	fmt.Println("disabled group children:", disabledRoot.ChildCount())

	// Output:
	// active empty group: constant false
	// disabled group children: 0
}

func ExamplePredicate_Requirements() {
	factory := weave.NewFactory[exampleCondition, exampleExpression](exampleCompiler{})
	one := 1

	predicate, err := factory.New().
		In("score", []*int{&one, nil}).
		Predicate()
	if err != nil {
		fmt.Println(err)
		return
	}

	required := predicate.Requirements().Operators
	fmt.Println("in:", required.Has(weave.OperatorIn))
	fmt.Println("is_null:", required.Has(weave.OperatorIsNull))

	// Output:
	// in: true
	// is_null: true
}

func ExampleError() {
	factory := weave.NewFactory[exampleCondition, exampleExpression](exampleCompiler{})
	var value *int

	_, err := factory.New().EQ("score", value).Predicate()
	var detail *weave.Error
	fmt.Println(errors.Is(err, weave.ErrInvalidValue))
	if errors.As(err, &detail) {
		fmt.Println(detail.Code, detail.Phase, detail.Origin.Sequence)
	}

	// Output:
	// true
	// invalid_value construct 1
}
