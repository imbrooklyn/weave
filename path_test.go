package weave

import (
	"slices"
	"testing"
)

func TestOriginHasValueSemantics(t *testing.T) {
	var zero Origin
	if zero.Sequence != 0 || zero.Operator != 0 {
		t.Fatalf("zero Origin = %+v, want zero fields", zero)
	}

	original := Origin{Sequence: 7, Operator: OperatorIn}
	copyOfOriginal := original
	copyOfOriginal.Sequence = 8
	copyOfOriginal.Operator = OperatorEQ

	if original.Sequence != 7 || original.Operator != OperatorIn {
		t.Fatalf("modifying an Origin copy changed the original: %+v", original)
	}
}

func TestPathSegmentKindString(t *testing.T) {
	tests := []struct {
		name  string
		value PathSegmentKind
		want  string
	}{
		{name: "root", value: PathSegmentRoot, want: "root"},
		{name: "child", value: PathSegmentChild, want: "child"},
		{name: "node", value: PathSegmentNode, want: "node"},
		{name: "zero", value: PathSegmentKind(0), want: "path_segment_kind(0)"},
		{name: "unknown", value: PathSegmentKind(255), want: "path_segment_kind(255)"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.value.String(); got != test.want {
				t.Fatalf("String() = %q, want %q", got, test.want)
			}
			if got := test.value.String(); got != test.want {
				t.Fatalf("second String() = %q, want deterministic result %q", got, test.want)
			}
		})
	}
}

func TestPathSegmentAccessors(t *testing.T) {
	tests := []struct {
		name           string
		segment        PathSegment
		wantKind       PathSegmentKind
		wantChildIndex int
		wantChildOK    bool
		wantNodeKind   Kind
		wantLogic      Logic
		wantOperator   Operator
	}{
		{name: "zero"},
		{
			name:         "root",
			segment:      newRootPathSegment(LogicAllOf),
			wantKind:     PathSegmentRoot,
			wantNodeKind: KindGroup,
			wantLogic:    LogicAllOf,
		},
		{
			name:           "child",
			segment:        newChildPathSegment(4),
			wantKind:       PathSegmentChild,
			wantChildIndex: 4,
			wantChildOK:    true,
		},
		{
			name:         "group node",
			segment:      newNodePathSegment(KindGroup, LogicAnyOf, 0),
			wantKind:     PathSegmentNode,
			wantNodeKind: KindGroup,
			wantLogic:    LogicAnyOf,
		},
		{
			name:         "leaf node",
			segment:      newNodePathSegment(KindComparison, 0, OperatorGTE),
			wantKind:     PathSegmentNode,
			wantNodeKind: KindComparison,
			wantOperator: OperatorGTE,
		},
		{
			name:         "malformed leaf logic is hidden",
			segment:      newNodePathSegment(KindComparison, LogicAnyOf, OperatorGTE),
			wantKind:     PathSegmentNode,
			wantNodeKind: KindComparison,
			wantOperator: OperatorGTE,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.segment.Kind(); got != test.wantKind {
				t.Errorf("Kind() = %v, want %v", got, test.wantKind)
			}
			index, ok := test.segment.ChildIndex()
			if index != test.wantChildIndex || ok != test.wantChildOK {
				t.Errorf("ChildIndex() = (%d, %v), want (%d, %v)", index, ok, test.wantChildIndex, test.wantChildOK)
			}
			if got := test.segment.NodeKind(); got != test.wantNodeKind {
				t.Errorf("NodeKind() = %v, want %v", got, test.wantNodeKind)
			}
			if got := test.segment.Logic(); got != test.wantLogic {
				t.Errorf("Logic() = %v, want %v", got, test.wantLogic)
			}
			if got := test.segment.Operator(); got != test.wantOperator {
				t.Errorf("Operator() = %v, want %v", got, test.wantOperator)
			}
		})
	}
}

