package weave

import (
	"fmt"
	"sync/atomic"
	"testing"
)

type benchmarkCondition int
type benchmarkExpression struct{ id int }

type benchmarkCompiler struct{}

func (benchmarkCompiler) Compile(
	predicate Predicate[benchmarkCondition, benchmarkExpression],
) (benchmarkCondition, error) {
	root := predicate.Root()
	if !root.Valid() {
		return 0, ErrInvalidPredicate
	}

	count := 0
	stack := []NodeView[benchmarkCondition, benchmarkExpression]{root}
	for len(stack) != 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		count += int(current.Kind())
		group, ok := current.AsGroup()
		if !ok {
			continue
		}
		for index := group.ChildCount() - 1; index >= 0; index-- {
			child, childOK := group.Child(index)
			if !childOK {
				return 0, ErrInvalidPredicate
			}
			stack = append(stack, child)
		}
	}
	return benchmarkCondition(count), nil
}

func (benchmarkCompiler) Capabilities() Capabilities {
	return benchmarkCapabilities
}

var benchmarkCapabilities = Capabilities{
	Operators: NewOperatorSet(
		OperatorEQ,
		OperatorNEQ,
		OperatorLT,
		OperatorLTE,
		OperatorGT,
		OperatorGTE,
		OperatorIn,
		OperatorNotIn,
		OperatorBetween,
		OperatorIsNull,
		OperatorNotNull,
		OperatorContains,
		OperatorHasPrefix,
		OperatorHasSuffix,
	),
	Features: NewFeatureSet(
		FeatureNativeCondition,
		FeatureNativeExpression,
	),
}

type benchmarkCase struct {
	name string
	add  func(*Builder[benchmarkCondition, benchmarkExpression])
}

var benchmarkCases = []benchmarkCase{
	{name: "5_leaves", add: func(builder *Builder[benchmarkCondition, benchmarkExpression]) {
		addBenchmarkLeaves(builder, 5)
	}},
	{name: "20_leaves", add: func(builder *Builder[benchmarkCondition, benchmarkExpression]) {
		addBenchmarkLeaves(builder, 20)
	}},
	{name: "3_level_group", add: addBenchmarkThreeLevelGroup},
	{name: "in_100", add: func(builder *Builder[benchmarkCondition, benchmarkExpression]) {
		builder.In("field", benchmarkInts(100))
	}},
	{name: "in_1000", add: func(builder *Builder[benchmarkCondition, benchmarkExpression]) {
		builder.In("field", benchmarkInts(1000))
	}},
	{name: "nullable_in_100", add: func(builder *Builder[benchmarkCondition, benchmarkExpression]) {
		builder.In("field", benchmarkNullableInts(100))
	}},
}

var (
	benchmarkConditionSink           benchmarkCondition
	benchmarkConcurrentConditionSink atomic.Int64
	benchmarkPredicateSink           Predicate[benchmarkCondition, benchmarkExpression]
)

func BenchmarkPredicateSnapshot(b *testing.B) {
	for _, test := range benchmarkCases {
		test := test
		b.Run(test.name, func(b *testing.B) {
			factory := NewFactory[benchmarkCondition, benchmarkExpression](benchmarkCompiler{})
			builder := factory.New()
			test.add(builder)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				predicate, err := builder.Predicate()
				if err != nil {
					b.Fatalf("Predicate: %v", err)
				}
				benchmarkPredicateSink = predicate
			}
		})
	}
}

func BenchmarkFactoryCompile(b *testing.B) {
	for _, test := range benchmarkCases {
		test := test
		b.Run("repeated_predicate/"+test.name, func(b *testing.B) {
			factory := NewFactory[benchmarkCondition, benchmarkExpression](benchmarkCompiler{})
			builder := factory.New()
			test.add(builder)
			predicate, err := builder.Predicate()
			if err != nil {
				b.Fatalf("Predicate: %v", err)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				condition, compileErr := factory.Compile(predicate)
				if compileErr != nil {
					b.Fatalf("Compile: %v", compileErr)
				}
				benchmarkConditionSink = condition
			}
		})
	}
}

func BenchmarkFactoryCompileConcurrent(b *testing.B) {
	factory := NewFactory[benchmarkCondition, benchmarkExpression](benchmarkCompiler{})
	builder := factory.New()
	addBenchmarkLeaves(builder, 20)
	addBenchmarkThreeLevelGroup(builder)
	predicate, err := builder.Predicate()
	if err != nil {
		b.Fatalf("Predicate: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(parallel *testing.PB) {
		var lastCondition benchmarkCondition
		for parallel.Next() {
			condition, compileErr := factory.Compile(predicate)
			if compileErr != nil {
				b.Errorf("Compile: %v", compileErr)
				return
			}
			lastCondition = condition
		}
		benchmarkConcurrentConditionSink.Add(int64(lastCondition))
	})
}

func addBenchmarkLeaves(
	builder *Builder[benchmarkCondition, benchmarkExpression],
	count int,
) {
	for index := 0; index < count; index++ {
		field := fmt.Sprintf("field_%d", index)
		switch index % 14 {
		case 0:
			builder.EQ(field, index)
		case 1:
			builder.NEQ(field, index)
		case 2:
			builder.LT(field, index)
		case 3:
			builder.LTE(field, index)
		case 4:
			builder.GT(field, index)
		case 5:
			builder.GTE(field, index)
		case 6:
			builder.In(field, []int{index, index + 1})
		case 7:
			builder.NotIn(field, []int{index, index + 1})
		case 8:
			builder.Between(field, index, index+1)
		case 9:
			builder.IsNull(field)
		case 10:
			builder.NotNull(field)
		case 11:
			builder.Contains(field, "literal")
		case 12:
			builder.HasPrefix(field, "literal")
		case 13:
			builder.HasSuffix(field, "literal")
		}
	}
}

func addBenchmarkThreeLevelGroup(
	builder *Builder[benchmarkCondition, benchmarkExpression],
) {
	builder.AllOf(func(first *Group[benchmarkExpression]) {
		first.AnyOf(func(second *Group[benchmarkExpression]) {
			second.NoneOf(func(third *Group[benchmarkExpression]) {
				third.EQ("field", 1).NEQ("other", 2)
			})
		})
	})
}

func benchmarkInts(count int) []int {
	values := make([]int, count)
	for index := range values {
		values[index] = index
	}
	return values
}

func benchmarkNullableInts(count int) []*int {
	allocated := benchmarkInts(count)
	values := make([]*int, count)
	for index := range values {
		if index%10 != 0 {
			values[index] = &allocated[index]
		}
	}
	return values
}
