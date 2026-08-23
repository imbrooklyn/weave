package weave

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/imbrooklyn/weave/when"
)

type fuzzNamedFloat64 float64

func FuzzBuilderConstructionSequence(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 6, 14, 20, 18})
	f.Add([]byte{128, 137, 149, 23, 21})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 64 {
			data = data[:64]
		}
		builder := newConstructionBuilder()
		for index, raw := range data {
			addFuzzConstructionCall(builder, index, raw)
		}

		first, firstError := builder.Predicate()
		second, secondError := builder.Predicate()
		firstErrorFingerprint := predicateErrorFingerprint(firstError)
		secondErrorFingerprint := predicateErrorFingerprint(secondError)
		if firstErrorFingerprint != secondErrorFingerprint {
			t.Fatalf(
				"repeated error fingerprints differ: %q and %q",
				firstErrorFingerprint,
				secondErrorFingerprint,
			)
		}
		if firstError != nil {
			if first.Root().Valid() || second.Root().Valid() {
				t.Fatal("failed Predicate() returned a valid root")
			}
			return
		}

		firstFingerprint, ok := predicateViewFingerprint(t, first.Root())
		if !ok {
			t.Fatal("first snapshot contains an invalid view")
		}
		secondFingerprint, ok := predicateViewFingerprint(t, second.Root())
		if !ok {
			t.Fatal("second snapshot contains an invalid view")
		}
		if firstFingerprint != secondFingerprint {
			t.Fatalf(
				"repeated snapshot fingerprints differ: %q and %q",
				firstFingerprint,
				secondFingerprint,
			)
		}
	})
}

func FuzzPredicateDepthBoundary(f *testing.F) {
	for _, depth := range []uint16{0, 1, 2, 127, 128, 129, 130, 65535} {
		f.Add(depth)
	}
	f.Fuzz(func(t *testing.T, rawDepth uint16) {
		depth := int(rawDepth % uint16(MaxPredicateDepth+3))
		builder := newConstructionBuilder()
		switch depth {
		case 0:
		case 1:
			builder.EQ("field", 1)
		default:
			builder.AllOf(nestedConstructionScope(depth - 2))
		}

		predicate, err := builder.Predicate()
		if depth > MaxPredicateDepth {
			if !errors.Is(err, ErrDepthLimit) {
				t.Fatalf("depth %d error = %v, want ErrDepthLimit", depth, err)
			}
			if predicate.Root().Valid() {
				t.Fatalf("depth %d failure returned a valid Predicate", depth)
			}
			return
		}
		if err != nil {
			t.Fatalf("depth %d Predicate() error = %v, want nil", depth, err)
		}
		if got := maximumViewDepth(t, predicate.Root()); got != depth {
			t.Fatalf("maximum view depth = %d, want %d", got, depth)
		}
	})
}

func FuzzNullableMembershipInputs(f *testing.F) {
	f.Add(uint8(0), []byte{})
	f.Add(uint8(0), []byte{1, 0, 2})
	f.Add(uint8(1), []byte{1, 2})
	f.Add(uint8(1), []byte{0})
	f.Add(uint8(2), []byte{1, 0})
	f.Add(uint8(3), []byte{})
	f.Fuzz(func(t *testing.T, rawMode uint8, data []byte) {
		if len(data) > 32 {
			data = data[:32]
		}
		mode := rawMode % 4
		builder := newConstructionBuilder()
		wantError := false
		wantOriginOperator := OperatorIn

		switch mode {
		case 0, 1:
			allocated := make([]int, len(data))
			values := make([]*int, len(data))
			containsNil := false
			for index, raw := range data {
				allocated[index] = int(raw)
				if raw%3 == 0 {
					containsNil = true
					continue
				}
				values[index] = &allocated[index]
			}
			if mode == 0 {
				builder.In("field", values)
			} else {
				wantOriginOperator = OperatorNotIn
				wantError = containsNil
				builder.NotIn("field", values)
			}
		case 2:
			values := make([]any, 0, len(data))
			for _, raw := range data {
				if raw%3 == 0 {
					wantError = true
					values = append(values, nil)
					continue
				}
				if raw%3 == 1 {
					wantError = true
					values = append(values, map[string]int(nil))
					continue
				}
				values = append(values, int(raw))
			}
			builder.In("field", values)
		case 3:
			allocated := make([]int, len(data))
			inner := make([]*int, len(data))
			values := make([]**int, len(data))
			for index, raw := range data {
				allocated[index] = int(raw)
				inner[index] = &allocated[index]
				values[index] = &inner[index]
			}
			wantError = true
			builder.In("field", values)
		}

		predicate, err := builder.Predicate()
		if wantError {
			if !errors.Is(err, ErrInvalidValue) {
				t.Fatalf("mode %d error = %v, want ErrInvalidValue", mode, err)
			}
			if predicate.Root().Valid() {
				t.Fatalf("mode %d failure returned a valid Predicate", mode)
			}
			return
		}
		if err != nil {
			t.Fatalf("mode %d Predicate() error = %v, want nil", mode, err)
		}
		observations := collectViewObservations(t, predicate.Root())
		for index, observation := range observations {
			if index == 0 {
				continue
			}
			wantOrigin := Origin{Sequence: 1, Operator: wantOriginOperator}
			if observation.origin != wantOrigin {
				t.Fatalf("mode %d node %d origin = %#v, want %#v", mode, index, observation.origin, wantOrigin)
			}
		}
		assertMembershipViewsContainNoNil(t, predicate.Root())
	})
}

