package weave

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

type standardOperatorStressCase struct {
	name       string
	operator   Operator
	kind       Kind
	addBuilder func(*Builder[constructionCondition, constructionExpression], bool)
	addGroup   func(*Group[constructionExpression], bool)
}

func standardOperatorStressCases() []standardOperatorStressCase {
	return []standardOperatorStressCase{
		{
			name:     "EQ",
			operator: OperatorEQ,
			kind:     KindComparison,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], enabled bool) {
				builder.EQ("field", 1, func(int) bool { return enabled })
			},
			addGroup: func(group *Group[constructionExpression], enabled bool) {
				group.EQ("field", 1, func(int) bool { return enabled })
			},
		},
		{
			name:     "NEQ",
			operator: OperatorNEQ,
			kind:     KindComparison,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], enabled bool) {
				builder.NEQ("field", 1, func(int) bool { return enabled })
			},
			addGroup: func(group *Group[constructionExpression], enabled bool) {
				group.NEQ("field", 1, func(int) bool { return enabled })
			},
		},
		{
			name:     "LT",
			operator: OperatorLT,
			kind:     KindComparison,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], enabled bool) {
				builder.LT("field", 1, func(int) bool { return enabled })
			},
			addGroup: func(group *Group[constructionExpression], enabled bool) {
				group.LT("field", 1, func(int) bool { return enabled })
			},
		},
		{
			name:     "LTE",
			operator: OperatorLTE,
			kind:     KindComparison,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], enabled bool) {
				builder.LTE("field", 1, func(int) bool { return enabled })
			},
			addGroup: func(group *Group[constructionExpression], enabled bool) {
				group.LTE("field", 1, func(int) bool { return enabled })
			},
		},
		{
			name:     "GT",
			operator: OperatorGT,
			kind:     KindComparison,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], enabled bool) {
				builder.GT("field", 1, func(int) bool { return enabled })
			},
			addGroup: func(group *Group[constructionExpression], enabled bool) {
				group.GT("field", 1, func(int) bool { return enabled })
			},
		},
		{
			name:     "GTE",
			operator: OperatorGTE,
			kind:     KindComparison,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], enabled bool) {
				builder.GTE("field", 1, func(int) bool { return enabled })
			},
			addGroup: func(group *Group[constructionExpression], enabled bool) {
				group.GTE("field", 1, func(int) bool { return enabled })
			},
		},
		{
			name:     "In",
			operator: OperatorIn,
			kind:     KindMembership,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], enabled bool) {
				builder.In("field", constructionNumbers{1, 2}, func(constructionNumbers) bool { return enabled })
			},
			addGroup: func(group *Group[constructionExpression], enabled bool) {
				group.In("field", constructionNumbers{1, 2}, func(constructionNumbers) bool { return enabled })
			},
		},
		{
			name:     "NotIn",
			operator: OperatorNotIn,
			kind:     KindMembership,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], enabled bool) {
				builder.NotIn("field", constructionNumbers{1, 2}, func(constructionNumbers) bool { return enabled })
			},
			addGroup: func(group *Group[constructionExpression], enabled bool) {
				group.NotIn("field", constructionNumbers{1, 2}, func(constructionNumbers) bool { return enabled })
			},
		},
		{
			name:     "Between",
			operator: OperatorBetween,
			kind:     KindRange,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], enabled bool) {
				builder.Between("field", 1, 2, func(int, int) bool { return enabled })
			},
			addGroup: func(group *Group[constructionExpression], enabled bool) {
				group.Between("field", 1, 2, func(int, int) bool { return enabled })
			},
		},
		{
			name:     "IsNull",
			operator: OperatorIsNull,
			kind:     KindNull,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], enabled bool) {
				builder.IsNull("field", enabled)
			},
			addGroup: func(group *Group[constructionExpression], enabled bool) {
				group.IsNull("field", enabled)
			},
		},
		{
			name:     "NotNull",
			operator: OperatorNotNull,
			kind:     KindNull,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], enabled bool) {
				builder.NotNull("field", enabled)
			},
			addGroup: func(group *Group[constructionExpression], enabled bool) {
				group.NotNull("field", enabled)
			},
		},
		{
			name:     "Contains",
			operator: OperatorContains,
			kind:     KindText,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], enabled bool) {
				builder.Contains("field", "value", func(string) bool { return enabled })
			},
			addGroup: func(group *Group[constructionExpression], enabled bool) {
				group.Contains("field", "value", func(string) bool { return enabled })
			},
		},
		{
			name:     "HasPrefix",
			operator: OperatorHasPrefix,
			kind:     KindText,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], enabled bool) {
				builder.HasPrefix("field", "value", func(string) bool { return enabled })
			},
			addGroup: func(group *Group[constructionExpression], enabled bool) {
				group.HasPrefix("field", "value", func(string) bool { return enabled })
			},
		},
		{
			name:     "HasSuffix",
			operator: OperatorHasSuffix,
			kind:     KindText,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], enabled bool) {
				builder.HasSuffix("field", "value", func(string) bool { return enabled })
			},
			addGroup: func(group *Group[constructionExpression], enabled bool) {
				group.HasSuffix("field", "value", func(string) bool { return enabled })
			},
		},
	}
}

