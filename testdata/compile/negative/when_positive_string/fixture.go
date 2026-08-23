package whenpositivestring

import "github.com/imbrooklyn/weave/when"

type text string

var _ = when.Positive(text("value"))
