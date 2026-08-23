package weave

import (
	"math"
	"reflect"

	"github.com/imbrooklyn/weave/when"
)

// MaxPredicateDepth is the greatest legal node depth. The implicit root has
// depth zero, and each child edge increases depth by one.
const MaxPredicateDepth = 128

type groupLifecycle uint8

const (
	groupCreated groupLifecycle = iota + 1
	groupActive
	groupFrozen
)

type groupControl struct {
	lifecycle groupLifecycle
}

type constructionState struct {
	root     *groupNode
	sequence uint64
	errors   []*Error
}

func newConstructionState() *constructionState {
	return &constructionState{
		root: &groupNode{
			logic: LogicAllOf,
		},
	}
}

func (s *constructionState) nextOrigin(operator Operator) (Origin, bool) {
	if s.sequence == math.MaxUint64 {
		origin := Origin{Sequence: math.MaxUint64, Operator: operator}
		s.record(&Error{
			Code:     CodeInvalidState,
			Phase:    PhaseConstruct,
			Origin:   origin,
			Operator: operator,
		})
		return origin, false
	}

	s.sequence++
	return Origin{Sequence: s.sequence, Operator: operator}, true
}

func (s *constructionState) record(err *Error) {
	s.errors = append(s.errors, err)
}

type constructionContext struct {
	state       *constructionState
	parent      *groupNode
	parentDepth int
	active      bool
}

func (c constructionContext) begin(operator Operator) (Origin, bool) {
	return c.state.nextOrigin(operator)
}

func (c constructionContext) validateTarget(origin Origin, operator Operator) bool {
	if !c.active {
		c.recordError(CodeInvalidState, origin, operator, nil, nil)
		return false
	}
	if c.parentDepth+1 > MaxPredicateDepth {
		c.recordError(CodeDepthLimit, origin, operator, nil, nil)
		return false
	}
	return true
}

func (c constructionContext) recordError(
	code ErrorCode,
	origin Origin,
	operator Operator,
	fieldType reflect.Type,
	valueType reflect.Type,
) {
	c.state.record(&Error{
		Code:      code,
		Phase:     PhaseConstruct,
		Origin:    origin,
		Operator:  operator,
		FieldType: fieldType,
		ValueType: valueType,
	})
}

func addComparison[T any](
	context constructionContext,
	operator Operator,
	field any,
	value T,
	predicates []when.Predicate[T],
) {
	origin, ok := context.begin(operator)
	if !ok {
		return
	}
	if !includeValue(context, origin, operator, field, value, predicates) {
		return
	}
	if !context.validateTarget(origin, operator) {
		return
	}

	valid := true
	fieldType := reflect.TypeOf(field)
	valueType := reflect.TypeOf(value)
	if isNilLike(field) {
		context.recordError(CodeInvalidField, origin, operator, fieldType, valueType)
		valid = false
	}
	if isNilLike(value) {
		context.recordError(CodeInvalidValue, origin, operator, fieldType, valueType)
		valid = false
	}
	if !valid {
		return
	}

	context.parent.children = append(context.parent.children, &comparisonNode{
		nodeBase:  nodeBase{origin: origin},
		operator:  operator,
		field:     field,
		value:     cloneScalarByteSlice(value),
		valueType: valueType,
	})
}

func addMembership[T any, S ~[]T](
	context constructionContext,
	operator Operator,
	field any,
	values S,
	predicates []when.Predicate[S],
) {
	origin, ok := context.begin(operator)
	if !ok {
		return
	}
	if !includeValue(context, origin, operator, field, values, predicates) {
		return
	}
	if !context.validateTarget(origin, operator) {
		return
	}

	fieldType := reflect.TypeOf(field)
	inputSliceType := reflect.TypeFor[S]()
	inputElementType := reflect.TypeFor[T]()
	valid := true
	if isNilLike(field) {
		context.recordError(
			CodeInvalidField,
			origin,
			operator,
			fieldType,
			inputElementType,
		)
		valid = false
	}

	pointerElements, containsNull, nonNullCount, valuesValid := validateMembershipValues(
		values,
		operator,
		inputElementType,
	)
	if !valuesValid {
		context.recordError(
			CodeInvalidValue,
			origin,
			operator,
			fieldType,
			inputElementType,
		)
		valid = false
	}
	if operator == OperatorIn &&
		containsNull &&
		nonNullCount != 0 &&
		context.parentDepth+2 > MaxPredicateDepth {
		context.recordError(CodeDepthLimit, origin, operator, nil, nil)
		valid = false
	}
	if !valid {
		return
	}

	elementType := inputElementType
	if pointerElements {
		elementType = inputElementType.Elem()
	}
	normalizedValues := cloneMembershipValues(values, pointerElements)
	membership := &membershipNode{
		nodeBase:         nodeBase{origin: origin},
		operator:         operator,
		field:            field,
		values:           normalizedValues,
		containsNull:     containsNull,
		inputSliceType:   inputSliceType,
		inputElementType: inputElementType,
		elementType:      elementType,
	}
	context.parent.children = append(context.parent.children, membership)
}