func TestEveryOperatorIncludedDisabledAndNested(t *testing.T) {
	for _, test := range standardOperatorStressCases() {
		t.Run(test.name, func(t *testing.T) {
			t.Run("root included", func(t *testing.T) {
				builder := newConstructionBuilder()
				test.addBuilder(builder, true)
				predicate, err := builder.Predicate()
				if err != nil {
					t.Fatalf("Predicate() error = %v, want nil", err)
				}
				root, _ := predicate.Root().AsGroup()
				child := requireViewChild(t, root, 0, test.kind)
				if got := operatorFromView(t, child); got != test.operator {
					t.Fatalf("view operator = %v, want %v", got, test.operator)
				}
				wantOrigin := Origin{Sequence: 1, Operator: test.operator}
				if child.Origin() != wantOrigin {
					t.Fatalf("origin = %#v, want %#v", child.Origin(), wantOrigin)
				}
				wantPath := "root.allOf[0]." + test.operator.String()
				if got := child.Path().String(); got != wantPath {
					t.Fatalf("path = %q, want %q", got, wantPath)
				}
			})

			t.Run("root disabled", func(t *testing.T) {
				builder := newConstructionBuilder()
				test.addBuilder(builder, false)
				if builder.state.sequence != 1 ||
					len(builder.state.root.children) != 0 ||
					len(builder.state.errors) != 0 {
					t.Fatalf(
						"disabled state = sequence %d, %d children, %d errors",
						builder.state.sequence,
						len(builder.state.root.children),
						len(builder.state.errors),
					)
				}
				predicate, err := builder.Predicate()
				if err != nil {
					t.Fatalf("Predicate() error = %v, want nil", err)
				}
				root, _ := predicate.Root().AsGroup()
				if root.ChildCount() != 0 {
					t.Fatalf("disabled snapshot child count = %d, want 0", root.ChildCount())
				}
			})

			t.Run("group included", func(t *testing.T) {
				builder := newConstructionBuilder()
				builder.AllOf(func(group *Group[constructionExpression]) {
					test.addGroup(group, true)
				})
				predicate, err := builder.Predicate()
				if err != nil {
					t.Fatalf("Predicate() error = %v, want nil", err)
				}
				root, _ := predicate.Root().AsGroup()
				groupNodeView := requireViewChild(t, root, 0, KindGroup)
				group, _ := groupNodeView.AsGroup()
				child := requireViewChild(t, group, 0, test.kind)
				if got := operatorFromView(t, child); got != test.operator {
					t.Fatalf("view operator = %v, want %v", got, test.operator)
				}
				wantOrigin := Origin{Sequence: 2, Operator: test.operator}
				if child.Origin() != wantOrigin {
					t.Fatalf("origin = %#v, want %#v", child.Origin(), wantOrigin)
				}
				wantPath := "root.allOf[0].allOf[0]." + test.operator.String()
				if got := child.Path().String(); got != wantPath {
					t.Fatalf("path = %q, want %q", got, wantPath)
				}
			})

			t.Run("group child disabled", func(t *testing.T) {
				builder := newConstructionBuilder()
				builder.AllOf(func(group *Group[constructionExpression]) {
					test.addGroup(group, false)
				})
				if builder.state.sequence != 2 || len(builder.state.errors) != 0 {
					t.Fatalf(
						"disabled group-child state = sequence %d, %d errors",
						builder.state.sequence,
						len(builder.state.errors),
					)
				}
				outer := requireSingleRootChild[*groupNode](t, builder)
				if len(outer.children) != 0 {
					t.Fatalf("disabled group child count = %d, want 0", len(outer.children))
				}
			})
		})
	}
}

