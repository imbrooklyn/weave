package weave

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

type concurrentFactoryCompiler struct {
	capabilitiesCalls atomic.Int64
	compileCalls      atomic.Int64
}

func (c *concurrentFactoryCompiler) Compile(
	predicate Predicate[string, string],
) (string, error) {
	c.compileCalls.Add(1)
	root, ok := predicate.Root().AsGroup()
	if !ok {
		return "", ErrInvalidPredicate
	}
	return fmt.Sprintf("children:%d", root.ChildCount()), nil
}

func (c *concurrentFactoryCompiler) Capabilities() Capabilities {
	c.capabilitiesCalls.Add(1)
	return Capabilities{Operators: NewOperatorSet(OperatorEQ)}
}

func TestFactoryAndPredicateSupportConcurrentReuse(t *testing.T) {
	compiler := &concurrentFactoryCompiler{}
	factory := NewFactory[string, string](compiler)
	predicate, err := factory.New().EQ("shared", 1).Predicate()
	if err != nil {
		t.Fatalf("Predicate failed: %v", err)
	}

	const workers = 64
	start := make(chan struct{})
	errorsFound := make(chan string, workers)
	var waitGroup sync.WaitGroup
	waitGroup.Add(workers)
	for worker := 0; worker < workers; worker++ {
		worker := worker
		go func() {
			defer waitGroup.Done()
			<-start

			compiled, compileErr := factory.Compile(predicate)
			if compileErr != nil || compiled != "children:1" {
				errorsFound <- fmt.Sprintf(
					"worker %d shared Compile = (%q, %v)",
					worker,
					compiled,
					compileErr,
				)
				return
			}
			if capabilities := factory.Capabilities(); !capabilities.Operators.Has(OperatorEQ) {
				errorsFound <- fmt.Sprintf(
					"worker %d observed missing EQ capability",
					worker,
				)
				return
			}

			built, buildErr := factory.New().EQ("fresh", worker).Build()
			if buildErr != nil || built != "children:1" {
				errorsFound <- fmt.Sprintf(
					"worker %d Build = (%q, %v)",
					worker,
					built,
					buildErr,
				)
			}
		}()
	}
	close(start)
	waitGroup.Wait()
	close(errorsFound)
	for failure := range errorsFound {
		t.Error(failure)
	}

	if calls := compiler.capabilitiesCalls.Load(); calls != 1 {
		t.Fatalf("Capabilities calls = %d, want 1", calls)
	}
	if calls := compiler.compileCalls.Load(); calls != workers*2 {
		t.Fatalf("Compile calls = %d, want %d", calls, workers*2)
	}
}