func FuzzNumericRangeNaNAndDeterminism(f *testing.F) {
	f.Add(uint8(0), math.Float64bits(1), math.Float64bits(2))
	f.Add(uint8(0), math.Float64bits(math.NaN()), math.Float64bits(2))
	f.Add(uint8(0), math.Float64bits(1), math.Float64bits(math.NaN()))
	f.Add(uint8(1), uint64(math.Float32bits(float32(math.NaN()))), uint64(math.Float32bits(1)))
	f.Add(uint8(2), math.Float64bits(2), math.Float64bits(1))

	f.Fuzz(func(t *testing.T, rawMode uint8, lowerBits uint64, upperBits uint64) {
		switch rawMode % 3 {
		case 0:
			fuzzNumericRangeCase(
				t,
				math.Float64frombits(lowerBits),
				math.Float64frombits(upperBits),
			)
		case 1:
			fuzzNumericRangeCase(
				t,
				math.Float32frombits(uint32(lowerBits)),
				math.Float32frombits(uint32(upperBits)),
			)
		case 2:
			fuzzNumericRangeCase(
				t,
				fuzzNamedFloat64(math.Float64frombits(lowerBits)),
				fuzzNamedFloat64(math.Float64frombits(upperBits)),
			)
		}
	})
}

func fuzzNumericRangeCase[T when.Number](t *testing.T, lower T, upper T) {
	t.Helper()
	builder := newConstructionBuilder()
	builder.Between("field", lower, upper)

	first, firstError := builder.Predicate()
	second, secondError := builder.Predicate()
	if firstFingerprint, secondFingerprint := predicateErrorFingerprint(firstError), predicateErrorFingerprint(secondError); firstFingerprint != secondFingerprint {
		t.Fatalf("repeated range errors differ: first %q, second %q", firstFingerprint, secondFingerprint)
	}

	wantError := error(nil)
	if lower != lower || upper != upper {
		wantError = ErrInvalidValue
	} else if lower > upper {
		wantError = ErrInvalidRange
	}
	if wantError != nil {
		if !errors.Is(firstError, wantError) {
			t.Fatalf("Between(%v, %v) error = %v, want %v", lower, upper, firstError, wantError)
		}
		if errors.Is(firstError, ErrCompile) {
			t.Fatal("range construction error unexpectedly matches ErrCompile")
		}
		if first.Root().Valid() || second.Root().Valid() {
			t.Fatal("invalid range returned a valid Predicate")
		}
		return
	}

	if firstError != nil || secondError != nil {
		t.Fatalf("valid range errors = (%v, %v), want nil", firstError, secondError)
	}
	firstFingerprint, ok := predicateViewFingerprint(t, first.Root())
	if !ok {
		t.Fatal("first range snapshot contains an invalid view")
	}
	secondFingerprint, ok := predicateViewFingerprint(t, second.Root())
	if !ok || secondFingerprint != firstFingerprint {
		t.Fatalf("repeated range snapshots differ: first %q, second %q", firstFingerprint, secondFingerprint)
	}
	root, _ := first.Root().AsGroup()
	rangeNode := requireViewChild(t, root, 0, KindRange)
	rangeView, _ := rangeNode.AsRange()
	if rangeView.Lower() != any(lower) || rangeView.Upper() != any(upper) {
		t.Fatalf("range bounds = (%v, %v), want (%v, %v)", rangeView.Lower(), rangeView.Upper(), lower, upper)
	}
	assertRequirementsEqual(t, first.Requirements(), Requirements{
		Operators: NewOperatorSet(OperatorBetween),
	})
}

