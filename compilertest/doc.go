// Package compilertest provides backend-neutral semantic fixtures for Weave
// Compiler implementations.
//
// Adapter tests bind their own typed fields and execution callback to Harness.
// The suite compares stable record-ID match sets; it never interprets or
// compares a Compiler's textual or structural backend output. An Adapter may
// use Harness.InspectCondition for its own representation-specific safety
// assertions.
package compilertest
