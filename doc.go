// Package weave constructs backend-neutral, adapter-bound query predicates.
// It defines standard query operators, explicit Boolean groups, structurally
// immutable predicate snapshots, a sealed read-only AST, capability preflight,
// and structured errors. It does not execute queries or hold database handles,
// sessions, contexts, loggers, or transactions.
//
// A [Factory] binds one request-stateless [Compiler]. Each request obtains a
// fresh mutable [Builder], constructs a predicate, and either snapshots it with
// [Builder.Predicate] or compiles it with [Builder.Build]. Builders and Groups
// are not safe for concurrent use. Factories, conforming Compilers, Predicates,
// and node views can be reused concurrently subject to the borrowed-payload
// rules below.
//
// # Match-set semantics
//
// Predicates have two-valued match-set semantics: a record either matches or
// does not. Standard comparison, membership, range, and literal-text operators
// do not match explicit null or missing fields. [OperatorIsNull] matches only a
// present field whose value is explicitly null; [OperatorNotNull] matches only
// a present, non-null field. Consequently, NEQ is not the Boolean complement of
// EQ. A Compiler must preserve these rules when its backend has different null
// or missing-value behavior.
//
// The root is an implicit [LogicAllOf]. Explicit [LogicAllOf], [LogicAnyOf],
// [LogicNoneOf], and [LogicNotAllOf] groups may nest up to
// [MaxPredicateDepth]. Empty groups use their Boolean identities. Empty In is
// false, empty NotIn is true, and an In over one-level pointers lowers nil
// elements to an IsNull alternative. Text operators always treat their input
// as literal text, not as a backend pattern.
//
// # Capabilities and native values
//
// [Capabilities] is a stable Compiler-instance commitment. A normalized
// [Predicate] exposes its [Requirements], and [Factory.Compile] rejects missing
// capabilities before invoking the Compiler. Field recognition, value
// compatibility, and per-field operator applicability remain Compiler
// responsibilities.
//
// Native(C) is restricted to direct children of the root. Expr(E) is an opaque
// escape hatch that may appear anywhere in the Boolean tree. The Compiler and
// caller define which C and E values are valid and safe for the backend.
//
// # Ownership
//
// Predicate topology and core-owned backing storage are immutable. Core clones
// membership slices, top-level byte-slice comparison values, and top-level
// slice Native values as documented by their accessors. Fields, nested
// references, pointer targets, map contents, slice elements that contain
// references, and all Expr payloads remain borrowed. Callers must not mutate
// borrowed payloads while a Predicate can be read or compiled.
package weave
