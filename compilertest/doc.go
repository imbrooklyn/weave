// Package compilertest provides backend-neutral semantic fixtures for Weave
// Compiler implementations.
//
// Adapter tests bind their own typed fields and execution callback to Harness.
// Run executes the complete testing contract, while Scenarios exposes the same
// canonical semantic cases to ordinary programs such as runnable demos. Both
// paths compare stable record-ID match sets; neither interprets or compares a
// Compiler's textual or structural backend output. An Adapter may use
// Harness.InspectCondition for its own representation-specific safety
// assertions.
package compilertest
