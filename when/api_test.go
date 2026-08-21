package when_test

import (
	"time"

	"github.com/imbrooklyn/weave/when"
)

type apiNamedInt int64
type apiNamedFloat float64
type apiNamedInts []apiNamedInt

var (
	_ when.Predicate[apiNamedInt]                      = when.NotZero[apiNamedInt]
	_ when.Predicate[apiNamedInt]                      = when.Positive
	_ when.Predicate[apiNamedInt]                      = when.NonNegative
	_ when.Predicate[apiNamedInts]                     = when.NotEmpty
	_ when.Predicate[*apiNamedInt]                     = when.NotNil[apiNamedInt]
	_ when.Predicate[string]                           = when.NotBlank
	_ when.Predicate[time.Time]                        = when.NotZeroTime
	_ when.Predicate[*bool]                            = when.True
	_ when.Predicate[*bool]                            = when.False
	_ when.PairPredicate[apiNamedFloat, apiNamedFloat] = when.ValidRange
	_ when.Predicate[apiNamedInt]                      = when.All(when.Positive[apiNamedInt], when.NonNegative[apiNamedInt])
	_ when.Predicate[apiNamedInt]                      = when.Any(when.Positive[apiNamedInt], when.NonNegative[apiNamedInt])
	_ when.Predicate[apiNamedInt]                      = when.Not(when.Positive[apiNamedInt])
	_ when.Predicate[apiNamedInt]                      = when.If[apiNamedInt](true)
	_ when.PairPredicate[string, apiNamedInt]          = when.PairIf[string, apiNamedInt](true)
)

func apiAcceptsNamedNumber[T when.Number](value T) bool {
	return when.Positive(value)
}

var (
	_ = apiAcceptsNamedNumber(apiNamedInt(1))
	_ = apiAcceptsNamedNumber(apiNamedFloat(1))
)