type logicStressCase struct {
	name       string
	logic      Logic
	addBuilder func(*Builder[constructionCondition, constructionExpression], Scope[constructionExpression], bool)
	addGroup   func(*Group[constructionExpression], Scope[constructionExpression], bool)
}

func logicStressCases() []logicStressCase {
	return []logicStressCase{
		{
			name:  "AllOf",
			logic: LogicAllOf,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], scope Scope[constructionExpression], enabled bool) {
				builder.AllOf(scope, enabled)
			},
			addGroup: func(group *Group[constructionExpression], scope Scope[constructionExpression], enabled bool) {
				group.AllOf(scope, enabled)
			},
		},
		{
			name:  "AnyOf",
			logic: LogicAnyOf,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], scope Scope[constructionExpression], enabled bool) {
				builder.AnyOf(scope, enabled)
			},
			addGroup: func(group *Group[constructionExpression], scope Scope[constructionExpression], enabled bool) {
				group.AnyOf(scope, enabled)
			},
		},
		{
			name:  "NoneOf",
			logic: LogicNoneOf,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], scope Scope[constructionExpression], enabled bool) {
				builder.NoneOf(scope, enabled)
			},
			addGroup: func(group *Group[constructionExpression], scope Scope[constructionExpression], enabled bool) {
				group.NoneOf(scope, enabled)
			},
		},
		{
			name:  "NotAllOf",
			logic: LogicNotAllOf,
			addBuilder: func(builder *Builder[constructionCondition, constructionExpression], scope Scope[constructionExpression], enabled bool) {
				builder.NotAllOf(scope, enabled)
			},
			addGroup: func(group *Group[constructionExpression], scope Scope[constructionExpression], enabled bool) {
				group.NotAllOf(scope, enabled)
			},
		},
	}
}

