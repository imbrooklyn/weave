package weave

type predicateNormalizer[C, E any] struct{}

func normalizePredicateState[C, E any](
	source *predicateState,
	domain *predicateDomain,
) (*predicateState, *Error) {
	if !validPredicateState(source) || domain == nil {
		return nil, newSnapshotError(
			CodeInvalidPredicate,
			NodePath{},
			Origin{},
			0,
		)
	}

	normalizer := predicateNormalizer[C, E]{}
	children := make([]node, 0, len(source.root.children))
	for _, child := range source.root.children {
		normalized, err := normalizer.normalizeNode(child, 1)
		if err != nil {
			return nil, err
		}
		if value, ok := constantNodeValue(normalized); ok && value {
			continue
		}
		children = append(children, normalized)
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
		children: children,
	}
	state.root = root

	for index, child := range root.children {
		if err := bindNormalizedNode[C, E](
			child,
			state,
			rootPath,
			index,
			1,
		); err != nil {
			return nil, err
		}
	}
	state.requirements = calculateRequirements[C, E](root)
	return state, nil
}

func (predicateNormalizer[C, E]) normalizeNode(
	source node,
	depth int,
) (node, *Error) {
	metadata, ok := inspectSnapshotNode[C, E](source)
	if !ok {
		return nil, newSnapshotError(
			CodeInvalidPredicate,
			snapshotNodePath(source),
			metadata.origin,
			metadata.operator,
		)
	}
	if depth > MaxPredicateDepth {
		return nil, newSnapshotError(
			CodeDepthLimit,
			snapshotNodePath(source),
			metadata.origin,
			metadata.operator,
		)
	}

	base := nodeBase{origin: metadata.origin}
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
		return normalizeMembershipNode(value, base, depth)
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
		return predicateNormalizer[C, E]{}.normalizeGroup(value, base, depth)
	case *nativeConditionNode[C]:
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
			snapshotNodePath(source),
			metadata.origin,
			metadata.operator,
		)
	}
}

func (predicateNormalizer[C, E]) normalizeGroup(
	source *groupNode,
	base nodeBase,
	depth int,
) (node, *Error) {
	children := make([]node, 0, len(source.children))
	allConstants := true
	for _, child := range source.children {
		normalized, err := (predicateNormalizer[C, E]{}).normalizeNode(
			child,
			depth+1,
		)
		if err != nil {
			return nil, err
		}
		if _, ok := constantNodeValue(normalized); !ok {
			allConstants = false
		}
		children = append(children, normalized)
	}

	if allConstants {
		return &constantNode{
			nodeBase: base,
			value:    evaluateConstantGroup(source.logic, children),
		}, nil
	}

	identity := identityConstant(source.logic)
	kept := make([]node, 0, len(children))
	for _, child := range children {
		if value, ok := constantNodeValue(child); ok && value == identity {
			continue
		}
		kept = append(kept, child)
	}
	return &groupNode{
		nodeBase: base,
		logic:    source.logic,
		children: kept,
	}, nil
}

func normalizeMembershipNode(
	source *membershipNode,
	base nodeBase,
	depth int,
) (node, *Error) {
	values := make([]any, len(source.values))
	copy(values, source.values)
	membership := &membershipNode{
		nodeBase:         base,
		operator:         source.operator,
		field:            source.field,
		values:           values,
		inputSliceType:   source.inputSliceType,
		inputElementType: source.inputElementType,
		elementType:      source.elementType,
	}

	if source.containsNull {
		if source.operator != OperatorIn {
			return nil, newSnapshotError(
				CodeInvalidValue,
				snapshotNodePath(source),
				source.origin,
				source.operator,
			)
		}
		nullCheck := &nullNode{
			nodeBase: base,
			operator: OperatorIsNull,
			field:    source.field,
		}
		if len(values) == 0 {
			return nullCheck, nil
		}
		if depth+1 > MaxPredicateDepth {
			return nil, newSnapshotError(
				CodeDepthLimit,
				snapshotNodePath(source),
				source.origin,
				source.operator,
			)
		}
		return &groupNode{
			nodeBase: base,
			logic:    LogicAnyOf,
			children: []node{membership, nullCheck},
		}, nil
	}

	if len(values) == 0 {
		return &constantNode{
			nodeBase: base,
			value:    source.operator == OperatorNotIn,
		}, nil
	}
	return membership, nil
}

