package weave_test

import (
	"reflect"
	"testing"

	"github.com/imbrooklyn/weave"
	"github.com/imbrooklyn/weave/when"
)

type apiCondition []string
type apiExpression struct{ name string }
type apiNumber int64
type apiNumbers []apiNumber

type apiCompiler struct{}

func (apiCompiler) Compile(
	weave.Predicate[apiCondition, apiExpression],
) (apiCondition, error) {
	return apiCondition{"compiled"}, nil
}

func (apiCompiler) Capabilities() weave.Capabilities {
	return weave.Capabilities{
		Operators: weave.NewOperatorSet(weave.OperatorEQ),
	}
}

var _ weave.Compiler[apiCondition, apiExpression] = apiCompiler{}

func TestBuilderAndGroupAPIShapesCompile(t *testing.T) {
	var builder weave.Builder[apiCondition, apiExpression]
	values := apiNumbers{1, 2}
	expression := apiExpression{name: "expression"}

	builder.
		EQ("field", apiNumber(1), when.NotZero).
		NEQ("field", apiNumber(2), when.Positive).
		LT("field", apiNumber(3), when.NonNegative).
		LTE("field", apiNumber(4)).
		GT("field", apiNumber(5)).
		GTE("field", apiNumber(6)).
		In("field", values, when.NotEmpty).
		NotIn("field", values).
		Between("field", apiNumber(1), apiNumber(2), when.ValidRange).
		IsNull("field").
		NotNull("field", true, true).
		Contains("field", "text", when.NotBlank).
		HasPrefix("field", "prefix").
		HasSuffix("field", "suffix").
		AllOf(func(group *weave.Group[apiExpression]) {
			group.
				EQ("field", apiNumber(1)).
				NEQ("field", apiNumber(2)).
				LT("field", apiNumber(3)).
				LTE("field", apiNumber(4)).
				GT("field", apiNumber(5)).
				GTE("field", apiNumber(6)).
				In("field", values).
				NotIn("field", values).
				Between("field", apiNumber(1), apiNumber(2)).
				IsNull("field").
				NotNull("field").
				Contains("field", "text").
				HasPrefix("field", "prefix").
				HasSuffix("field", "suffix").
				AllOf(func(*weave.Group[apiExpression]) {}).
				AnyOf(func(*weave.Group[apiExpression]) {}).
				NoneOf(func(*weave.Group[apiExpression]) {}).
				NotAllOf(func(*weave.Group[apiExpression]) {}).
				Expr(expression)
		}).
		AnyOf(func(*weave.Group[apiExpression]) {}).
		NoneOf(func(*weave.Group[apiExpression]) {}).
		NotAllOf(func(*weave.Group[apiExpression]) {}).
		Native(apiCondition{"native"}).
		Expr(expression)

	var _ weave.Scope[apiExpression] = func(*weave.Group[apiExpression]) {}
}

func TestIntentionallyAbsentMethodShapes(t *testing.T) {
	builderType := reflect.TypeFor[*weave.Builder[apiCondition, apiExpression]]()
	groupType := reflect.TypeFor[*weave.Group[apiExpression]]()

	for _, name := range []string{"Or", "BetweenOn", "DayRange"} {
		if _, present := builderType.MethodByName(name); present {
			t.Fatalf("Builder unexpectedly exposes %s", name)
		}
		if _, present := groupType.MethodByName(name); present {
			t.Fatalf("Group unexpectedly exposes %s", name)
		}
	}

	for _, name := range []string{"Native", "Predicate", "Build"} {
		if _, present := groupType.MethodByName(name); present {
			t.Fatalf("Group unexpectedly exposes %s", name)
		}
	}
}

func TestCompilerAndFactoryAPIShapesCompile(t *testing.T) {
	factory := weave.NewFactory[apiCondition, apiExpression](apiCompiler{})
	builder := factory.New().EQ("field", apiNumber(1))
	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate failed: %v", err)
	}
	compiled, err := factory.Compile(predicate)
	if err != nil || !reflect.DeepEqual(compiled, apiCondition{"compiled"}) {
		t.Fatalf("Compile = (%#v, %v), want compiled condition", compiled, err)
	}
	built, err := builder.Build()
	if err != nil || !reflect.DeepEqual(built, compiled) {
		t.Fatalf("Build = (%#v, %v), want explicit Compile result", built, err)
	}
	if !factory.Capabilities().Operators.Has(weave.OperatorEQ) {
		t.Fatal("Factory capability snapshot does not contain EQ")
	}
}