func TestEveryLogicNormalDisabledEmptyNilAndNested(t *testing.T) {
	for _, test := range logicStressCases() {
		t.Run(test.name, func(t *testing.T) {
			t.Run("normal", func(t *testing.T) {
				builder := newConstructionBuilder()
				scopeCalls := 0
				test.addBuilder(builder, func(group *Group[constructionExpression]) {
					scopeCalls++
					group.EQ("field", 1)
				}, true)
				if scopeCalls != 1 || builder.state.sequence != 2 {
					t.Fatalf("normal state = %d scope calls, sequence %d", scopeCalls, builder.state.sequence)
				}
				predicate, err := builder.Predicate()
				if err != nil {
					t.Fatalf("Predicate() error = %v, want nil", err)
				}
				root, _ := predicate.Root().AsGroup()
				nodeView := requireViewChild(t, root, 0, KindGroup)
				group, _ := nodeView.AsGroup()
				if group.Logic() != test.logic || group.ChildCount() != 1 {
					t.Fatalf("group = logic %v, %d children", group.Logic(), group.ChildCount())
				}
			})

			t.Run("disabled", func(t *testing.T) {
				builder := newConstructionBuilder()
				scopeCalls := 0
				test.addBuilder(builder, func(*Group[constructionExpression]) {
					scopeCalls++
				}, false)
				if scopeCalls != 0 ||
					builder.state.sequence != 1 ||
					len(builder.state.root.children) != 0 ||
					len(builder.state.errors) != 0 {
					t.Fatalf(
						"disabled state = %d scope calls, sequence %d, %d children, %d errors",
						scopeCalls,
						builder.state.sequence,
						len(builder.state.root.children),
						len(builder.state.errors),
					)
				}
			})

			t.Run("empty scope", func(t *testing.T) {
				builder := newConstructionBuilder()
				test.addBuilder(builder, func(*Group[constructionExpression]) {}, true)
				group := requireSingleRootChild[*groupNode](t, builder)
				if group.logic != test.logic || len(group.children) != 0 {
					t.Fatalf("empty group = %#v, want empty %v", group, test.logic)
				}
				predicate, err := builder.Predicate()
				if err != nil {
					t.Fatalf("Predicate() error = %v, want nil", err)
				}
				root, _ := predicate.Root().AsGroup()
				want := test.logic == LogicAnyOf || test.logic == LogicNotAllOf
				if !want {
					if root.ChildCount() != 0 {
						t.Fatalf("true identity root child count = %d, want 0", root.ChildCount())
					}
					return
				}
				nodeView := requireViewChild(t, root, 0, KindConstant)
				constant, _ := nodeView.AsConstant()
				if constant.Value() {
					t.Fatal("false empty-group identity normalized to true")
				}
				if nodeView.Origin() != (Origin{Sequence: 1}) {
					t.Fatalf("empty group origin = %#v, want sequence 1", nodeView.Origin())
				}
			})

			t.Run("nil scope", func(t *testing.T) {
				builder := newConstructionBuilder()
				test.addBuilder(builder, nil, true)
				if len(builder.state.root.children) != 0 || len(builder.state.errors) != 1 {
					t.Fatalf(
						"nil scope state = %d children, %d errors",
						len(builder.state.root.children),
						len(builder.state.errors),
					)
				}
				diagnostic := builder.state.errors[0]
				if !errors.Is(diagnostic, ErrInvalidPredicate) ||
					diagnostic.Origin != (Origin{Sequence: 1}) {
					t.Fatalf("nil scope diagnostic = %#v", diagnostic)
				}
			})

			t.Run("nested with disabled gap", func(t *testing.T) {
				builder := newConstructionBuilder()
				disabledCalls := 0
				test.addBuilder(builder, func(outer *Group[constructionExpression]) {
					test.addGroup(outer, func(*Group[constructionExpression]) {
						disabledCalls++
					}, false)
					test.addGroup(outer, func(inner *Group[constructionExpression]) {
						inner.EQ("field", 1)
					}, true)
				}, true)
				if disabledCalls != 0 || builder.state.sequence != 4 {
					t.Fatalf("nested state = %d disabled calls, sequence %d", disabledCalls, builder.state.sequence)
				}
				predicate, err := builder.Predicate()
				if err != nil {
					t.Fatalf("Predicate() error = %v, want nil", err)
				}
				root, _ := predicate.Root().AsGroup()
				outerNode := requireViewChild(t, root, 0, KindGroup)
				outer, _ := outerNode.AsGroup()
				innerNode := requireViewChild(t, outer, 0, KindGroup)
				inner, _ := innerNode.AsGroup()
				leaf := requireViewChild(t, inner, 0, KindComparison)
				if outer.Logic() != test.logic || inner.Logic() != test.logic {
					t.Fatalf("nested logic = (%v, %v), want %v", outer.Logic(), inner.Logic(), test.logic)
				}
				if outerNode.Origin() != (Origin{Sequence: 1}) ||
					innerNode.Origin() != (Origin{Sequence: 3}) ||
					leaf.Origin() != (Origin{Sequence: 4, Operator: OperatorEQ}) {
					t.Fatalf(
						"nested origins = (%#v, %#v, %#v)",
						outerNode.Origin(),
						innerNode.Origin(),
						leaf.Origin(),
					)
				}
				logicName := pathLogicString(test.logic)
				wantPath := fmt.Sprintf(
					"root.allOf[0].%s[0].%s[0].eq",
					logicName,
					logicName,
				)
				if got := leaf.Path().String(); got != wantPath {
					t.Fatalf("nested leaf path = %q, want %q", got, wantPath)
				}
			})

			t.Run("group empty scope", func(t *testing.T) {
				builder := newConstructionBuilder()
				builder.AllOf(func(outer *Group[constructionExpression]) {
					test.addGroup(outer, func(*Group[constructionExpression]) {}, true)
				})
				outer := requireSingleRootChild[*groupNode](t, builder)
				if len(outer.children) != 1 {
					t.Fatalf("outer child count = %d, want 1", len(outer.children))
				}
				inner, ok := outer.children[0].(*groupNode)
				if !ok || inner.logic != test.logic || len(inner.children) != 0 {
					t.Fatalf("nested empty group = %#v, want empty %v", outer.children[0], test.logic)
				}
			})
		})
	}
}

