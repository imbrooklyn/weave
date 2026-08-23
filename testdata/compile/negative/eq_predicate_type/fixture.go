package eqpredicatetype

import (
	"github.com/imbrooklyn/weave"
	"github.com/imbrooklyn/weave/when"
)

func invalid(builder *weave.Builder[string, string], value int) {
	builder.EQ("field", value, when.NotBlank)
}
