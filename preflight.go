package weave

import "reflect"

type compileVisit struct {
	node       node
	parentPath NodePath
	childIndex int
	depth      int
}

func validateCompilePredicate[C, E any](state *predicateState) *Error {
	if !validPredicateState(state) {
		return newCompileError(
			CodeInvalidPredicate,
			PhasePreflight,
			NodePath{},
			Origin{},
			0,
			0,
			nil,
		)
	}

	seen := map[node]struct{}{state.root: {}}
	stack := make([]compileVisit, 0, len(state.root.children))
	for index := len(state.root.children) - 1; index >= 0; index-- {
		stack = append(stack, compileVisit{
			node:       state.root.children[index],
			parentPath: state.root.path,
			childIndex: index,
			depth:      1,
		})
	}

	for len(stack) != 0 {
		last := len(stack) - 1
		visit := stack[last]
		stack = stack[:last]

		metadata, ok := inspectSnapshotNode[C, E](visit.node)
		path := appendSnapshotPath(
			visit.parentPath,
			visit.childIndex,
			metadata.kind,
			metadata.logic,
			metadata.operator,
		)
		feature := featureForKind(metadata.kind)
		if !ok {
			return newCompileError(
				CodeInvalidPredicate,
				PhasePreflight,
				path,
				metadata.origin,
				metadata.operator,
				feature,
				nil,
			)
		}
		if metadata.origin.Sequence == 0 ||
			visit.node.nodeOwner() != state ||
			!equalNodePath(visit.node.nodePath(), path) {
			return newCompileError(
				CodeInvalidPredicate,
				PhasePreflight,
				path,
				metadata.origin,
				metadata.operator,
				feature,
				nil,
			)
		}
		if _, exists := seen[visit.node]; exists {
			return newCompileError(
				CodeInvalidPredicate,
				PhasePreflight,
				path,
				metadata.origin,
				metadata.operator,
				feature,
				nil,
			)
		}
		seen[visit.node] = struct{}{}

		if !validCompileNodePayload[C, E](visit.node, visit.depth) {
			return newCompileError(
				CodeInvalidPredicate,
				PhasePreflight,
				path,
				metadata.origin,
				metadata.operator,
				feature,
				nil,
			)
		}
		if visit.depth > MaxPredicateDepth {
			return newCompileError(
				CodeDepthLimit,
				PhasePreflight,
				path,
				metadata.origin,
				metadata.operator,
				feature,
				nil,
			)
		}
		if metadata.kind == KindNativeCondition && visit.depth != 1 {
			return newCompileError(
				CodeNonNestableNative,
				PhasePreflight,
				path,
				metadata.origin,
				0,
				FeatureNativeCondition,
				nil,
			)
		}
		if group, ok := visit.node.(*groupNode); ok {
			for index := len(group.children) - 1; index >= 0; index-- {
				stack = append(stack, compileVisit{
					node:       group.children[index],
					parentPath: path,
					childIndex: index,
					depth:      visit.depth + 1,
				})
			}
		}
	}

	if calculated := calculateRequirements[C, E](state.root); calculated != state.requirements {
		return newCompileError(
			CodeInvalidPredicate,
			PhasePreflight,
			state.root.path,
			Origin{},
			0,
			0,
			nil,
		)
	}
	return nil
}

func validCompileNodePayload[C, E any](value node, depth int) bool {
	switch typed := value.(type) {
	case *constantNode:
		return depth != 1 || !typed.value
	case *comparisonNode:
		return !isNilLike(typed.field) &&
			!isNilLike(typed.value) &&
			reflect.TypeOf(typed.value) == typed.valueType
	case *membershipNode:
		return !isNilLike(typed.field) &&
			!typed.containsNull &&
			len(typed.values) != 0
	case *rangeNode:
		return !isNilLike(typed.field) &&
			reflect.TypeOf(typed.lower) == typed.boundType &&
			reflect.TypeOf(typed.upper) == typed.boundType
	case *nullNode:
		return !isNilLike(typed.field)
	case *textNode:
		return !isNilLike(typed.field)
	case *groupNode:
		return validNormalizedGroup(typed)
	case *nativeConditionNode[C]:
		return true
	case *nativeExpressionNode[E]:
		return true
	default:
		return false
	}
}

func validNormalizedGroup(group *groupNode) bool {
	if group == nil || len(group.children) == 0 {
		return false
	}

	allConstants := true
	identity := identityConstant(group.logic)
	for _, child := range group.children {
		value, constant := constantNodeValue(child)
		if !constant {
			allConstants = false
			continue
		}
		if value == identity {
			return false
		}
	}
	return !allConstants
}

func preflightCapabilities[C, E any](
	state *predicateState,
	capabilities Capabilities,
) *Error {
	if capabilities.Supports(state.requirements) {
		return nil
	}

	stack := []node{state.root}
	for len(stack) != 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]

		switch value := current.(type) {
		case *comparisonNode:
			if !capabilities.Operators.Has(value.operator) {
				return unsupportedOperatorError(value.nodeBase, value.operator)
			}
		case *membershipNode:
			if !capabilities.Operators.Has(value.operator) {
				return unsupportedOperatorError(value.nodeBase, value.operator)
			}
		case *rangeNode:
			if !capabilities.Operators.Has(value.operator) {
				return unsupportedOperatorError(value.nodeBase, value.operator)
			}
		case *nullNode:
			if !capabilities.Operators.Has(value.operator) {
				return unsupportedOperatorError(value.nodeBase, value.operator)
			}
		case *textNode:
			if !capabilities.Operators.Has(value.operator) {
				return unsupportedOperatorError(value.nodeBase, value.operator)
			}
		case *groupNode:
			for index := len(value.children) - 1; index >= 0; index-- {
				stack = append(stack, value.children[index])
			}
		case *nativeConditionNode[C]:
			if !capabilities.Features.Has(FeatureNativeCondition) {
				return unsupportedFeatureError(
					value.nodeBase,
					FeatureNativeCondition,
				)
			}
		case *nativeExpressionNode[E]:
			if !capabilities.Features.Has(FeatureNativeExpression) {
				return unsupportedFeatureError(
					value.nodeBase,
					FeatureNativeExpression,
				)
			}
		}
	}
	return nil
}

func unsupportedOperatorError(base nodeBase, operator Operator) *Error {
	return newCompileError(
		CodeUnsupportedOperator,
		PhasePreflight,
		base.path,
		base.origin,
		operator,
		0,
		nil,
	)
}

func unsupportedFeatureError(base nodeBase, feature Feature) *Error {
	return newCompileError(
		CodeUnsupportedFeature,
		PhasePreflight,
		base.path,
		base.origin,
		0,
		feature,
		nil,
	)
}

func featureForKind(kind Kind) Feature {
	switch kind {
	case KindNativeCondition:
		return FeatureNativeCondition
	case KindNativeExpression:
		return FeatureNativeExpression
	default:
		return 0
	}
}

func equalNodePath(left, right NodePath) bool {
	if len(left.segments) != len(right.segments) {
		return false
	}
	for index := range left.segments {
		if left.segments[index] != right.segments[index] {
			return false
		}
	}
	return true
}
