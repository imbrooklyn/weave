package compilertest

import (
	"slices"
	"testing"

	"github.com/imbrooklyn/weave"
)

type scenarioCompiler struct{}

func (scenarioCompiler) Compile(weave.Predicate[string, string]) (string, error) {
	return "compiled", nil
}

func (scenarioCompiler) Capabilities() weave.Capabilities {
	return weave.Capabilities{
		Operators: weave.NewOperatorSet(allOperators...),
		Features: weave.NewFeatureSet(
			weave.FeatureNativeCondition,
			weave.FeatureNativeExpression,
		),
	}
}

func scenarioHarness() Harness[string, string] {
	return Harness[string, string]{
		Factory: weave.NewFactory[string, string](scenarioCompiler{}),
		Fields: Fields{
			Number:           "number",
			Text:             "text",
			NullableNumber:   "nullable_number",
			NullableText:     "nullable_text",
			EqualityOnlyText: "equality_only_text",
		},
		NativeCondition:      func([]string) string { return "native condition" },
		NativeExpression:     func([]string) string { return "native expression" },
		DistinguishesMissing: true,
	}
}

func TestScenariosExposeCanonicalReadOnlyCases(t *testing.T) {
	scenarios := Scenarios(scenarioHarness())
	if got, want := len(scenarios), 28; got != want {
		t.Fatalf("len(Scenarios()) = %d, want %d", got, want)
	}

	wantNames := []string{
		"constant true root",
		"inequality excludes null and missing",
		"nullable membership",
		"three-level logic nesting",
		"native condition in root conjunction",
		"native expression inside group",
	}
	for _, wantName := range wantNames {
		if !slices.ContainsFunc(scenarios, func(scenario Scenario[string, string]) bool {
			return scenario.Name() == wantName
		}) {
			t.Errorf("Scenarios() does not contain %q", wantName)
		}
	}

	missingCount := 0
	for _, scenario := range scenarios {
		if scenario.RequiresDistinctMissing() {
			missingCount++
		}
		condition, err := scenario.Build(scenarioHarness().Factory)
		if err != nil {
			t.Fatalf("Scenario %q Build() error = %v", scenario.Name(), err)
		}
		if condition != "compiled" {
			t.Fatalf("Scenario %q condition = %q, want compiled", scenario.Name(), condition)
		}
	}
	if got, want := missingCount, 3; got != want {
		t.Fatalf("missing-sensitive scenario count = %d, want %d", got, want)
	}
}

func TestScenariosOmitUnavailableNativeCases(t *testing.T) {
	harness := scenarioHarness()
	harness.NativeCondition = nil
	harness.NativeExpression = nil
	if got, want := len(Scenarios(harness)), 26; got != want {
		t.Fatalf("len(Scenarios()) = %d, want %d", got, want)
	}
}

func TestScenariosUseMissingCollapsedMatchSets(t *testing.T) {
	harness := scenarioHarness()
	harness.DistinguishesMissing = false
	scenarios := Scenarios(harness)

	missingCount := 0
	for _, scenario := range scenarios {
		if scenario.RequiresDistinctMissing() {
			missingCount++
		}
		switch scenario.Name() {
		case "explicit null only":
			if !scenario.UsesMissingCollapsedMatchSet() {
				t.Error("collapsed explicit-null scenario is not marked as storage-adjusted")
			}
			if got, want := scenario.ExpectedIDs(), []string{"r03", "r04"}; !slices.Equal(got, want) {
				t.Errorf("collapsed explicit-null IDs = %v, want %v", got, want)
			}
		case "nullable membership":
			if !scenario.UsesMissingCollapsedMatchSet() {
				t.Error("collapsed nullable-In scenario is not marked as storage-adjusted")
			}
			if got, want := scenario.ExpectedIDs(), []string{"r02", "r03", "r04", "r06"}; !slices.Equal(got, want) {
				t.Errorf("collapsed nullable-In IDs = %v, want %v", got, want)
			}
		}
	}
	if got, want := missingCount, 1; got != want {
		t.Fatalf("collapsed missing-sensitive scenario count = %d, want %d", got, want)
	}
}

func TestScenarioExpectedIDsAreIndependent(t *testing.T) {
	scenario := Scenarios(scenarioHarness())[0]
	first := scenario.ExpectedIDs()
	second := scenario.ExpectedIDs()
	if len(first) == 0 || !slices.Equal(first, second) {
		t.Fatalf("ExpectedIDs() copies = (%v, %v), want equal non-empty sets", first, second)
	}
	first[0] = "changed"
	if slices.Equal(first, scenario.ExpectedIDs()) {
		t.Fatal("ExpectedIDs() exposed mutable scenario storage")
	}
}

func TestScenarioBuildRejectsInvalidInputs(t *testing.T) {
	scenario := Scenarios(scenarioHarness())[0]
	if condition, err := scenario.Build(nil); err == nil || condition != "" {
		t.Fatalf("Build(nil) = (%q, %v), want zero condition and error", condition, err)
	}

	var zero Scenario[string, string]
	if condition, err := zero.Build(scenarioHarness().Factory); err == nil || condition != "" {
		t.Fatalf("zero Scenario Build() = (%q, %v), want zero condition and error", condition, err)
	}
}
