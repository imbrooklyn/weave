package betweentime

import (
	"time"

	"github.com/imbrooklyn/weave"
)

func invalid(builder *weave.Builder[string, string], lower time.Time, upper time.Time) {
	builder.Between("field", lower, upper)
}