func evaluateConstantGroup(logic Logic, children []node) bool {
	switch logic {
	case LogicAllOf:
		for _, child := range children {
			value, _ := constantNodeValue(child)
			if !value {
				return false
			}
		}
		return true
	case LogicAnyOf:
		for _, child := range children {
			value, _ := constantNodeValue(child)
			if value {
				return true
			}
		}
		return false
	case LogicNoneOf:
		return !evaluateConstantGroup(LogicAnyOf, children)
	case LogicNotAllOf:
		return !evaluateConstantGroup(LogicAllOf, children)
	default:
		return false
	}
}

func identityConstant(logic Logic) bool {
	return logic == LogicAllOf || logic == LogicNotAllOf
}

func constantNodeValue(value node) (bool, bool) {
	constant, ok := value.(*constantNode)
	if !ok || constant == nil {
		return false, false
	}
	return constant.value, true
}

func snapshotNodePath(value node) NodePath {
	if isNilNode(value) {
		return NodePath{}
	}
	return newNodePath(value.nodePath().segments...)
}

func bindNormalizedNode[C, E any](
	value node,
	state *predicateState,
	parentPath NodePath,
	childIndex int,
	depth int,
) *Error {
	metadata, ok := inspectSnapshotNode[C, E](value)
	path := appendSnapshotPath(
		parentPath,
		childIndex,
		metadata.kind,
		metadata.logic,
		metadata.operator,
	)
	if !ok {
		return newSnapshotError(
			CodeInvalidPredicate,
			path,
			metadata.origin,
			metadata.operator,
		)
	}
	if depth > MaxPredicateDepth {
		return newSnapshotError(
			CodeDepthLimit,
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
	switch typed := value.(type) {
	case *constantNode:
		typed.nodeBase = base
	case *comparisonNode:
		typed.nodeBase = base
	case *membershipNode:
		typed.nodeBase = base
	case *rangeNode:
		typed.nodeBase = base
	case *nullNode:
		typed.nodeBase = base
	case *textNode:
		typed.nodeBase = base
	case *groupNode:
		typed.nodeBase = base
		for index, child := range typed.children {
			if err := bindNormalizedNode[C, E](
				child,
				state,
				path,
				index,
				depth+1,
			); err != nil {
				return err
			}
		}
	case *nativeConditionNode[C]:
		typed.nodeBase = base
	case *nativeExpressionNode[E]:
		typed.nodeBase = base
	default:
		return newSnapshotError(
			CodeInvalidPredicate,
			path,
			metadata.origin,
			metadata.operator,
		)
	}
	return nil
}

func calculateRequirements[C, E any](root *groupNode) Requirements {
	var requirements Requirements
	stack := []node{root}
	for len(stack) != 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]

		switch value := current.(type) {
		case *comparisonNode:
			addOperatorRequirement(&requirements, value.operator)
		case *membershipNode:
			addOperatorRequirement(&requirements, value.operator)
		case *rangeNode:
			addOperatorRequirement(&requirements, value.operator)
		case *nullNode:
			addOperatorRequirement(&requirements, value.operator)
		case *textNode:
			addOperatorRequirement(&requirements, value.operator)
		case *groupNode:
			for index := len(value.children) - 1; index >= 0; index-- {
				stack = append(stack, value.children[index])
			}
		case *nativeConditionNode[C]:
			addFeatureRequirement(
				&requirements,
				FeatureNativeCondition,
			)
		case *nativeExpressionNode[E]:
			addFeatureRequirement(
				&requirements,
				FeatureNativeExpression,
			)
		}
	}
	return requirements
}

func addOperatorRequirement(requirements *Requirements, operator Operator) {
	if bit, ok := operatorBit(operator); ok {
		requirements.Operators.bits |= bit
	}
}

func addFeatureRequirement(requirements *Requirements, feature Feature) {
	if bit, ok := featureBit(feature); ok {
		requirements.Features.bits |= bit
	}
}
