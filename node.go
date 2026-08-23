package weave

import "reflect"

// node is the sealed internal representation used by construction state and
// Predicate snapshots. No public API exposes a node value directly.
type node interface {
	nodeKind() Kind
	nodeOrigin() Origin
	nodeOwner() *predicateState
	nodePath() NodePath
}

type nodeBase struct {
	origin Origin
	owner  *predicateState
	path   NodePath
}

func (b nodeBase) nodeOrigin() Origin {
	return b.origin
}

func (b nodeBase) nodeOwner() *predicateState {
	return b.owner
}

func (b nodeBase) nodePath() NodePath {
	return b.path
}

type constantNode struct {
	nodeBase
	value bool
}

func (*constantNode) nodeKind() Kind {
	return KindConstant
}

type comparisonNode struct {
	nodeBase
	operator  Operator
	field     any
	value     any
	valueType reflect.Type
}

func (*comparisonNode) nodeKind() Kind {
	return KindComparison
}

type membershipNode struct {
	nodeBase
	operator         Operator
	field            any
	values           []any
	containsNull     bool
	inputSliceType   reflect.Type
	inputElementType reflect.Type
	elementType      reflect.Type
}

func (*membershipNode) nodeKind() Kind {
	return KindMembership
}

type rangeNode struct {
	nodeBase
	operator  Operator
	field     any
	lower     any
	upper     any
	boundType reflect.Type
}

func (*rangeNode) nodeKind() Kind {
	return KindRange
}

type nullNode struct {
	nodeBase
	operator Operator
	field    any
}

func (*nullNode) nodeKind() Kind {
	return KindNull
}

type textNode struct {
	nodeBase
	operator Operator
	field    any
	value    string
}

func (*textNode) nodeKind() Kind {
	return KindText
}

type groupNode struct {
	nodeBase
	logic    Logic
	children []node
}

func (*groupNode) nodeKind() Kind {
	return KindGroup
}

type nativeConditionNode[C any] struct {
	nodeBase
	condition C
}

func (*nativeConditionNode[C]) nodeKind() Kind {
	return KindNativeCondition
}

type nativeExpressionNode[E any] struct {
	nodeBase
	expression E
}

func (*nativeExpressionNode[E]) nodeKind() Kind {
	return KindNativeExpression
}
