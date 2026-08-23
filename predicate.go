package weave

import (
	"errors"
	"reflect"
	"sort"
)

type predicateSeal struct {
	marker byte
}

var validPredicateSeal = &predicateSeal{marker: 1}

type predicateDomain struct {
	marker byte
}

func newPredicateDomain() *predicateDomain {
	return &predicateDomain{marker: 1}
}

type predicateState struct {
	seal         *predicateSeal
	domain       *predicateDomain
	root         *groupNode
	requirements Requirements
}

// Predicate is a structurally immutable, normalized snapshot of a Builder. Its
// zero value is invalid. Predicate and its node views are safe for concurrent
// reads when callers do not mutate borrowed fields, nested value references,
// native payload references, or expression payloads.
type Predicate[C, E any] struct {
	state *predicateState
}

type predicateStatus uint8

const (
	predicateInvalid predicateStatus = iota
	predicateForeign
	predicateValid
)

// Predicate creates an independent structural snapshot of the Builder's
// current nodes and normalizes empty collections, nullable In, empty groups,
// and identity constants. It returns a zero Predicate when construction or
// snapshot validation fails.
func (b *Builder[C, E]) Predicate() (Predicate[C, E], error) {
	construction := b.ensureState()
	state, snapshotError := snapshotConstruction[C, E](
		construction,
		b.domain,
	)

	diagnostics := make([]*Error, 0, len(construction.errors)+1)
	for _, diagnostic := range construction.errors {
		if diagnostic != nil {
			diagnostics = append(diagnostics, clonePredicateDiagnostic(diagnostic))
		}
	}
	if snapshotError != nil {
		diagnostics = append(
			diagnostics,
			clonePredicateDiagnostic(snapshotError),
		)
	}
	if len(diagnostics) != 0 {
		return Predicate[C, E]{}, joinPredicateDiagnostics(diagnostics)
	}

	return Predicate[C, E]{state: state}, nil
}

// Build snapshots and normalizes the Builder with Predicate, then compiles the
// resulting Predicate through the owning Factory. If Predicate fails, Build
// returns that error without calling the Compiler.
func (b *Builder[C, E]) Build() (C, error) {
	var zero C
	predicate, err := b.Predicate()
	if err != nil {
		return zero, err
	}
	if !validFactory(b.factory) {
		return zero, newCompileError(
			CodeInvalidState,
			PhasePreflight,
			NodePath{},
			Origin{},
			0,
			0,
			nil,
		)
	}
	return b.factory.Compile(predicate)
}

// Root returns the implicit LogicAllOf root. It returns an invalid NodeView
// for a zero or otherwise invalid Predicate value.
func (p Predicate[C, E]) Root() NodeView[C, E] {
	if !validPredicateState(p.state) {
		return NodeView[C, E]{}
	}
	return NodeView[C, E]{state: p.state, node: p.state.root}
}

// Requirements returns an immutable value snapshot of the operators and
// optional features used by the normalized predicate. It returns zero
// Requirements for a zero or otherwise invalid Predicate.
func (p Predicate[C, E]) Requirements() Requirements {
	if !validPredicateState(p.state) {
		return Requirements{}
	}
	return p.state.requirements
}

func (p Predicate[C, E]) statusFor(domain *predicateDomain) predicateStatus {
	if !validPredicateState(p.state) {
		return predicateInvalid
	}
	if domain == nil || p.state.domain != domain {
		return predicateForeign
	}
	return predicateValid
}

func validPredicateState(state *predicateState) bool {
	if state == nil ||
		state.seal != validPredicateSeal ||
		state.domain == nil ||
		state.root == nil ||
		state.root.owner != state ||
		state.root.logic != LogicAllOf ||
		state.root.origin != (Origin{}) {
		return false
	}

	path := state.root.path
	if len(path.segments) != 1 {
		return false
	}
	segment := path.segments[0]
	return segment.kind == PathSegmentRoot &&
		segment.childIndex == 0 &&
		segment.nodeKind == KindGroup &&
		segment.logic == LogicAllOf &&
		segment.operator == 0
}

func snapshotConstruction[C, E any](
	construction *constructionState,
	domain *predicateDomain,
) (*predicateState, *Error) {
	raw, err := cloneConstructionSnapshot[C, E](construction, domain)
	if err != nil {
		return nil, err
	}
	return normalizePredicateState[C, E](raw, domain)
}

func cloneConstructionSnapshot[C, E any](
	construction *constructionState,
	domain *predicateDomain,
) (*predicateState, *Error) {
	if construction == nil ||
		domain == nil ||
		construction.root == nil ||
		construction.root.logic != LogicAllOf ||
		construction.root.origin != (Origin{}) {
		return nil, newSnapshotError(
			CodeInvalidPredicate,
			NodePath{},
			Origin{},
			0,
		)
	}

	state := &predicateState{
		seal:   validPredicateSeal,
		domain: domain,
	}
	rootPath := newNodePath(newRootPathSegment(LogicAllOf))
	root := &groupNode{
		nodeBase: nodeBase{
			owner: state,
			path:  rootPath,
		},
		logic:    LogicAllOf,
		children: make([]node, 0, len(construction.root.children)),
	}
	state.root = root

	visiting := make(map[node]struct{})
	visiting[construction.root] = struct{}{}
	for index, child := range construction.root.children {
		cloned, err := cloneSnapshotNode[C, E](
			child,
			state,
			rootPath,
			index,
			1,
			visiting,
		)
		if err != nil {
			return nil, err
		}
		root.children = append(root.children, cloned)
	}

	return state, nil
}

