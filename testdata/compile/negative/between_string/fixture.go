package betweenstring

import "github.com/imbrooklyn/weave"

func invalid(builder *weave.Builder[string, string]) {
	builder.Between("field", "a", "z")
}
