package when

import (
	"strings"
	"time"
)

// NotZero reports whether value differs from the zero value of its type.
func NotZero[T comparable](value T) bool {
	var zero T
	return value != zero
}

// NotBlank reports whether removing leading and trailing Unicode whitespace
// leaves any text.
func NotBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

// NotEmpty reports whether value contains at least one element. It accepts
// named slice types.
func NotEmpty[S ~[]E, E any](value S) bool {
	return len(value) != 0
}

// NotNil reports whether value is non-nil.
func NotNil[T any](value *T) bool {
	return value != nil
}

// NotZeroTime reports whether value is not the zero time.
func NotZeroTime(value time.Time) bool {
	return !value.IsZero()
}

// True reports whether value is non-nil and points to true.
func True(value *bool) bool {
	return value != nil && *value
}

// False reports whether value is non-nil and points to false.
func False(value *bool) bool {
	return value != nil && !*value
}