func cloneSnapshotNode[C, E any](
	source node,
	state *predicateState,
	parentPath NodePath,
	childIndex int,
	depth int,
	visiting map[node]struct{},
) (node, *Error) {
	metadata, ok := inspectSnapshotNode[C, E](source)
	if !ok {
		path := appendSnapshotPath(
			parentPath,
			childIndex,
			metadata.kind,
			metadata.logic,
			metadata.operator,
		)
		return nil, newSnapshotError(
			CodeInvalidPredicate,
			path,
			metadata.origin,
			metadata.operator,
		)
	}

	path := appendSnapshotPath(
		parentPath,
		childIndex,
		metadata.kind,
		metadata.logic,
		metadata.operator,
	)
	if depth > MaxPredicateDepth {
		return nil, newSnapshotError(
			CodeDepthLimit,
			path,
			metadata.origin,
			metadata.operator,
		)
	}
	if metadata.origin.Sequence == 0 {
		return nil, newSnapshotError(
			CodeInvalidPredicate,
			path,
			metadata.origin,
			metadata.operator,
		)
	}

	base := nodeBase{
		origin: metadata.origin,
		owner:  state,
		path:   path,
	}
	switch value := source.(type) {
	case *constantNode:
		return &constantNode{nodeBase: base, value: value.value}, nil
	case *comparisonNode:
		return &comparisonNode{
			nodeBase:  base,
			operator:  value.operator,
			field:     value.field,
			value:     cloneScalarByteSlice(value.value),
			valueType: value.valueType,
		}, nil
	case *membershipNode:
		values := make([]any, len(value.values))
		copy(values, value.values)
		return &membershipNode{
			nodeBase:         base,
			operator:         value.operator,
			field:            value.field,
			values:           values,
			containsNull:     value.containsNull,
			inputSliceType:   value.inputSliceType,
			inputElementType: value.inputElementType,
			elementType:      value.elementType,
		}, nil
	case *rangeNode:
		return &rangeNode{
			nodeBase:  base,
			operator:  value.operator,
			field:     value.field,
			lower:     value.lower,
			upper:     value.upper,
			boundType: value.boundType,
		}, nil
	case *nullNode:
		return &nullNode{
			nodeBase: base,
			operator: value.operator,
			field:    value.field,
		}, nil
	case *textNode:
		return &textNode{
			nodeBase: base,
			operator: value.operator,
			field:    value.field,
			value:    value.value,
		}, nil
	case *groupNode:
		if _, found := visiting[source]; found {
			return nil, newSnapshotError(
				CodeInvalidPredicate,
				path,
				metadata.origin,
				0,
			)
		}
		visiting[source] = struct{}{}
		defer delete(visiting, source)

		group := &groupNode{
			nodeBase: base,
			logic:    value.logic,
			children: make([]node, 0, len(value.children)),
		}
		for index, child := range value.children {
			cloned, err := cloneSnapshotNode[C, E](
				child,
				state,
				path,
				index,
				depth+1,
				visiting,
			)
			if err != nil {
				return nil, err
			}
			group.children = append(group.children, cloned)
		}
		return group, nil
	case *nativeConditionNode[C]:
		if depth != 1 {
			return nil, newSnapshotError(
				CodeNonNestableNative,
				path,
				metadata.origin,
				0,
			)
		}
		return &nativeConditionNode[C]{
			nodeBase:  base,
			condition: cloneTopLevelSlice(value.condition),
		}, nil
	case *nativeExpressionNode[E]:
		return &nativeExpressionNode[E]{
			nodeBase:   base,
			expression: value.expression,
		}, nil
	default:
		return nil, newSnapshotError(
			CodeInvalidPredicate,
			path,
			metadata.origin,
			metadata.operator,
		)
	}
}

type snapshotNodeMetadata struct {
	kind     Kind
	logic    Logic
	operator Operator
	origin   Origin
}

