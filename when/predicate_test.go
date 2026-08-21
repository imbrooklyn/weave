package when_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/imbrooklyn/weave/when"
)

func TestAllAndAnyEmpty(t *testing.T) {
	tests := []struct {
		name      string
		predicate when.Predicate[int]
		want      bool
	}{
		{name: "empty All", predicate: when.All[int](), want: true},
		{name: "empty Any", predicate: when.Any[int]()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.predicate == nil {
				t.Fatal("empty combinator returned nil")
			}
			if got := test.predicate(42); got != test.want {
				t.Fatalf("predicate() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestAllAndAnyReturnNilForNilPredicate(t *testing.T) {
	calls := 0
	predicate := when.Predicate[int](func(int) bool {
		calls++
		return true
	})
	tests := []struct {
		name string
		got  when.Predicate[int]
	}{
		{name: "All", got: when.All(predicate, nil, predicate)},
		{name: "Any", got: when.Any(predicate, nil, predicate)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != nil {
				t.Fatal("combinator with nil argument returned non-nil")
			}
		})
	}
	if calls != 0 {
		t.Fatalf("combinator construction evaluated predicates %d times, want 0", calls)
	}
}

func TestAllEvaluatesLeftToRightAndShortCircuits(t *testing.T) {
	var calls []string
	predicate := when.All(
		recordingPredicate("first", true, &calls),
		recordingPredicate("second", false, &calls),
		recordingPredicate("third", true, &calls),
	)

	if predicate == nil {
		t.Fatal("All() returned nil")
	}
	if predicate(7) {
		t.Fatal("All() = true, want false")
	}
	if want := []string{"first:7", "second:7"}; !slices.Equal(calls, want) {
		t.Fatalf("evaluation order = %v, want %v", calls, want)
	}
}

func TestAnyEvaluatesLeftToRightAndShortCircuits(t *testing.T) {
	var calls []string
	predicate := when.Any(
		recordingPredicate("first", false, &calls),
		recordingPredicate("second", true, &calls),
		recordingPredicate("third", false, &calls),
	)

	if predicate == nil {
		t.Fatal("Any() returned nil")
	}
	if !predicate(9) {
		t.Fatal("Any() = false, want true")
	}
	if want := []string{"first:9", "second:9"}; !slices.Equal(calls, want) {
		t.Fatalf("evaluation order = %v, want %v", calls, want)
	}
}

func TestAllAndAnyEvaluateEveryPredicateWhenNeeded(t *testing.T) {
	var allCalls []string
	all := when.All(
		recordingPredicate("first", true, &allCalls),
		recordingPredicate("second", true, &allCalls),
		recordingPredicate("third", true, &allCalls),
	)
	if all == nil || !all(11) {
		t.Fatal("All() = false or nil, want true")
	}
	if want := []string{"first:11", "second:11", "third:11"}; !slices.Equal(allCalls, want) {
		t.Fatalf("All evaluation order = %v, want %v", allCalls, want)
	}

	var anyCalls []string
	any := when.Any(
		recordingPredicate("first", false, &anyCalls),
		recordingPredicate("second", false, &anyCalls),
		recordingPredicate("third", false, &anyCalls),
	)
	if any == nil || any(12) {
		t.Fatal("Any() = true or nil, want false")
	}
	if want := []string{"first:12", "second:12", "third:12"}; !slices.Equal(anyCalls, want) {
		t.Fatalf("Any evaluation order = %v, want %v", anyCalls, want)
	}
}

func TestAllAndAnySnapshotPredicateSlice(t *testing.T) {
	truePredicate := when.Predicate[int](func(int) bool { return true })
	falsePredicate := when.Predicate[int](func(int) bool { return false })

	allInput := []when.Predicate[int]{truePredicate}
	all := when.All(allInput...)
	allInput[0] = falsePredicate
	if all == nil || !all(0) {
		t.Fatal("All() changed after its input slice was mutated")
	}

	anyInput := []when.Predicate[int]{falsePredicate}
	any := when.Any(anyInput...)
	anyInput[0] = truePredicate
	if any == nil || any(0) {
		t.Fatal("Any() changed after its input slice was mutated")
	}
}

func TestNot(t *testing.T) {
	if got := when.Not[int](nil); got != nil {
		t.Fatal("Not(nil) returned non-nil")
	}

	calls := 0
	predicate := when.Not(when.Predicate[int](func(value int) bool {
		calls++
		return value == 3
	}))
	if predicate == nil {
		t.Fatal("Not() returned nil for a non-nil predicate")
	}
	if predicate(3) {
		t.Fatal("Not() = true, want false")
	}
	if calls != 1 {
		t.Fatalf("wrapped predicate calls = %d, want 1", calls)
	}
	if !predicate(4) {
		t.Fatal("Not() = false, want true")
	}
	if calls != 2 {
		t.Fatalf("wrapped predicate calls = %d, want 2", calls)
	}
}

func TestIfAndPairIf(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{name: "disabled"},
		{name: "enabled", enabled: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			predicate := when.If[struct{ Value int }](test.enabled)
			if got := predicate(struct{ Value int }{Value: 99}); got != test.enabled {
				t.Errorf("If predicate = %v, want %v", got, test.enabled)
			}

			pairPredicate := when.PairIf[string, []int](test.enabled)
			if got := pairPredicate("ignored", []int{1, 2, 3}); got != test.enabled {
				t.Errorf("PairIf predicate = %v, want %v", got, test.enabled)
			}
		})
	}
}

func recordingPredicate(name string, result bool, calls *[]string) when.Predicate[int] {
	return func(value int) bool {
		*calls = append(*calls, name+":"+strconv.Itoa(value))
		return result
	}
}
