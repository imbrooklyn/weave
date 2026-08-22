package weave

import "reflect"

// NodeView is a sealed, read-only view of one Predicate node. Its zero value
// is invalid, and callers cannot construct a valid view. NodeView is safe for
// concurrent reads under the Predicate borrowed-payload contract.
type NodeView[C, E any] struct {
	state *predicateState
	node  node
}

// Valid reports whether n identifies a node owned by a valid Predicate.
func (n NodeView[C, E]) Valid() bool {
	if !validPredicateState(n.state) || isNilNode(n.node) {
		return false
	}
	if n.node.nodeOwner() != n.state {
		return false
	}
	metadata, ok := inspectSnapshotNode[C, E](n.node)
	if !ok {
		return false
	}
	return validViewPath(
		n.node.nodePath(),
		metadata,
		n.node == node(n.state.root),
	)
}

// Kind returns the structural family of n. It returns the zero Kind when n is
// invalid.
func (n NodeView[C, E]) Kind() Kind {
	if !n.Valid() {
		return 0
	}
	return n.node.nodeKind()
}

// Origin returns the Builder call origin of n. It returns the zero Origin when
// n is invalid or identifies the implicit root.
func (n NodeView[C, E]) Origin() Origin {
	if !n.Valid() {
		return Origin{}
	}
	return n.node.nodeOrigin()
}

// Path returns the core-owned structural path of n. The returned value owns a
// separate segment backing array. It returns the zero NodePath when n is
// invalid.
func (n NodeView[C, E]) Path() NodePath {
	if !n.Valid() {
		return NodePath{}
	}
	return newNodePath(n.node.nodePath().segments...)
}

// AsConstant returns a ConstantView when n is a constant node.
func (n NodeView[C, E]) AsConstant() (ConstantView, bool) {
	if !n.Valid() {
		return ConstantView{}, false
	}
	value, ok := n.node.(*constantNode)
	if !ok {
		return ConstantView{}, false
	}
	return ConstantView{state: n.state, node: value}, true
}

// AsComparison returns a ComparisonView when n is a comparison node.
func (n NodeView[C, E]) AsComparison() (ComparisonView, bool) {
	if !n.Valid() {
		return ComparisonView{}, false
	}
	value, ok := n.node.(*comparisonNode)
	if !ok {
		return ComparisonView{}, false
	}
	return ComparisonView{state: n.state, node: value}, true
}

// AsMembership returns a MembershipView when n is a membership node.
func (n NodeView[C, E]) AsMembership() (MembershipView, bool) {
	if !n.Valid() {
		return MembershipView{}, false
	}
	value, ok := n.node.(*membershipNode)
	if !ok {
		return MembershipView{}, false
	}
	return MembershipView{state: n.state, node: value}, true
}

// AsRange returns a RangeView when n is a range node.
func (n NodeView[C, E]) AsRange() (RangeView, bool) {
	if !n.Valid() {
		return RangeView{}, false
	}
	value, ok := n.node.(*rangeNode)
	if !ok {
		return RangeView{}, false
	}
	return RangeView{state: n.state, node: value}, true
}

// AsNull returns a NullView when n is an explicit-null test node.
func (n NodeView[C, E]) AsNull() (NullView, bool) {
	if !n.Valid() {
		return NullView{}, false
	}
	value, ok := n.node.(*nullNode)
	if !ok {
		return NullView{}, false
	}
	return NullView{state: n.state, node: value}, true
}

// AsText returns a TextView when n is a literal-text node.
func (n NodeView[C, E]) AsText() (TextView, bool) {
	if !n.Valid() {
		return TextView{}, false
	}
	value, ok := n.node.(*textNode)
	if !ok {
		return TextView{}, false
	}
	return TextView{state: n.state, node: value}, true
}

// AsGroup returns a GroupView when n is a Boolean group node.
func (n NodeView[C, E]) AsGroup() (GroupView[C, E], bool) {
	if !n.Valid() {
		return GroupView[C, E]{}, false
	}
	value, ok := n.node.(*groupNode)
	if !ok {
		return GroupView[C, E]{}, false
	}
	return GroupView[C, E]{state: n.state, node: value}, true
}