func inspectSnapshotNode[C, E any](source node) (snapshotNodeMetadata, bool) {
	switch value := source.(type) {
	case *constantNode:
		if value == nil {
			return snapshotNodeMetadata{}, false
		}
		return snapshotNodeMetadata{
			kind:   KindConstant,
			origin: value.origin,
		}, true
	case *comparisonNode:
		if value == nil {
			return snapshotNodeMetadata{}, false
		}
		metadata := snapshotNodeMetadata{
			kind:     KindComparison,
			operator: value.operator,
			origin:   value.origin,
		}
		return metadata,
			isComparisonOperator(value.operator) && value.valueType != nil
	case *membershipNode:
		if value == nil {
			return snapshotNodeMetadata{}, false
		}
		metadata := snapshotNodeMetadata{
			kind:     KindMembership,
			operator: value.operator,
			origin:   value.origin,
		}
		return metadata, validSnapshotMembership(value)
	case *rangeNode:
		if value == nil {
			return snapshotNodeMetadata{}, false
		}
		metadata := snapshotNodeMetadata{
			kind:     KindRange,
			operator: value.operator,
			origin:   value.origin,
		}
		return metadata,
			value.operator == OperatorBetween && value.boundType != nil
	case *nullNode:
		if value == nil {
			return snapshotNodeMetadata{}, false
		}
		metadata := snapshotNodeMetadata{
			kind:     KindNull,
			operator: value.operator,
			origin:   value.origin,
		}
		return metadata, isNullOperator(value.operator)
	case *textNode:
		if value == nil {
			return snapshotNodeMetadata{}, false
		}
		metadata := snapshotNodeMetadata{
			kind:     KindText,
			operator: value.operator,
			origin:   value.origin,
		}
		return metadata, isTextOperator(value.operator)
	case *groupNode:
		if value == nil {
			return snapshotNodeMetadata{}, false
		}
		metadata := snapshotNodeMetadata{
			kind:   KindGroup,
			logic:  value.logic,
			origin: value.origin,
		}
		return metadata, isGroupLogic(value.logic)
	case *nativeConditionNode[C]:
		if value == nil {
			return snapshotNodeMetadata{}, false
		}
		return snapshotNodeMetadata{
			kind:   KindNativeCondition,
			origin: value.origin,
		}, true
	case *nativeExpressionNode[E]:
		if value == nil {
			return snapshotNodeMetadata{}, false
		}
		return snapshotNodeMetadata{
			kind:   KindNativeExpression,
			origin: value.origin,
		}, true
	default:
		return snapshotNodeMetadata{}, false
	}
}

func validSnapshotMembership(value *membershipNode) bool {
	if !isMembershipOperator(value.operator) ||
		value.inputSliceType == nil ||
		value.inputSliceType.Kind() != reflect.Slice ||
		value.inputElementType == nil ||
		value.inputSliceType.Elem() != value.inputElementType ||
		value.elementType == nil {
		return false
	}

	if value.inputElementType.Kind() == reflect.Pointer {
		if value.inputElementType.Elem().Kind() == reflect.Pointer ||
			value.elementType != value.inputElementType.Elem() {
			return false
		}
	} else if value.elementType != value.inputElementType {
		return false
	}
	if value.containsNull &&
		(value.operator != OperatorIn ||
			value.inputElementType.Kind() != reflect.Pointer) {
		return false
	}

	for _, element := range value.values {
		if isNilLike(element) {
			return false
		}
		elementType := reflect.TypeOf(element)
		if elementType == nil || !elementType.AssignableTo(value.elementType) {
			return false
		}
	}
	return true
}

func appendSnapshotPath(
	parent NodePath,
	childIndex int,
	kind Kind,
	logic Logic,
	operator Operator,
) NodePath {
	segments := make([]PathSegment, 0, len(parent.segments)+2)
	segments = append(segments, parent.segments...)
	segments = append(segments, newChildPathSegment(childIndex))
	if kind != 0 {
		segments = append(
			segments,
			newNodePathSegment(kind, logic, operator),
		)
	}
	return newNodePath(segments...)
}

func newSnapshotError(
	code ErrorCode,
	path NodePath,
	origin Origin,
	operator Operator,
) *Error {
	return &Error{
		Code:     code,
		Phase:    PhaseNormalize,
		Path:     newNodePath(path.segments...),
		Origin:   origin,
		Operator: operator,
	}
}

func clonePredicateDiagnostic(diagnostic *Error) *Error {
	cloned := *diagnostic
	cloned.Path = newNodePath(diagnostic.Path.segments...)
	return &cloned
}

func joinPredicateDiagnostics(diagnostics []*Error) error {
	sort.SliceStable(diagnostics, func(left, right int) bool {
		return diagnostics[left].Origin.Sequence <
			diagnostics[right].Origin.Sequence
	})
	if len(diagnostics) == 1 {
		return diagnostics[0]
	}

	joined := make([]error, len(diagnostics))
	for index, diagnostic := range diagnostics {
		joined[index] = diagnostic
	}
	return errors.Join(joined...)
}

func isComparisonOperator(operator Operator) bool {
	switch operator {
	case OperatorEQ,
		OperatorNEQ,
		OperatorLT,
		OperatorLTE,
		OperatorGT,
		OperatorGTE:
		return true
	default:
		return false
	}
}

func isMembershipOperator(operator Operator) bool {
	return operator == OperatorIn || operator == OperatorNotIn
}

func isNullOperator(operator Operator) bool {
	return operator == OperatorIsNull || operator == OperatorNotNull
}

func isTextOperator(operator Operator) bool {
	switch operator {
	case OperatorContains, OperatorHasPrefix, OperatorHasSuffix:
		return true
	default:
		return false
	}
}

func isGroupLogic(logic Logic) bool {
	switch logic {
	case LogicAllOf, LogicAnyOf, LogicNoneOf, LogicNotAllOf:
		return true
	default:
		return false
	}
}
