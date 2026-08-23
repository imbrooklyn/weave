package whenvalidrangetime

import (
	"time"

	"github.com/imbrooklyn/weave/when"
)

var _ = when.ValidRange(time.Time{}, time.Time{})
