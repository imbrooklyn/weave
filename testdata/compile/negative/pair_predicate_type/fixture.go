package pairpredicatetype

import (
	"github.com/imbrooklyn/weave"
	"github.com/imbrooklyn/weave/when"
)

func invalid(builder *weave.Builder[string, string], predicate when.PairPredicate[int64, int64]) {
	builder.Between[int]("field", 1, 2, predicate)
}