func TestEveryLogicFreezesBeforeScopePanicEscapes(t *testing.T) {
	for _, test := range logicStressCases() {
		t.Run(test.name, func(t *testing.T) {
			builder := newConstructionBuilder()
			var leaked *Group[constructionExpression]
			token := &struct{ logic Logic }{logic: test.logic}
			recovered := recoverValue(func() {
				test.addBuilder(builder, func(group *Group[constructionExpression]) {
					leaked = group
					group.EQ("field", 1)
					panic(token)
				}, true)
			})
			if recovered != token {
				t.Fatalf("recovered value = %#v, want panic token", recovered)
			}
			if leaked == nil || leaked.control.lifecycle != groupFrozen {
				t.Fatalf("leaked group = %#v, want frozen", leaked)
			}

			predicate, err := builder.Predicate()
			if err != nil {
				t.Fatalf("partial Predicate() error = %v, want nil", err)
			}
			root, _ := predicate.Root().AsGroup()
			groupNodeView := requireViewChild(t, root, 0, KindGroup)
			group, _ := groupNodeView.AsGroup()
			if group.Logic() != test.logic || group.ChildCount() != 1 {
				t.Fatalf("partial group = logic %v, %d children", group.Logic(), group.ChildCount())
			}

			leaked.EQ("after panic", 2)
			if len(builder.state.errors) != 1 ||
				!errors.Is(builder.state.errors[0], ErrInvalidState) ||
				builder.state.errors[0].Origin != (Origin{Sequence: 3, Operator: OperatorEQ}) {
				t.Fatalf("post-panic diagnostics = %#v", builder.state.errors)
			}
		})
	}
}

func TestFrozenGroupCallsPreserveInclusionOrderAndNeverMutate(t *testing.T) {
	builder := newConstructionBuilder()
	var leaked *Group[constructionExpression]
	builder.AllOf(func(group *Group[constructionExpression]) {
		leaked = group
		group.EQ("initial", 1)
	})
	outer := requireSingleRootChild[*groupNode](t, builder)

	leaked.EQ("omitted", 2, func(int) bool { return false })
	if builder.state.sequence != 3 || len(builder.state.errors) != 0 {
		t.Fatalf("omitted frozen call = sequence %d, errors %#v", builder.state.sequence, builder.state.errors)
	}
	leaked.EQ("nil inclusion", 3, nil)
	if len(builder.state.errors) != 1 ||
		!errors.Is(builder.state.errors[0], ErrInvalidPredicate) ||
		builder.state.errors[0].Origin != (Origin{Sequence: 4, Operator: OperatorEQ}) {
		t.Fatalf("nil inclusion diagnostics = %#v", builder.state.errors)
	}

	nestedScopeCalls := 0
	calls := []struct {
		name     string
		operator Operator
		call     func()
	}{
		{name: "comparison", operator: OperatorNEQ, call: func() { leaked.NEQ("field", 1) }},
		{name: "membership", operator: OperatorIn, call: func() { leaked.In("field", constructionNumbers{1}) }},
		{name: "range", operator: OperatorBetween, call: func() { leaked.Between("field", 1, 2) }},
		{name: "null", operator: OperatorIsNull, call: func() { leaked.IsNull("field") }},
		{name: "text", operator: OperatorContains, call: func() { leaked.Contains("field", "value") }},
		{
			name: "group",
			call: func() {
				leaked.AnyOf(func(*Group[constructionExpression]) {
					nestedScopeCalls++
				})
			},
		},
		{name: "expression", call: func() { leaked.Expr(constructionExpression{name: "expression"}) }},
	}

	for index, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			beforeChildren := len(outer.children)
			call.call()
			if len(outer.children) != beforeChildren {
				t.Fatalf("frozen group child count changed from %d to %d", beforeChildren, len(outer.children))
			}
			wantSequence := uint64(index + 5)
			if builder.state.sequence != wantSequence {
				t.Fatalf("sequence = %d, want %d", builder.state.sequence, wantSequence)
			}
			if len(builder.state.errors) != index+2 {
				t.Fatalf("error count = %d, want %d", len(builder.state.errors), index+2)
			}
			diagnostic := builder.state.errors[len(builder.state.errors)-1]
			wantOrigin := Origin{Sequence: wantSequence, Operator: call.operator}
			if !errors.Is(diagnostic, ErrInvalidState) || diagnostic.Origin != wantOrigin {
				t.Fatalf("frozen diagnostic = %#v, want origin %#v", diagnostic, wantOrigin)
			}
		})
	}
	if nestedScopeCalls != 0 {
		t.Fatalf("nested frozen scope calls = %d, want 0", nestedScopeCalls)
	}
}

