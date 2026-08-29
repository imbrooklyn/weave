package compilertest

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/imbrooklyn/weave"
)

type subsetCondition func(Record) bool

type subsetExpression func(Record) bool

type subsetCompiler struct{}

func (subsetCompiler) Capabilities() weave.Capabilities {
	return weave.Capabilities{
		Operators: weave.NewOperatorSet(weave.OperatorEQ),
	}
}

func (subsetCompiler) Compile(
	predicate weave.Predicate[subsetCondition, subsetExpression],
) (subsetCondition, error) {
	return compileSubsetNode(predicate.Root())
}

type constantSubsetCompiler struct{}

func (constantSubsetCompiler) Capabilities() weave.Capabilities {
	return weave.Capabilities{}
}

func (constantSubsetCompiler) Compile(
	predicate weave.Predicate[subsetCondition, subsetExpression],
) (subsetCondition, error) {
	return compileSubsetNode(predicate.Root())
}

type subsetResolver struct {
	operators weave.OperatorSet
}

func (resolver subsetResolver) CapabilitiesFor(field any) (weave.FieldCapabilities, error) {
	name, ok := field.(string)
	if !ok || !slices.Contains([]string{
		"number",
		"text",
		"nullable_number",
		"nullable_text",
		"equality_only_text",
	}, name) {
		return weave.FieldCapabilities{}, weave.ErrInvalidField
	}
	return weave.FieldCapabilities{Operators: resolver.operators}, nil
}

type catalogCompiler struct {
	capabilities weave.Capabilities
}

func (compiler catalogCompiler) Capabilities() weave.Capabilities {
	return compiler.capabilities
}

func (catalogCompiler) Compile(weave.Predicate[string, string]) (string, error) {
	return "compiled", nil
}

func TestScenariosRespectCapabilitiesAndFieldApplicability(t *testing.T) {
	operators := weave.NewOperatorSet(
		weave.OperatorEQ,
		weave.OperatorContains,
	)
	harness := scenarioHarness()
	harness.Factory = weave.NewFactory[string, string](catalogCompiler{
		capabilities: weave.Capabilities{
			Operators: operators,
		},
	})
	harness.Resolver = subsetResolver{
		operators: weave.NewOperatorSet(weave.OperatorEQ),
	}

	got := make([]string, 0)
	for _, scenario := range Scenarios(harness) {
		got = append(got, scenario.Name())
		if _, err := scenario.Build(harness.Factory); err != nil {
			t.Fatalf("Scenario %q Build() error = %v", scenario.Name(), err)
		}
	}
	want := []string{
		"constant true root",
		"constant true empty all",
		"constant false empty any",
		"constant true empty none",
		"constant false empty not all",
		"scalar equality",
		"equality",
		"any of",
		"none of is match-set complement",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Scenarios() names = %v, want %v", got, want)
	}
}

func TestScenariosWithoutFactoryPreserveCatalog(t *testing.T) {
	harness := scenarioHarness()
	harness.Factory = nil
	harness.Resolver = subsetResolver{
		operators: weave.NewOperatorSet(weave.OperatorEQ),
	}

	if got, want := len(Scenarios(harness)), 31; got != want {
		t.Fatalf("len(Scenarios()) without Factory = %d, want %d", got, want)
	}
}

func TestRunSupportsCapabilitySubset(t *testing.T) {
	operators := weave.NewOperatorSet(weave.OperatorEQ)
	factory := weave.NewFactory[subsetCondition, subsetExpression](subsetCompiler{})
	Run(t, newSubsetHarness(factory, operators))
}

func TestRunSupportsConstantOnlyCompiler(t *testing.T) {
	factory := weave.NewFactory[subsetCondition, subsetExpression](
		constantSubsetCompiler{},
	)
	Run(t, newSubsetHarness(factory, weave.OperatorSet{}))
}

func newSubsetHarness(
	factory *weave.Factory[subsetCondition, subsetExpression],
	operators weave.OperatorSet,
) Harness[subsetCondition, subsetExpression] {
	return Harness[subsetCondition, subsetExpression]{
		Factory: factory,
		Fields: Fields{
			Number:           "number",
			Text:             "text",
			NullableNumber:   "nullable_number",
			NullableText:     "nullable_text",
			EqualityOnlyText: "equality_only_text",
		},
		Resolver: subsetResolver{operators: operators},
		Execute: func(condition subsetCondition) ([]string, error) {
			if condition == nil {
				return nil, errors.New("nil subset condition")
			}
			ids := make([]string, 0)
			for _, record := range Records() {
				if condition(record) {
					ids = append(ids, record.ID)
				}
			}
			return ids, nil
		},
		DistinguishesMissing: true,
	}
}

func compileSubsetNode(
	node weave.NodeView[subsetCondition, subsetExpression],
) (subsetCondition, error) {
	if constant, ok := node.AsConstant(); ok {
		return func(Record) bool { return constant.Value() }, nil
	}
	if comparison, ok := node.AsComparison(); ok {
		if comparison.Operator() != weave.OperatorEQ {
			return nil, fmt.Errorf("unexpected operator %s", comparison.Operator())
		}
		field, ok := comparison.Field().(string)
		if !ok || !slices.Contains([]string{
			"number",
			"text",
			"nullable_number",
			"nullable_text",
			"equality_only_text",
		}, field) {
			return nil, subsetCompileError(node, weave.CodeInvalidField)
		}
		value, ok := comparison.Value().(int64)
		if !ok {
			return nil, subsetCompileError(node, weave.CodeInvalidValue)
		}
		switch field {
		case "number":
			return func(record Record) bool { return record.Number == value }, nil
		case "nullable_number":
			return func(record Record) bool {
				return record.NullableNumberPresent &&
					record.NullableNumber != nil &&
					*record.NullableNumber == value
			}, nil
		default:
			return nil, subsetCompileError(node, weave.CodeInvalidValue)
		}
	}
	if group, ok := node.AsGroup(); ok {
		children := make([]subsetCondition, group.ChildCount())
		for index := range children {
			child, exists := group.Child(index)
			if !exists {
				return nil, errors.New("missing subset group child")
			}
			compiled, err := compileSubsetNode(child)
			if err != nil {
				return nil, err
			}
			children[index] = compiled
		}
		switch group.Logic() {
		case weave.LogicAllOf:
			return func(record Record) bool {
				for _, child := range children {
					if !child(record) {
						return false
					}
				}
				return true
			}, nil
		case weave.LogicAnyOf:
			return func(record Record) bool {
				for _, child := range children {
					if child(record) {
						return true
					}
				}
				return false
			}, nil
		case weave.LogicNoneOf:
			return func(record Record) bool {
				for _, child := range children {
					if child(record) {
						return false
					}
				}
				return true
			}, nil
		case weave.LogicNotAllOf:
			return func(record Record) bool {
				for _, child := range children {
					if !child(record) {
						return true
					}
				}
				return false
			}, nil
		default:
			return nil, fmt.Errorf("unexpected logic %s", group.Logic())
		}
	}
	return nil, fmt.Errorf("unexpected node kind %s", node.Kind())
}

func subsetCompileError(
	node weave.NodeView[subsetCondition, subsetExpression],
	code weave.ErrorCode,
) error {
	comparison, _ := node.AsComparison()
	return &weave.Error{
		Code:     code,
		Phase:    weave.PhaseValidate,
		Path:     node.Path(),
		Origin:   node.Origin(),
		Operator: comparison.Operator(),
	}
}