func FuzzNodePathAccess(f *testing.F) {
	f.Add([]byte{}, int16(0))
	f.Add([]byte{0, 1, 2, 255}, int16(-1))
	f.Add([]byte{3, 7, 11, 19, 127}, int16(4))
	f.Fuzz(func(t *testing.T, data []byte, rawIndex int16) {
		if len(data) > 128 {
			data = data[:128]
		}
		segments := make([]PathSegment, 0, len(data))
		for _, raw := range data {
			switch raw % 3 {
			case 0:
				segments = append(segments, newRootPathSegment(Logic(raw/3)))
			case 1:
				segments = append(segments, newChildPathSegment(int(int8(raw))))
			case 2:
				segments = append(
					segments,
					newNodePathSegment(
						Kind(raw/5),
						Logic(raw/7),
						Operator(raw/11),
					),
				)
			}
		}
		path := newNodePath(segments...)
		wantString := path.String()
		if got := path.String(); got != wantString {
			t.Fatalf("repeated String() = %q, want %q", got, wantString)
		}
		if len(segments) != 0 {
			segments[0] = PathSegment{}
			if got := path.String(); got != wantString {
				t.Fatalf("source mutation changed path String() to %q, want %q", got, wantString)
			}
		}

		indices := []int{
			int(rawIndex),
			-1,
			path.SegmentCount(),
			path.SegmentCount() + 1,
			int(^uint(0) >> 1),
		}
		for _, index := range indices {
			segment, ok := path.Segment(index)
			wantOK := index >= 0 && index < path.SegmentCount()
			if ok != wantOK {
				t.Fatalf("Segment(%d) ok = %t, want %t", index, ok, wantOK)
			}
			if !ok && segment != (PathSegment{}) {
				t.Fatalf("Segment(%d) = %#v, want zero segment", index, segment)
			}
			if ok {
				_ = segment.Kind().String()
				_, _ = segment.ChildIndex()
				_ = segment.NodeKind().String()
				_ = segment.Logic().String()
				_ = segment.Operator().String()
			}
		}
	})
}

func addFuzzConstructionCall(
	builder *Builder[constructionCondition, constructionExpression],
	index int,
	raw byte,
) {
	enabled := raw&0x80 == 0
	value := int(raw)
	switch raw % 24 {
	case 0:
		builder.EQ("field", value, func(int) bool { return enabled })
	case 1:
		builder.NEQ("field", value, func(int) bool { return enabled })
	case 2:
		builder.LT("field", value, func(int) bool { return enabled })
	case 3:
		builder.LTE("field", value, func(int) bool { return enabled })
	case 4:
		builder.GT("field", value, func(int) bool { return enabled })
	case 5:
		builder.GTE("field", value, func(int) bool { return enabled })
	case 6:
		builder.In("field", constructionNumbers{value, value + 1}, func(constructionNumbers) bool { return enabled })
	case 7:
		builder.NotIn("field", constructionNumbers{value, value + 1}, func(constructionNumbers) bool { return enabled })
	case 8:
		builder.Between("field", value, value+1, func(int, int) bool { return enabled })
	case 9:
		builder.IsNull("field", enabled)
	case 10:
		builder.NotNull("field", enabled)
	case 11:
		builder.Contains("field", fmt.Sprintf("value-%d", value), func(string) bool { return enabled })
	case 12:
		builder.HasPrefix("field", fmt.Sprintf("value-%d", value), func(string) bool { return enabled })
	case 13:
		builder.HasSuffix("field", fmt.Sprintf("value-%d", value), func(string) bool { return enabled })
	case 14:
		builder.AllOf(func(group *Group[constructionExpression]) {
			group.EQ("nested", index)
		}, enabled)
	case 15:
		builder.AnyOf(func(group *Group[constructionExpression]) {
			if raw&0x20 != 0 {
				group.Expr(constructionExpression{name: "nested"})
			}
		}, enabled)
	case 16:
		builder.NoneOf(func(group *Group[constructionExpression]) {
			group.NotNull("nested")
		}, enabled)
	case 17:
		builder.NotAllOf(func(group *Group[constructionExpression]) {
			group.Contains("nested", "value")
		}, enabled)
	case 18:
		builder.Native(constructionCondition{fmt.Sprintf("native-%d", index)}, enabled)
	case 19:
		builder.Expr(constructionExpression{name: fmt.Sprintf("expression-%d", index)}, enabled)
	case 20:
		pointer := value
		values := []*int{&pointer}
		if raw&0x20 != 0 {
			values = append(values, nil)
		}
		builder.In("nullable", values, func([]*int) bool { return enabled })
	case 21:
		builder.EQ((*int)(nil), (*int)(nil), func(*int) bool { return enabled })
	case 22:
		builder.EQ("field", value, nil)
	case 23:
		builder.AnyOf(nil, enabled)
	}
}

