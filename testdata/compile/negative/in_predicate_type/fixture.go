package inpredicatetype

import (
	"github.com/imbrooklyn/weave"
	"github.com/imbrooklyn/weave/when"
)

type namedInts []int

func invalid(builder *weave.Builder[string, string], predicate when.Predicate[[]int]) {
	builder.In("field", namedInts{1, 2}, predicate)
}
