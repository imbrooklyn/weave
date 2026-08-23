package betweenboundtypes

import "github.com/imbrooklyn/weave"

func invalid(builder *weave.Builder[string, string], lower int, upper int64) {
	builder.Between("field", lower, upper)
}