func predicateErrorFingerprint(err error) string {
	if err == nil {
		return ""
	}
	diagnostics := []error{err}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		diagnostics = joined.Unwrap()
	}
	var builder strings.Builder
	for _, wrapped := range diagnostics {
		var diagnostic *Error
		if !errors.As(wrapped, &diagnostic) {
			fmt.Fprintf(&builder, "%T:%s;", wrapped, wrapped.Error())
			continue
		}
		fmt.Fprintf(
			&builder,
			"%d/%d/%d/%d/%d/%d/%s;",
			diagnostic.Code,
			diagnostic.Phase,
			diagnostic.Origin.Sequence,
			diagnostic.Origin.Operator,
			diagnostic.Operator,
			diagnostic.Feature,
			diagnostic.Path.String(),
		)
	}
	return builder.String()
}

func predicateViewFingerprint[C, E any](
	t *testing.T,
	root NodeView[C, E],
) (string, bool) {
	t.Helper()
	type pendingView struct {
		view  NodeView[C, E]
		depth int
	}
	stack := []pendingView{{view: root}}
	var builder strings.Builder
	for len(stack) != 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if !current.view.Valid() || current.depth > MaxPredicateDepth {
			return "", false
		}
		path := current.view.Path()
		if path.SegmentCount() != 1+current.depth*2 {
			return "", false
		}
		if segment, ok := path.Segment(-1); ok || segment != (PathSegment{}) {
			return "", false
		}
		if segment, ok := path.Segment(path.SegmentCount()); ok || segment != (PathSegment{}) {
			return "", false
		}

		kind := current.view.Kind()
		origin := current.view.Origin()
		logic := Logic(0)
		operator := Operator(0)
		childCount := 0
		if group, ok := current.view.AsGroup(); ok {
			logic = group.Logic()
			childCount = group.ChildCount()
			for index := childCount - 1; index >= 0; index-- {
				child, ok := group.Child(index)
				if !ok {
					return "", false
				}
				stack = append(stack, pendingView{view: child, depth: current.depth + 1})
			}
		} else {
			operator = operatorFromView(t, current.view)
			if membership, ok := current.view.AsMembership(); ok {
				for index := 0; index < membership.ValueCount(); index++ {
					if _, ok := membership.Value(index); !ok {
						return "", false
					}
				}
			}
			if native, ok := current.view.AsNativeCondition(); ok {
				_ = native.Condition()
			}
			if expression, ok := current.view.AsNativeExpression(); ok {
				_ = expression.Expression()
			}
		}
		fmt.Fprintf(
			&builder,
			"%d/%d/%d/%d/%d/%d/%s;",
			kind,
			logic,
			operator,
			origin.Sequence,
			origin.Operator,
			childCount,
			path.String(),
		)
	}
	return builder.String(), true
}

func assertMembershipViewsContainNoNil[C, E any](
	t *testing.T,
	root NodeView[C, E],
) {
	t.Helper()
	stack := []NodeView[C, E]{root}
	for len(stack) != 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if membership, ok := current.AsMembership(); ok {
			for index := 0; index < membership.ValueCount(); index++ {
				value, ok := membership.Value(index)
				if !ok || isNilLike(value) {
					t.Fatalf("membership Value(%d) = (%#v, %t), want non-nil value", index, value, ok)
				}
			}
		}
		if group, ok := current.AsGroup(); ok {
			for index := 0; index < group.ChildCount(); index++ {
				child, ok := group.Child(index)
				if !ok {
					t.Fatalf("Child(%d) failed within bounds", index)
				}
				stack = append(stack, child)
			}
		}
	}
}