// AsNativeCondition returns a NativeConditionView when n is a root native
// condition node.
func (n NodeView[C, E]) AsNativeCondition() (NativeConditionView[C], bool) {
	if !n.Valid() {
		return NativeConditionView[C]{}, false
	}
	value, ok := n.node.(*nativeConditionNode[C])
	if !ok {
		return NativeConditionView[C]{}, false
	}
	return NativeConditionView[C]{state: n.state, node: value}, true
}

// AsNativeExpression returns a NativeExpressionView when n is a native
// expression node.
func (n NodeView[C, E]) AsNativeExpression() (NativeExpressionView[E], bool) {
	if !n.Valid() {
		return NativeExpressionView[E]{}, false
	}
	value, ok := n.node.(*nativeExpressionNode[E])
	if !ok {
		return NativeExpressionView[E]{}, false
	}
	return NativeExpressionView[E]{state: n.state, node: value}, true
}

// ConstantView is a sealed, read-only constant node view.
type ConstantView struct {
	state *predicateState
	node  *constantNode
}

// Value returns the constant's Boolean value. It returns false for a zero or
// invalid view.
func (v ConstantView) Value() bool {
	if !validConstantView(v) {
		return false
	}
	return v.node.value
}

// ComparisonView is a sealed, read-only comparison node view.
type ComparisonView struct {
	state *predicateState
	node  *comparisonNode
}

// Operator returns the comparison operator. It returns the zero Operator for
// a zero or invalid view.
func (v ComparisonView) Operator() Operator {
	if !validComparisonView(v) {
		return 0
	}
	return v.node.operator
}

// Field returns the borrowed field payload. It returns nil for a zero or
// invalid view.
func (v ComparisonView) Field() any {
	if !validComparisonView(v) {
		return nil
	}
	return v.node.field
}

// Value returns the comparison value. A top-level byte slice is copied before
// it is returned; nested references remain borrowed. It returns nil for a zero
// or invalid view.
func (v ComparisonView) Value() any {
	if !validComparisonView(v) {
		return nil
	}
	return cloneScalarByteSlice(v.node.value)
}

// ValueType returns the dynamic comparison value type. It returns nil for a
// zero or invalid view.
func (v ComparisonView) ValueType() reflect.Type {
	if !validComparisonView(v) {
		return nil
	}
	return v.node.valueType
}

// MembershipView is a sealed, read-only membership node view.
type MembershipView struct {
	state *predicateState
	node  *membershipNode
}

// Operator returns the membership operator. It returns the zero Operator for
// a zero or invalid view.
func (v MembershipView) Operator() Operator {
	if !validMembershipView(v) {
		return 0
	}
	return v.node.operator
}

// Field returns the borrowed field payload. It returns nil for a zero or
// invalid view.
func (v MembershipView) Field() any {
	if !validMembershipView(v) {
		return nil
	}
	return v.node.field
}

// ValueCount returns the number of normalized membership values. It returns
// zero for a zero or invalid view.
func (v MembershipView) ValueCount() int {
	if !validMembershipView(v) {
		return 0
	}
	return len(v.node.values)
}

// Value returns the normalized membership value at index. It returns nil and
// false for a negative or out-of-range index or an invalid view.
func (v MembershipView) Value(index int) (any, bool) {
	if !validMembershipView(v) || index < 0 || index >= len(v.node.values) {
		return nil, false
	}
	return v.node.values[index], true
}

// InputSliceType returns the Builder input slice type. It returns nil for a
// zero or invalid view.
func (v MembershipView) InputSliceType() reflect.Type {
	if !validMembershipView(v) {
		return nil
	}
	return v.node.inputSliceType
}

// InputElementType returns the Builder input element type. It returns nil for
// a zero or invalid view.
func (v MembershipView) InputElementType() reflect.Type {
	if !validMembershipView(v) {
		return nil
	}
	return v.node.inputElementType
}