func TestNilInclusionIsReportedForEveryPredicateShape(t *testing.T) {
	tests := []struct {
		name     string
		operator Operator
		add      func(*Builder[constructionCondition, constructionExpression])
	}{
		{name: "comparison", operator: OperatorEQ, add: func(builder *Builder[constructionCondition, constructionExpression]) {
			builder.EQ("field", 1, nil)
		}},
		{name: "membership", operator: OperatorIn, add: func(builder *Builder[constructionCondition, constructionExpression]) {
			builder.In("field", constructionNumbers{1}, nil)
		}},
		{name: "range", operator: OperatorBetween, add: func(builder *Builder[constructionCondition, constructionExpression]) {
			builder.Between("field", 1, 2, nil)
		}},
		{name: "text", operator: OperatorContains, add: func(builder *Builder[constructionCondition, constructionExpression]) {
			builder.Contains("field", "value", nil)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			builder := newConstructionBuilder()
			test.add(builder)
			if len(builder.state.root.children) != 0 || len(builder.state.errors) != 1 {
				t.Fatalf(
					"nil inclusion state = %d children, %d errors",
					len(builder.state.root.children),
					len(builder.state.errors),
				)
			}
			diagnostic := builder.state.errors[0]
			wantOrigin := Origin{Sequence: 1, Operator: test.operator}
			if !errors.Is(diagnostic, ErrInvalidPredicate) ||
				diagnostic.Code != CodeInvalidPredicate ||
				diagnostic.Phase != PhaseConstruct ||
				diagnostic.Origin != wantOrigin {
				t.Fatalf("nil inclusion diagnostic = %#v, want origin %#v", diagnostic, wantOrigin)
			}
		})
	}
}

func TestOriginTimelineSurvivesSnapshotAndLowering(t *testing.T) {
	builder := newConstructionBuilder()
	builder.EQ("root", 1)
	builder.LT("disabled", 2, func(int) bool { return false })
	builder.AnyOf(func(group *Group[constructionExpression]) {
		group.NEQ("nested", 3)
		group.NotNull("disabled", false)
		one := 1
		group.In("nullable", []*int{&one, nil})
	})
	builder.NoneOf(func(group *Group[constructionExpression]) {
		group.Expr(constructionExpression{name: "expression"})
	})
	builder.Native(constructionCondition{"disabled"}, false)
	builder.HasSuffix("tail", "value")

	if builder.state.sequence != 10 {
		t.Fatalf("last sequence = %d, want 10", builder.state.sequence)
	}
	predicate, err := builder.Predicate()
	if err != nil {
		t.Fatalf("Predicate() error = %v, want nil", err)
	}

	observations := collectViewObservations(t, predicate.Root())
	want := []viewObservation{
		{kind: KindGroup, logic: LogicAllOf, path: "root.allOf"},
		{kind: KindComparison, operator: OperatorEQ, origin: Origin{Sequence: 1, Operator: OperatorEQ}, path: "root.allOf[0].eq"},
		{kind: KindGroup, logic: LogicAnyOf, origin: Origin{Sequence: 3}, path: "root.allOf[1].anyOf"},
		{kind: KindComparison, operator: OperatorNEQ, origin: Origin{Sequence: 4, Operator: OperatorNEQ}, path: "root.allOf[1].anyOf[0].neq"},
		{kind: KindGroup, logic: LogicAnyOf, origin: Origin{Sequence: 6, Operator: OperatorIn}, path: "root.allOf[1].anyOf[1].anyOf"},
		{kind: KindMembership, operator: OperatorIn, origin: Origin{Sequence: 6, Operator: OperatorIn}, path: "root.allOf[1].anyOf[1].anyOf[0].in"},
		{kind: KindNull, operator: OperatorIsNull, origin: Origin{Sequence: 6, Operator: OperatorIn}, path: "root.allOf[1].anyOf[1].anyOf[1].is_null"},
		{kind: KindGroup, logic: LogicNoneOf, origin: Origin{Sequence: 7}, path: "root.allOf[2].noneOf"},
		{kind: KindNativeExpression, origin: Origin{Sequence: 8}, path: "root.allOf[2].noneOf[0].native_expression"},
		{kind: KindText, operator: OperatorHasSuffix, origin: Origin{Sequence: 10, Operator: OperatorHasSuffix}, path: "root.allOf[3].has_suffix"},
	}
	if !reflect.DeepEqual(observations, want) {
		t.Fatalf("snapshot observations = %#v, want %#v", observations, want)
	}
}

