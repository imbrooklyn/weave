package weave

import (
	"strconv"
	"strings"
)

// Origin identifies the builder call that produced a predicate node or
// construction error. Its zero value represents an unknown origin.
type Origin struct {
	// Sequence is the one-based builder call sequence. Zero means unknown.
	Sequence uint64
	// Operator is the operation requested by the originating builder call. It is
	// zero for groups, native conditions, and native expressions.
	Operator Operator
}

// PathSegmentKind identifies the role of one segment in a NodePath.
//
// The zero value is invalid. A PathSegmentKind's underlying integer is an
// implementation detail, not a persistence, serialization, or interchange
// protocol.
type PathSegmentKind uint8

const (
	// PathSegmentRoot identifies the predicate root.
	PathSegmentRoot PathSegmentKind = iota + 1
	// PathSegmentChild identifies a child index within a group.
	PathSegmentChild
	// PathSegmentNode identifies the child node reached by a preceding index.
	PathSegmentNode
)

// String returns the stable English diagnostic identifier for k. It returns
// path_segment_kind(n), with n in decimal, for zero or an unrecognized value.
// The result is intended for diagnostics, not serialization.
func (k PathSegmentKind) String() string {
	switch k {
	case PathSegmentRoot:
		return "root"
	case PathSegmentChild:
		return "child"
	case PathSegmentNode:
		return "node"
	default:
		return unknownEnumString("path_segment_kind", uint64(k))
	}
}

// PathSegment is one immutable component of a NodePath. Values are created by
// the weave package and expose no mutation API.
type PathSegment struct {
	kind       PathSegmentKind
	childIndex int
	nodeKind   Kind
	logic      Logic
	operator   Operator
}

// Kind returns the segment's role. The zero value denotes an invalid segment.
func (s PathSegment) Kind() PathSegmentKind {
	return s.kind
}

// ChildIndex returns the child index represented by s. It returns zero and
// false when s is not a child segment.
func (s PathSegment) ChildIndex() (int, bool) {
	if s.kind != PathSegmentChild {
		return 0, false
	}
	return s.childIndex, true
}

// NodeKind returns the structural kind described by a root or node segment. It
// returns the zero Kind for every other segment.
func (s PathSegment) NodeKind() Kind {
	if s.kind != PathSegmentRoot && s.kind != PathSegmentNode {
		return 0
	}
	return s.nodeKind
}

// Logic returns the group logic described by a root or node segment. It
// returns the zero Logic for every other segment or for a non-group node.
func (s PathSegment) Logic() Logic {
	if s.kind != PathSegmentRoot && s.kind != PathSegmentNode {
		return 0
	}
	if s.nodeKind != KindGroup {
		return 0
	}
	return s.logic
}

// Operator returns the operation described by a node segment. It returns the
// zero Operator for every other segment or for a node without an operator.
func (s PathSegment) Operator() Operator {
	if s.kind != PathSegmentNode {
		return 0
	}
	return s.operator
}

// NodePath is an immutable structural location within a normalized predicate.
// Its zero value is an empty path. Values are created by the weave package and
// expose no mutation API.
type NodePath struct {
	segments []PathSegment
}

// SegmentCount returns the number of segments in p.
func (p NodePath) SegmentCount() int {
	return len(p.segments)
}

// Segment returns the segment at index. It returns the zero PathSegment and
// false for a negative or out-of-range index.
func (p NodePath) Segment(index int) (PathSegment, bool) {
	if index < 0 || index >= len(p.segments) {
		return PathSegment{}, false
	}
	return p.segments[index], true
}

// String returns a deterministic human-readable path. The result is intended
// for diagnostics, not parsing. It returns an empty string for the zero path.
func (p NodePath) String() string {
	if len(p.segments) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, segment := range p.segments {
		switch segment.Kind() {
		case PathSegmentRoot:
			writePathSeparator(&builder)
			builder.WriteString("root")
			if logic := segment.Logic(); logic != 0 {
				builder.WriteByte('.')
				builder.WriteString(pathLogicString(logic))
			}
		case PathSegmentChild:
			index, _ := segment.ChildIndex()
			builder.WriteByte('[')
			builder.WriteString(strconv.FormatInt(int64(index), 10))
			builder.WriteByte(']')
		case PathSegmentNode:
			writePathSeparator(&builder)
			builder.WriteString(pathNodeString(segment))
		default:
			writePathSeparator(&builder)
			builder.WriteString(segment.Kind().String())
		}
	}
	return builder.String()
}

func newNodePath(segments ...PathSegment) NodePath {
	cloned := make([]PathSegment, len(segments))
	copy(cloned, segments)
	return NodePath{segments: cloned}
}

func newRootPathSegment(logic Logic) PathSegment {
	return PathSegment{
		kind:     PathSegmentRoot,
		nodeKind: KindGroup,
		logic:    logic,
	}
}

func newChildPathSegment(index int) PathSegment {
	return PathSegment{
		kind:       PathSegmentChild,
		childIndex: index,
	}
}

func newNodePathSegment(nodeKind Kind, logic Logic, operator Operator) PathSegment {
	return PathSegment{
		kind:     PathSegmentNode,
		nodeKind: nodeKind,
		logic:    logic,
		operator: operator,
	}
}

func writePathSeparator(builder *strings.Builder) {
	if builder.Len() != 0 {
		builder.WriteByte('.')
	}
}

func pathLogicString(logic Logic) string {
	switch logic {
	case LogicAllOf:
		return "allOf"
	case LogicAnyOf:
		return "anyOf"
	case LogicNoneOf:
		return "noneOf"
	case LogicNotAllOf:
		return "notAllOf"
	default:
		return logic.String()
	}
}

func pathNodeString(segment PathSegment) string {
	if logic := segment.Logic(); logic != 0 {
		return pathLogicString(logic)
	}
	if operator := segment.Operator(); operator != 0 {
		return operator.String()
	}
	if kind := segment.NodeKind(); kind != 0 {
		return kind.String()
	}
	return "node"
}