// ElementType returns the normalized non-null element type. It returns nil
// for a zero or invalid view.
func (v MembershipView) ElementType() reflect.Type {
	if !validMembershipView(v) {
		return nil
	}
	return v.node.elementType
}

// RangeView is a sealed, read-only numeric range node view.
type RangeView struct {
	state *predicateState
	node  *rangeNode
}

// Operator returns OperatorBetween. It returns the zero Operator for a zero or
// invalid view.
func (v RangeView) Operator() Operator {
	if !validRangeView(v) {
		return 0
	}
	return v.node.operator
}

// Field returns the borrowed field payload. It returns nil for a zero or
// invalid view.
func (v RangeView) Field() any {
	if !validRangeView(v) {
		return nil
	}
	return v.node.field
}

// Lower returns the inclusive lower bound. It returns nil for a zero or
// invalid view.
func (v RangeView) Lower() any {
	if !validRangeView(v) {
		return nil
	}
	return v.node.lower
}

// Upper returns the inclusive upper bound. It returns nil for a zero or
// invalid view.
func (v RangeView) Upper() any {
	if !validRangeView(v) {
		return nil
	}
	return v.node.upper
}

// BoundType returns the range bound type. It returns nil for a zero or invalid
// view.
func (v RangeView) BoundType() reflect.Type {
	if !validRangeView(v) {
		return nil
	}
	return v.node.boundType
}

// NullView is a sealed, read-only explicit-null test view.
type NullView struct {
	state *predicateState
	node  *nullNode
}

// Operator returns the null-test operator. It returns the zero Operator for a
// zero or invalid view.
func (v NullView) Operator() Operator {
	if !validNullView(v) {
		return 0
	}
	return v.node.operator
}

// Field returns the borrowed field payload. It returns nil for a zero or
// invalid view.
func (v NullView) Field() any {
	if !validNullView(v) {
		return nil
	}
	return v.node.field
}

// TextView is a sealed, read-only literal-text node view.
type TextView struct {
	state *predicateState
	node  *textNode
}

// Operator returns the text operator. It returns the zero Operator for a zero
// or invalid view.
func (v TextView) Operator() Operator {
	if !validTextView(v) {
		return 0
	}
	return v.node.operator
}

// Field returns the borrowed field payload. It returns nil for a zero or
// invalid view.
func (v TextView) Field() any {
	if !validTextView(v) {
		return nil
	}
	return v.node.field
}

// Value returns the literal text value. It returns an empty string for a zero
// or invalid view.
func (v TextView) Value() string {
	if !validTextView(v) {
		return ""
	}
	return v.node.value
}

// GroupView is a sealed, read-only Boolean group view.
type GroupView[C, E any] struct {
	state *predicateState
	node  *groupNode
}

// Logic returns the group's Boolean logic. It returns the zero Logic for a
// zero or invalid view.
func (v GroupView[C, E]) Logic() Logic {
	if !v.valid() {
		return 0
	}
	return v.node.logic
}

// ChildCount returns the number of direct children. It returns zero for a zero
// or invalid view.
func (v GroupView[C, E]) ChildCount() int {
	if !v.valid() {
		return 0
	}
	return len(v.node.children)
}

// Child returns the child at index. It returns an invalid NodeView and false
// for a negative or out-of-range index or an invalid group view.
func (v GroupView[C, E]) Child(index int) (NodeView[C, E], bool) {
	if !v.valid() || index < 0 || index >= len(v.node.children) {
		return NodeView[C, E]{}, false
	}
	child := NodeView[C, E]{state: v.state, node: v.node.children[index]}
	if !child.Valid() {
		return NodeView[C, E]{}, false
	}
	return child, true
}

func (v GroupView[C, E]) valid() bool {
	if v.node == nil {
		return false
	}
	return (NodeView[C, E]{state: v.state, node: v.node}).Valid()
}

// NativeConditionView is a sealed, read-only native condition view.
type NativeConditionView[C any] struct {
	state *predicateState
	node  *nativeConditionNode[C]
}

