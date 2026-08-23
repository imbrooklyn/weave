package whenvalidrangeboundtypes

import "github.com/imbrooklyn/weave/when"

func invalid(lower int, upper int64) {
	_ = when.ValidRange(lower, upper)
}