func validateMembershipValues[T any, S ~[]T](
	values S,
	operator Operator,
	elementType reflect.Type,
) (pointerElements bool, containsNull bool, nonNullCount int, valid bool) {
	pointerElements = elementType.Kind() == reflect.Pointer
	if pointerElements && elementType.Elem().Kind() == reflect.Pointer {
		return pointerElements, false, 0, false
	}

	for index := range values {
		value := reflect.ValueOf(values[index])
		if pointerElements {
			if value.IsNil() {
				if operator == OperatorNotIn {
					return pointerElements, true, nonNullCount, false
				}
				containsNull = true
				continue
			}
			if isNilLike(value.Elem().Interface()) {
				return pointerElements, containsNull, nonNullCount, false
			}
			nonNullCount++
			continue
		}

		if isNilLike(values[index]) {
			return false, false, nonNullCount, false
		}
		nonNullCount++
	}
	return pointerElements, containsNull, nonNullCount, true
}

func cloneMembershipValues[T any, S ~[]T](values S, pointerElements bool) []any {
	cloned := make([]any, 0, len(values))
	for index := range values {
		value := reflect.ValueOf(values[index])
		if pointerElements {
			if value.IsNil() {
				continue
			}
			cloned = append(cloned, value.Elem().Interface())
			continue
		}
		cloned = append(cloned, values[index])
	}
	return cloned
}

func addRange[T when.Number](
	context constructionContext,
	field any,
	lower T,
	upper T,
	predicates []when.PairPredicate[T, T],
) {
	const operator = OperatorBetween
	origin, ok := context.begin(operator)
	if !ok {
		return
	}
	if !includePair(context, origin, operator, field, lower, upper, predicates) {
		return
	}
	if !context.validateTarget(origin, operator) {
		return
	}

	fieldType := reflect.TypeOf(field)
	boundType := reflect.TypeFor[T]()
	valid := true
	if isNilLike(field) {
		context.recordError(CodeInvalidField, origin, operator, fieldType, boundType)
		valid = false
	}
	if lower != lower || upper != upper {
		context.recordError(CodeInvalidValue, origin, operator, fieldType, boundType)
		valid = false
	} else if lower > upper {
		context.recordError(CodeInvalidRange, origin, operator, fieldType, boundType)
		valid = false
	}
	if !valid {
		return
	}

	context.parent.children = append(context.parent.children, &rangeNode{
		nodeBase:  nodeBase{origin: origin},
		operator:  operator,
		field:     field,
		lower:     lower,
		upper:     upper,
		boundType: boundType,
	})
}

func addNull(
	context constructionContext,
	operator Operator,
	field any,
	enabled []bool,
) {
	origin, ok := context.begin(operator)
	if !ok {
		return
	}
	if !includeEnabled(enabled) {
		return
	}
	if !context.validateTarget(origin, operator) {
		return
	}
	if isNilLike(field) {
		context.recordError(
			CodeInvalidField,
			origin,
			operator,
			reflect.TypeOf(field),
			nil,
		)
		return
	}

	context.parent.children = append(context.parent.children, &nullNode{
		nodeBase: nodeBase{origin: origin},
		operator: operator,
		field:    field,
	})
}