// Condition returns the native condition. A top-level slice is copied before
// it is returned; nested references remain borrowed. It returns the zero C for
// a zero or invalid view.
func (v NativeConditionView[C]) Condition() C {
	if !v.valid() {
		var zero C
		return zero
	}
	return cloneTopLevelSlice(v.node.condition)
}

func (v NativeConditionView[C]) valid() bool {
	return v.node != nil &&
		validPredicateState(v.state) &&
		v.node.owner == v.state
}

// NativeExpressionView is a sealed, read-only native expression view.
type NativeExpressionView[E any] struct {
	state *predicateState
	node  *nativeExpressionNode[E]
}

// Expression returns the borrowed opaque native expression. It returns the
// zero E for a zero or invalid view.
func (v NativeExpressionView[E]) Expression() E {
	if !v.valid() {
		var zero E
		return zero
	}
	return v.node.expression
}

func (v NativeExpressionView[E]) valid() bool {
	return v.node != nil &&
		validPredicateState(v.state) &&
		v.node.owner == v.state
}

func validConstantView(view ConstantView) bool {
	return view.node != nil &&
		validPredicateState(view.state) &&
		view.node.owner == view.state
}

func validComparisonView(view ComparisonView) bool {
	return view.node != nil &&
		validPredicateState(view.state) &&
		view.node.owner == view.state &&
		isComparisonOperator(view.node.operator) &&
		view.node.valueType != nil
}

func validMembershipView(view MembershipView) bool {
	return view.node != nil &&
		validPredicateState(view.state) &&
		view.node.owner == view.state &&
		isMembershipOperator(view.node.operator) &&
		view.node.inputSliceType != nil &&
		view.node.inputElementType != nil &&
		view.node.elementType != nil
}

func validRangeView(view RangeView) bool {
	return view.node != nil &&
		validPredicateState(view.state) &&
		view.node.owner == view.state &&
		view.node.operator == OperatorBetween &&
		view.node.boundType != nil
}

func validNullView(view NullView) bool {
	return view.node != nil &&
		validPredicateState(view.state) &&
		view.node.owner == view.state &&
		isNullOperator(view.node.operator)
}

func validTextView(view TextView) bool {
	return view.node != nil &&
		validPredicateState(view.state) &&
		view.node.owner == view.state &&
		isTextOperator(view.node.operator)
}

func isNilNode(value node) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	return reflected.Kind() == reflect.Pointer && reflected.IsNil()
}

func validViewPath(
	path NodePath,
	metadata snapshotNodeMetadata,
	isRoot bool,
) bool {
	if len(path.segments) == 0 || len(path.segments)%2 == 0 {
		return false
	}
	if (len(path.segments)-1)/2 > MaxPredicateDepth {
		return false
	}
	root := path.segments[0]
	if root.kind != PathSegmentRoot ||
		root.childIndex != 0 ||
		root.nodeKind != KindGroup ||
		root.logic != LogicAllOf ||
		root.operator != 0 {
		return false
	}
	if isRoot {
		return len(path.segments) == 1 &&
			metadata.kind == KindGroup &&
			metadata.logic == LogicAllOf
	}
	if len(path.segments) < 3 {
		return false
	}

	for index := 1; index < len(path.segments); index += 2 {
		child := path.segments[index]
		if child.kind != PathSegmentChild ||
			child.childIndex < 0 ||
			child.nodeKind != 0 ||
			child.logic != 0 ||
			child.operator != 0 {
			return false
		}
		nodeSegment := path.segments[index+1]
		if nodeSegment.kind != PathSegmentNode || nodeSegment.childIndex != 0 {
			return false
		}
		if index+2 < len(path.segments) &&
			(nodeSegment.nodeKind != KindGroup ||
				!isGroupLogic(nodeSegment.logic) ||
				nodeSegment.operator != 0) {
			return false
		}
	}

	last := path.segments[len(path.segments)-1]
	return last.nodeKind == metadata.kind &&
		last.logic == metadata.logic &&
		last.operator == metadata.operator
}
