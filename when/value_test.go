package when_test

import (
	"math"
	"testing"
	"time"

	"github.com/imbrooklyn/weave/when"
)

type namedInt int
type namedFloat64 float64
type namedInts []namedInt

type unaryCase[T any] struct {
	name  string
	value T
	want  bool
}

type pairCase[T any] struct {
	name  string
	lower T
	upper T
	want  bool
}

func TestNotZero(t *testing.T) {
	runUnaryCases(t, when.NotZero[int], []unaryCase[int]{
		{name: "zero", value: 0},
		{name: "positive", value: 1, want: true},
		{name: "negative", value: -1, want: true},
	})
	runUnaryCases(t, when.NotZero[string], []unaryCase[string]{
		{name: "empty"},
		{name: "text", value: "value", want: true},
	})
	runUnaryCases(t, when.NotZero[namedInt], []unaryCase[namedInt]{
		{name: "named zero"},
		{name: "named nonzero", value: 3, want: true},
	})
	runUnaryCases(t, when.NotZero[struct{ Value int }], []unaryCase[struct{ Value int }]{
		{name: "comparable struct zero"},
		{name: "comparable struct nonzero", value: struct{ Value int }{Value: 1}, want: true},
	})
}

func TestPositive(t *testing.T) {
	runUnaryCases(t, when.Positive[int64], []unaryCase[int64]{
		{name: "negative", value: -1},
		{name: "zero"},
		{name: "positive", value: 1, want: true},
	})
	runUnaryCases(t, when.Positive[uint], []unaryCase[uint]{
		{name: "unsigned zero"},
		{name: "unsigned positive", value: 1, want: true},
	})
	runUnaryCases(t, when.Positive[uintptr], []unaryCase[uintptr]{
		{name: "uintptr zero"},
		{name: "uintptr positive", value: 1, want: true},
	})
	runUnaryCases(t, when.Positive[float64], []unaryCase[float64]{
		{name: "negative infinity", value: math.Inf(-1)},
		{name: "negative", value: -0.5},
		{name: "negative zero", value: math.Copysign(0, -1)},
		{name: "zero"},
		{name: "positive", value: 0.5, want: true},
		{name: "positive infinity", value: math.Inf(1), want: true},
		{name: "nan", value: math.NaN()},
	})
	runUnaryCases(t, when.Positive[float32], []unaryCase[float32]{
		{name: "float32 nan", value: float32(math.NaN())},
	})
	runUnaryCases(t, when.Positive[namedFloat64], []unaryCase[namedFloat64]{
		{name: "named positive", value: 2, want: true},
		{name: "named nan", value: namedFloat64(math.NaN())},
	})
}

func TestNonNegative(t *testing.T) {
	runUnaryCases(t, when.NonNegative[int], []unaryCase[int]{
		{name: "negative", value: -1},
		{name: "zero", want: true},
		{name: "positive", value: 1, want: true},
	})
	runUnaryCases(t, when.NonNegative[uint16], []unaryCase[uint16]{
		{name: "unsigned zero", want: true},
		{name: "unsigned positive", value: 1, want: true},
	})
	runUnaryCases(t, when.NonNegative[float64], []unaryCase[float64]{
		{name: "negative", value: -0.5},
		{name: "negative zero", value: math.Copysign(0, -1), want: true},
		{name: "zero", want: true},
		{name: "positive", value: 0.5, want: true},
		{name: "nan", value: math.NaN()},
	})
	runUnaryCases(t, when.NonNegative[namedFloat64], []unaryCase[namedFloat64]{
		{name: "named zero", want: true},
		{name: "named nan", value: namedFloat64(math.NaN())},
	})
}

func TestValidRange(t *testing.T) {
	runPairCases(t, when.ValidRange[int], []pairCase[int]{
		{name: "ascending", lower: 1, upper: 2, want: true},
		{name: "equal", lower: 1, upper: 1, want: true},
		{name: "descending", lower: 2, upper: 1},
	})
	runPairCases(t, when.ValidRange[uint], []pairCase[uint]{
		{name: "unsigned ascending", lower: 0, upper: 1, want: true},
		{name: "unsigned descending", lower: 1, upper: 0},
	})
	runPairCases(t, when.ValidRange[float64], []pairCase[float64]{
		{name: "float ascending", lower: -0.5, upper: 0.5, want: true},
		{name: "nan lower", lower: math.NaN(), upper: 1},
		{name: "nan upper", lower: 1, upper: math.NaN()},
		{name: "both nan", lower: math.NaN(), upper: math.NaN()},
	})
	runPairCases(t, when.ValidRange[float32], []pairCase[float32]{
		{name: "float32 nan lower", lower: float32(math.NaN()), upper: 1},
	})
	runPairCases(t, when.ValidRange[namedFloat64], []pairCase[namedFloat64]{
		{name: "named ascending", lower: 1, upper: 2, want: true},
		{name: "named nan", lower: namedFloat64(math.NaN()), upper: 2},
	})
}

func TestNotBlank(t *testing.T) {
	tests := []unaryCase[string]{
		{name: "empty"},
		{name: "ascii whitespace", value: " \t\r\n"},
		{name: "unicode whitespace", value: "\u00a0\u2003\u3000"},
		{name: "text", value: "value", want: true},
		{name: "trimmed text", value: "\u2003value\u3000", want: true},
		{name: "zero width space is not TrimSpace whitespace", value: "\u200b", want: true},
	}
	runUnaryCases(t, when.NotBlank, tests)
}

func TestNotEmpty(t *testing.T) {
	runUnaryCases(t, when.NotEmpty[[]int], []unaryCase[[]int]{
		{name: "nil"},
		{name: "empty", value: []int{}},
		{name: "nonempty", value: []int{0}, want: true},
	})
	runUnaryCases(t, when.NotEmpty[namedInts], []unaryCase[namedInts]{
		{name: "named nil"},
		{name: "named empty", value: namedInts{}},
		{name: "named nonempty", value: namedInts{0}, want: true},
	})
}

func TestNotNil(t *testing.T) {
	value := 0
	tests := []unaryCase[*int]{
		{name: "nil"},
		{name: "non-nil", value: &value, want: true},
	}
	runUnaryCases(t, when.NotNil[int], tests)
}

func TestNotZeroTime(t *testing.T) {
	tests := []unaryCase[time.Time]{
		{name: "zero"},
		{name: "nonzero", value: time.Unix(0, 1).UTC(), want: true},
	}
	runUnaryCases(t, when.NotZeroTime, tests)
}

func TestBooleanPointers(t *testing.T) {
	trueValue := true
	falseValue := false
	tests := []struct {
		name      string
		value     *bool
		wantTrue  bool
		wantFalse bool
	}{
		{name: "nil"},
		{name: "true", value: &trueValue, wantTrue: true},
		{name: "false", value: &falseValue, wantFalse: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := when.True(test.value); got != test.wantTrue {
				t.Errorf("True() = %v, want %v", got, test.wantTrue)
			}
			if got := when.False(test.value); got != test.wantFalse {
				t.Errorf("False() = %v, want %v", got, test.wantFalse)
			}
		})
	}
}

func runUnaryCases[T any](t *testing.T, predicate func(T) bool, tests []unaryCase[T]) {
	t.Helper()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := predicate(test.value); got != test.want {
				t.Fatalf("predicate(%v) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}

func runPairCases[T any](t *testing.T, predicate func(T, T) bool, tests []pairCase[T]) {
	t.Helper()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := predicate(test.lower, test.upper); got != test.want {
				t.Fatalf("predicate(%v, %v) = %v, want %v", test.lower, test.upper, got, test.want)
			}
		})
	}
}
