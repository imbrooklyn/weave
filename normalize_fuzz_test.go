package weave

import (
	"fmt"
	"testing"
)

func FuzzNormalizationIdempotenceAndRequirements(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 1, 2, 3, 4})
	f.Add([]byte{7, 8, 19, 31, 43, 55})
	f.Add([]byte{255, 128, 64, 32, 16, 8, 4, 2, 1})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 48 {
			data = data[:48]
		}
		builder := newConstructionBuilder()
		for index, raw := range data {
			addNormalizationFuzzCall(builder, index, raw)
		}

		predicate, err := builder.Predicate()
		if err != nil {
			t.Fatalf("Predicate() error = %v, want nil", err)
		}
		before, ok := predicateViewFingerprint(t, predicate.Root())
		if !ok {
			t.Fatal("normalized Predicate contains an invalid view")
		}
		wantRequirements := requirementsFromNodeViews(t, predicate.Root())
		assertRequirementsEqual(t, predicate.Requirements(), wantRequirements)
		repeated, repeatedError := builder.Predicate()
		if repeatedError != nil {
			t.Fatalf("repeated Predicate() error = %v, want nil", repeatedError)
		}
		repeatedFingerprint, ok := predicateViewFingerprint(t, repeated.Root())
		if !ok || repeatedFingerprint != before {
			t.Fatalf("repeated normalization is not deterministic: first %q, repeated %q", before, repeatedFingerprint)
		}
		if repeated.state == predicate.state || repeated.state.root == predicate.state.root {
			t.Fatal("repeated Predicate() reused snapshot topology")
		}
		assertRequirementsEqual(t, repeated.Requirements(), wantRequirements)

		renormalizedState, normalizeError := normalizePredicateState[constructionCondition, constructionExpression](
			predicate.state,
			predicate.state.domain,
		)
		if normalizeError != nil {
			t.Fatalf("second normalization error = %v", normalizeError)
		}
		renormalized := Predicate[constructionCondition, constructionExpression]{
			state: renormalizedState,
		}
		after, ok := predicateViewFingerprint(t, renormalized.Root())
		if !ok {
			t.Fatal("second normalization contains an invalid view")
		}
		if after != before {
			t.Fatalf("normalization is not idempotent:\nfirst:  %s\nsecond: %s", before, after)
		}
		if renormalized.state == predicate.state ||
			renormalized.state.root == predicate.state.root {
			t.Fatal("second normalization reused source topology")
		}

		sourceAfter, ok := predicateViewFingerprint(t, predicate.Root())
		if !ok || sourceAfter != before {
			t.Fatalf("second normalization modified its source: before %q, after %q", before, sourceAfter)
		}
		assertRequirementsEqual(t, renormalized.Requirements(), wantRequirements)
		assertRequirementsEqual(
			t,
			renormalized.Requirements(),
			requirementsFromNodeViews(t, renormalized.Root()),
		)
	})
}

func addNormalizationFuzzCall(
	builder *Builder[constructionCondition, constructionExpression],
	index int,
	raw byte,
) {
	value := int(raw)
	switch raw % 12 {
	case 0:
		builder.In("field", []int{})
	case 1:
		builder.NotIn("field", []int{})
	case 2:
		builder.In("field", []int{value, value, value + 1})
	case 3:
		first, second := value, value+1
		builder.In("field", []*int{&second, nil, &first, &second})
	case 4:
		builder.In("field", []*int{nil, nil})
	case 5:
		builder.EQ("field", value)
	case 6:
		builder.NEQ("field", value)
	case 7:
		addEmptyFuzzGroup(builder, Logic(raw/12)%4+1)
	case 8:
		logic := Logic(raw/12)%4 + 1
		addGroupByLogic(builder, logic, func(group *Group[constructionExpression]) {
			if logic == LogicAllOf || logic == LogicNotAllOf {
				group.In("field", []int{})
			} else {
				group.NotIn("field", []int{})
			}
			group.EQ("field", value)
		})
	case 9:
		builder.Native(constructionCondition{fmt.Sprintf("native-%d", index)})
	case 10:
		builder.Expr(constructionExpression{name: fmt.Sprintf("expression-%d", index)})
	case 11:
		builder.NoneOf(func(group *Group[constructionExpression]) {
			group.EQ("field", value)
		})
	}
}

func addEmptyFuzzGroup(
	builder *Builder[constructionCondition, constructionExpression],
	logic Logic,
) {
	addGroupByLogic(builder, logic, func(*Group[constructionExpression]) {})
}