func TestMultipleConstructionDiagnosticsHaveStableOrder(t *testing.T) {
	builder := newConstructionBuilder()
	builder.EQ((*int)(nil), (*int)(nil))
	builder.Between("range", 3, 2)
	builder.AnyOf(nil)
	builder.In("membership", []any{nil})

	wantCodes := []ErrorCode{
		CodeInvalidField,
		CodeInvalidValue,
		CodeInvalidRange,
		CodeInvalidPredicate,
		CodeInvalidValue,
	}
	wantSequences := []uint64{1, 1, 2, 3, 4}
	var firstFingerprint string
	for attempt := 0; attempt < 3; attempt++ {
		predicate, err := builder.Predicate()
		if err == nil || predicate.Root().Valid() {
			t.Fatalf("attempt %d returned (%#v, %v), want zero Predicate and error", attempt, predicate, err)
		}
		joined, ok := err.(interface{ Unwrap() []error })
		if !ok {
			t.Fatalf("attempt %d error type = %T, want joined error", attempt, err)
		}
		diagnostics := joined.Unwrap()
		if len(diagnostics) != len(wantCodes) {
			t.Fatalf("attempt %d diagnostic count = %d, want %d", attempt, len(diagnostics), len(wantCodes))
		}
		fingerprint := ""
		for index, wrapped := range diagnostics {
			diagnostic, ok := wrapped.(*Error)
			if !ok {
				t.Fatalf("diagnostic %d type = %T, want *Error", index, wrapped)
			}
			if diagnostic.Code != wantCodes[index] || diagnostic.Origin.Sequence != wantSequences[index] {
				t.Fatalf(
					"diagnostic %d = code %v sequence %d, want code %v sequence %d",
					index,
					diagnostic.Code,
					diagnostic.Origin.Sequence,
					wantCodes[index],
					wantSequences[index],
				)
			}
			fingerprint += fmt.Sprintf("%d/%d;", diagnostic.Code, diagnostic.Origin.Sequence)
		}
		if attempt == 0 {
			firstFingerprint = fingerprint
		} else if fingerprint != firstFingerprint {
			t.Fatalf("attempt %d fingerprint = %q, want %q", attempt, fingerprint, firstFingerprint)
		}
	}
}

type viewObservation struct {
	kind     Kind
	logic    Logic
	operator Operator
	origin   Origin
	path     string
}

func collectViewObservations[C, E any](t *testing.T, root NodeView[C, E]) []viewObservation {
	t.Helper()
	stack := []NodeView[C, E]{root}
	var observations []viewObservation
	for len(stack) != 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if !current.Valid() {
			t.Fatal("observation traversal encountered an invalid view")
		}
		observation := viewObservation{
			kind:   current.Kind(),
			origin: current.Origin(),
			path:   current.Path().String(),
		}
		if group, ok := current.AsGroup(); ok {
			observation.logic = group.Logic()
			for index := group.ChildCount() - 1; index >= 0; index-- {
				child, ok := group.Child(index)
				if !ok {
					t.Fatalf("Child(%d) failed within bounds", index)
				}
				stack = append(stack, child)
			}
		} else {
			observation.operator = operatorFromView(t, current)
		}
		observations = append(observations, observation)
	}
	return observations
}

func operatorFromView[C, E any](t *testing.T, nodeView NodeView[C, E]) Operator {
	t.Helper()
	switch nodeView.Kind() {
	case KindConstant, KindNativeCondition, KindNativeExpression:
		return 0
	case KindComparison:
		view, ok := nodeView.AsComparison()
		if !ok {
			t.Fatal("AsComparison() failed for comparison node")
		}
		return view.Operator()
	case KindMembership:
		view, ok := nodeView.AsMembership()
		if !ok {
			t.Fatal("AsMembership() failed for membership node")
		}
		return view.Operator()
	case KindRange:
		view, ok := nodeView.AsRange()
		if !ok {
			t.Fatal("AsRange() failed for range node")
		}
		return view.Operator()
	case KindNull:
		view, ok := nodeView.AsNull()
		if !ok {
			t.Fatal("AsNull() failed for null node")
		}
		return view.Operator()
	case KindText:
		view, ok := nodeView.AsText()
		if !ok {
			t.Fatal("AsText() failed for text node")
		}
		return view.Operator()
	default:
		t.Fatalf("node kind = %v, want leaf kind", nodeView.Kind())
		return 0
	}
}
