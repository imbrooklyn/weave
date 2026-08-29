// Package compilertest provides backend-neutral semantic fixtures for Weave
// Compiler implementations.
//
// Adapter tests bind their own typed fields and execution callback to Harness.
// Run executes the capability-aware testing contract, while Scenarios exposes
// the applicable canonical semantic cases to ordinary programs such as
// runnable demos. Both paths compare stable record-ID match sets; neither
// interprets or compares a Compiler's textual or structural backend output.
// Run also verifies structured Factory-preflight rejection for undeclared
// operators and native features. An Adapter may use Harness.InspectCondition
// for its own representation-specific safety assertions.
package compilertest
