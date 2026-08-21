package when

// Predicate decides whether a value causes its associated query node to be
// included.
type Predicate[T any] func(T) bool

// PairPredicate decides whether a pair of values causes its associated query
// node to be included.
type PairPredicate[A, B any] func(A, B) bool

// All returns a predicate that evaluates predicates from left to right and
// reports true only when every predicate reports true. Evaluation stops at the
// first false result. All with no arguments reports true. If any argument is
// nil, All returns nil without evaluating any predicate.
func All[T any](predicates ...Predicate[T]) Predicate[T] {
	for _, predicate := range predicates {
		if predicate == nil {
			return nil
		}
	}

	cloned := append([]Predicate[T](nil), predicates...)
	return func(value T) bool {
		for _, predicate := range cloned {
			if !predicate(value) {
				return false
			}
		}
		return true
	}
}

// Any returns a predicate that evaluates predicates from left to right and
// reports true when at least one predicate reports true. Evaluation stops at
// the first true result. Any with no arguments reports false. If any argument
// is nil, Any returns nil without evaluating any predicate.
func Any[T any](predicates ...Predicate[T]) Predicate[T] {
	for _, predicate := range predicates {
		if predicate == nil {
			return nil
		}
	}

	cloned := append([]Predicate[T](nil), predicates...)
	return func(value T) bool {
		for _, predicate := range cloned {
			if predicate(value) {
				return true
			}
		}
		return false
	}
}

// Not returns a predicate that negates predicate. It returns nil when
// predicate is nil.
func Not[T any](predicate Predicate[T]) Predicate[T] {
	if predicate == nil {
		return nil
	}
	return func(value T) bool {
		return !predicate(value)
	}
}

// If returns a predicate that ignores its value and reports enabled.
func If[T any](enabled bool) Predicate[T] {
	return func(T) bool {
		return enabled
	}
}

// PairIf returns a pair predicate that ignores both values and reports
// enabled.
func PairIf[A, B any](enabled bool) PairPredicate[A, B] {
	return func(A, B) bool {
		return enabled
	}
}