func TestNodePathString(t *testing.T) {
	tests := []struct {
		name string
		path NodePath
		want string
	}{
		{name: "zero"},
		{name: "constructed empty", path: newNodePath()},
		{
			name: "root",
			path: newNodePath(newRootPathSegment(LogicAllOf)),
			want: "root.allOf",
		},
		{
			name: "nested groups and leaf",
			path: newNodePath(
				newRootPathSegment(LogicAllOf),
				newChildPathSegment(0),
				newNodePathSegment(KindGroup, LogicAnyOf, 0),
				newChildPathSegment(2),
				newNodePathSegment(KindRange, 0, OperatorBetween),
			),
			want: "root.allOf[0].anyOf[2].between",
		},
		{
			name: "none of",
			path: newNodePath(
				newRootPathSegment(LogicAllOf),
				newChildPathSegment(1),
				newNodePathSegment(KindGroup, LogicNoneOf, 0),
			),
			want: "root.allOf[1].noneOf",
		},
		{
			name: "not all of",
			path: newNodePath(
				newRootPathSegment(LogicAllOf),
				newChildPathSegment(3),
				newNodePathSegment(KindGroup, LogicNotAllOf, 0),
			),
			want: "root.allOf[3].notAllOf",
		},
		{
			name: "constant node",
			path: newNodePath(
				newRootPathSegment(LogicAllOf),
				newChildPathSegment(0),
				newNodePathSegment(KindConstant, 0, 0),
			),
			want: "root.allOf[0].constant",
		},
		{
			name: "native expression node",
			path: newNodePath(
				newRootPathSegment(LogicAllOf),
				newChildPathSegment(0),
				newNodePathSegment(KindNativeExpression, 0, 0),
			),
			want: "root.allOf[0].native_expression",
		},
		{
			name: "negative child index is deterministic",
			path: newNodePath(
				newRootPathSegment(LogicAllOf),
				newChildPathSegment(-1),
				newNodePathSegment(KindComparison, 0, 0),
			),
			want: "root.allOf[-1].comparison",
		},
		{
			name: "unknown segment kind",
			path: newNodePath(PathSegment{kind: PathSegmentKind(255)}),
			want: "path_segment_kind(255)",
		},
		{
			name: "unknown logic",
			path: newNodePath(newRootPathSegment(Logic(255))),
			want: "root.logic(255)",
		},
		{
			name: "unknown operator",
			path: newNodePath(newNodePathSegment(KindComparison, 0, Operator(65535))),
			want: "operator(65535)",
		},
		{
			name: "unknown node kind",
			path: newNodePath(newNodePathSegment(Kind(255), 0, 0)),
			want: "kind(255)",
		},
		{
			name: "malformed leaf logic does not mask operator",
			path: newNodePath(newNodePathSegment(KindComparison, LogicAnyOf, OperatorGTE)),
			want: "gte",
		},
		{
			name: "empty node metadata",
			path: newNodePath(newNodePathSegment(0, 0, 0)),
			want: "node",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.path.String(); got != test.want {
				t.Fatalf("String() = %q, want %q", got, test.want)
			}
			if got := test.path.String(); got != test.want {
				t.Fatalf("second String() = %q, want deterministic result %q", got, test.want)
			}
		})
	}
}

func TestNodePathSegmentsAndBounds(t *testing.T) {
	want := []PathSegment{
		newRootPathSegment(LogicAllOf),
		newChildPathSegment(2),
		newNodePathSegment(KindText, 0, OperatorContains),
	}
	path := newNodePath(want...)

	if got := path.SegmentCount(); got != len(want) {
		t.Fatalf("SegmentCount() = %d, want %d", got, len(want))
	}

	got := make([]PathSegment, 0, path.SegmentCount())
	for index := 0; index < path.SegmentCount(); index++ {
		segment, ok := path.Segment(index)
		if !ok {
			t.Fatalf("Segment(%d) returned false below SegmentCount()", index)
		}
		got = append(got, segment)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("segments = %+v, want %+v", got, want)
	}

	for _, index := range []int{-1, path.SegmentCount(), path.SegmentCount() + 10} {
		segment, ok := path.Segment(index)
		if ok || segment != (PathSegment{}) {
			t.Errorf("Segment(%d) = (%+v, %v), want (zero, false)", index, segment, ok)
		}
	}
}

func TestZeroNodePathSegments(t *testing.T) {
	var path NodePath
	if got := path.SegmentCount(); got != 0 {
		t.Fatalf("SegmentCount() = %d, want 0", got)
	}
	for _, index := range []int{-1, 0, int(^uint(0) >> 1)} {
		segment, ok := path.Segment(index)
		if ok || segment != (PathSegment{}) {
			t.Errorf("Segment(%d) = (%+v, %v), want (zero, false)", index, segment, ok)
		}
	}
}

func TestNodePathOwnsSegments(t *testing.T) {
	segments := []PathSegment{
		newRootPathSegment(LogicAllOf),
		newChildPathSegment(0),
		newNodePathSegment(KindNull, 0, OperatorIsNull),
	}
	path := newNodePath(segments...)
	segments[0] = newRootPathSegment(LogicAnyOf)

	if got, want := path.String(), "root.allOf[0].is_null"; got != want {
		t.Fatalf("String() after input mutation = %q, want %q", got, want)
	}

	segment, ok := path.Segment(0)
	if !ok {
		t.Fatal("Segment(0) returned false")
	}
	segment.logic = LogicNoneOf
	if got, want := path.String(), "root.allOf[0].is_null"; got != want {
		t.Fatalf("String() after returned-segment mutation = %q, want %q", got, want)
	}
}