func addText(
	context constructionContext,
	operator Operator,
	field any,
	value string,
	predicates []when.Predicate[string],
) {
	origin, ok := context.begin(operator)
	if !ok {
		return
	}
	if !includeValue(context, origin, operator, field, value, predicates) {
		return
	}
	if !context.validateTarget(origin, operator) {
		return
	}
	if isNilLike(field) {
		context.recordError(
			CodeInvalidField,
			origin,
			operator,
			reflect.TypeOf(field),
			reflect.TypeFor[string](),
		)
		return
	}

	context.parent.children = append(context.parent.children, &textNode{
		nodeBase: nodeBase{origin: origin},
		operator: operator,
		field:    field,
		value:    value,
	})
}

func addGroup[E any](
	context constructionContext,
	logic Logic,
	scope Scope[E],
	enabled []bool,
) {
	origin, ok := context.begin(0)
	if !ok {
		return
	}
	if !includeEnabled(enabled) {
		return
	}
	if !context.validateTarget(origin, 0) {
		return
	}
	if scope == nil {
		context.recordError(CodeInvalidPredicate, origin, 0, nil, nil)
		return
	}

	node := &groupNode{
		nodeBase: nodeBase{origin: origin},
		logic:    logic,
	}
	context.parent.children = append(context.parent.children, node)

	control := &groupControl{lifecycle: groupCreated}
	group := &Group[E]{
		state:   context.state,
		node:    node,
		depth:   context.parentDepth + 1,
		control: control,
	}
	control.lifecycle = groupActive
	defer func() {
		control.lifecycle = groupFrozen
	}()
	scope(group)
}

func addNativeCondition[C any](
	context constructionContext,
	condition C,
	enabled []bool,
) {
	origin, ok := context.begin(0)
	if !ok {
		return
	}
	if !includeEnabled(enabled) {
		return
	}
	if !context.validateTarget(origin, 0) {
		return
	}

	context.parent.children = append(context.parent.children, &nativeConditionNode[C]{
		nodeBase:  nodeBase{origin: origin},
		condition: cloneTopLevelSlice(condition),
	})
}

func addNativeExpression[E any](
	context constructionContext,
	expression E,
	enabled []bool,
) {
	origin, ok := context.begin(0)
	if !ok {
		return
	}
	if !includeEnabled(enabled) {
		return
	}
	if !context.validateTarget(origin, 0) {
		return
	}

	context.parent.children = append(context.parent.children, &nativeExpressionNode[E]{
		nodeBase:   nodeBase{origin: origin},
		expression: expression,
	})
}

func includeValue[T any](
	context constructionContext,
	origin Origin,
	operator Operator,
	field any,
	value T,
	predicates []when.Predicate[T],
) bool {
	for _, predicate := range predicates {
		if predicate == nil {
			context.recordError(
				CodeInvalidPredicate,
				origin,
				operator,
				reflect.TypeOf(field),
				reflect.TypeOf(value),
			)
			return false
		}
		if !predicate(value) {
			return false
		}
	}
	return true
}

func includePair[A, B any](
	context constructionContext,
	origin Origin,
	operator Operator,
	field any,
	left A,
	right B,
	predicates []when.PairPredicate[A, B],
) bool {
	for _, predicate := range predicates {
		if predicate == nil {
			context.recordError(
				CodeInvalidPredicate,
				origin,
				operator,
				reflect.TypeOf(field),
				reflect.TypeOf(left),
			)
			return false
		}
		if !predicate(left, right) {
			return false
		}
	}
	return true
}

func includeEnabled(enabled []bool) bool {
	for _, value := range enabled {
		if !value {
			return false
		}
	}
	return true
}

func isNilLike(value any) bool {
	if value == nil {
		return true
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice,
		reflect.UnsafePointer:
		return reflected.IsNil()
	default:
		return false
	}
}

func cloneScalarByteSlice[T any](value T) any {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() ||
		reflected.Kind() != reflect.Slice ||
		reflected.Type().Elem().Kind() != reflect.Uint8 {
		return value
	}

	cloned := reflect.MakeSlice(reflected.Type(), reflected.Len(), reflected.Len())
	reflect.Copy(cloned, reflected)
	return cloned.Interface()
}

func cloneTopLevelSlice[T any](value T) T {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() || reflected.Kind() != reflect.Slice || reflected.IsNil() {
		return value
	}

	cloned := reflect.MakeSlice(reflected.Type(), reflected.Len(), reflected.Len())
	reflect.Copy(cloned, reflected)
	return cloned.Interface().(T)
}
