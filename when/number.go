package when

// Number is the set of integer and floating-point types accepted by numeric
// inclusion predicates. It includes named types with one of these underlying
// types.
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// Positive reports whether value is greater than zero. It reports false for
// floating-point NaN values.
func Positive[T Number](value T) bool {
	return value > 0
}

// NonNegative reports whether value is greater than or equal to zero. It
// reports false for floating-point NaN values.
func NonNegative[T Number](value T) bool {
	return value >= 0
}

// ValidRange reports whether neither bound is a floating-point NaN and lower
// is less than or equal to upper.
func ValidRange[T Number](lower, upper T) bool {
	return lower == lower && upper == upper && lower <= upper
}
