# Weave

[![CI](https://github.com/imbrooklyn/weave/actions/workflows/ci.yml/badge.svg)](https://github.com/imbrooklyn/weave/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Weave is a backend-neutral query predicate construction and compilation core for Go. It builds an adapter-bound, structurally immutable predicate tree from typed fluent calls, then delegates field validation and backend expression generation to a caller-supplied `Compiler`.

Weave does not execute queries and does not own database handles, sessions, contexts, loggers, or transactions. Its runtime packages depend only on the Go standard library.

## Requirements

- Go 1.27 or newer. The fluent API uses concrete generic methods introduced in Go 1.27.
- A `Compiler[C, E]` implementation for the condition type `C` and native expression carrier `E` used by the calling application.

The module and package paths are:

```text
github.com/imbrooklyn/weave
github.com/imbrooklyn/weave/when
github.com/imbrooklyn/weave/compilertest
```

The `compilertest` package provides a shared scalar fixture and a
Factory/field/capability/execution callback harness for Adapter semantic tests.
It checks every standard operator, all four Boolean logics, null and missing
states, Native and Expr, redacted validation errors, and repeated and
concurrent compilation by comparing stable record-ID match sets. It never
interprets backend output. Harnesses may add their own condition inspection
callback for representation-specific parameterization or safety assertions.

## Core API

| API | Responsibility |
| --- | --- |
| `Compiler[C, E]` | Validates an adapter-bound predicate and emits a backend condition of type `C`. |
| `Factory[C, E]` | Binds one Compiler, snapshots its capabilities, creates Builders, and performs compile preflight. |
| `Builder[C, E]` | Provides the mutable, fluent root construction API. Its root is an implicit `AllOf`. |
| `Group[E]` / `Scope[E]` | Builds explicit nested Boolean groups during a callback. |
| `Predicate[C, E]` | Holds a normalized, structurally immutable snapshot. |
| `NodeView[C, E]` | Exposes the sealed read-only tree to Compiler implementations. |
| `Capabilities` | Describes a configured Compiler's stable operator and feature commitment. |
| `Requirements` | Describes the operators and features used by a normalized Predicate. |
| `Error` | Provides classification, phase, normalized path, and construction origin without formatting query values. |

`C` and `E` bind a Predicate to one Compiler type domain. `C` is the final compiled condition. `E` is the opaque native expression carrier accepted by `Expr`. They may be the same Go type, but their positions and responsibilities remain different.

### Standard operators

| Family | Builder and Group methods |
| --- | --- |
| Comparison | `EQ`, `NEQ`, `LT`, `LTE`, `GT`, `GTE` |
| Membership | `In`, `NotIn` |
| Numeric range | `Between` |
| Null | `IsNull`, `NotNull` |
| Literal text | `Contains`, `HasPrefix`, `HasSuffix` |
| Boolean group | `AllOf`, `AnyOf`, `NoneOf`, `NotAllOf` |
| Native | `Expr`; root Builder also provides `Native` |

`Between` accepts `when.Number` values and represents the closed range `field >= lower AND field <= upper`. Enabled inverted ranges return `ErrInvalidRange`; a floating-point NaN bound returns `ErrInvalidValue`. Non-numeric ranges can be written explicitly with `GTE` and `LTE` or `LT`.

Text values are literal text, not backend patterns. A Compiler must parameterize or escape them for its backend and must not silently substitute regex, wildcard, full-text, or collation semantics.

## Constructing a Predicate

A Factory is supplied with a Compiler during application assembly. Fields are adapter-specific values; Weave deliberately accepts them as `any`, and the Compiler remains responsible for recognizing a field and validating its value type.

```go
package filters

import (
    "github.com/imbrooklyn/weave"
    "github.com/imbrooklyn/weave/when"
)

type UserFields struct {
    TenantID any
    Status   any
    Name     any
    Email    any
}

type UserInput struct {
    TenantID int64
    Statuses []int
    Keyword  string
}

func UserPredicate[C, E any](
    factory *weave.Factory[C, E],
    fields UserFields,
    input UserInput,
) (weave.Predicate[C, E], error) {
    return factory.New().
        EQ(fields.TenantID, input.TenantID).
        In(fields.Status, input.Statuses, when.NotEmpty[[]int]).
        AnyOf(func(group *weave.Group[E]) {
            group.Contains(fields.Name, input.Keyword).
                Contains(fields.Email, input.Keyword)
        }, when.NotBlank(input.Keyword)).
        Predicate()
}
```

Compile a completed Predicate through the Factory:

```go
condition, err := factory.Compile(predicate)
```

Or use `Build` to snapshot and compile the current Builder in one call:

```go
condition, err := factory.New().EQ(field, value).Build()
```

The package examples include a small inspection Compiler so that the complete lifecycle is runnable without a query backend:

```sh
go test -run '^Example' ./...
```

## Inclusion with `when`

`when.Predicate[T]` and `when.PairPredicate[A, B]` decide whether a method call contributes a node. They are inclusion rules, not validation rules:

- No predicate means the node is enabled.
- Predicates run from left to right and stop at the first false result.
- Every predicate that runs is called exactly once.
- False omits the node; it does not add a Boolean false node.
- A nil predicate is invalid.
- An omitted node does not undergo its field, value, or range validation.

Methods with `enabled ...bool` are enabled when no values are supplied or when every supplied value is true. A disabled Group is omitted and its Scope is not called.

The `when` package provides `NotZero`, `Positive`, `NonNegative`, `NotBlank`, `NotEmpty`, `NotNil`, `NotZeroTime`, `ValidRange`, `True`, `False`, `All`, `Any`, `Not`, `If`, and `PairIf`.

## Boolean groups

The Builder root is always an implicit `AllOf`. Explicit groups preserve their nesting and child order:

```text
AllOf(A, B)    = A AND B
AnyOf(A, B)    = A OR B
NoneOf(A, B)   = NOT (A OR B)
NotAllOf(A, B) = NOT (A AND B)
```

Enabled empty groups use Boolean identities:

| Group | Empty result |
| --- | ---: |
| `AllOf` | `true` |
| `AnyOf` | `false` |
| `NoneOf` | `true` |
| `NotAllOf` | `false` |

An enabled Group whose children are all omitted is still an enabled empty Group. This differs from disabling the Group itself. In particular, an optional `AnyOf` should normally be disabled at the Group level when its shared input is absent.

A Group is valid only while its Scope callback runs. It freezes before the callback returns or a panic unwinds through it. A later method call records `ErrInvalidState` and cannot change the tree.

## Two-valued match-set semantics

A Predicate determines whether a record belongs to a result set. Standard operators therefore have a two-valued `true` or `false` result even if the backend internally uses another truth model.

| Field state | Comparison, membership, range, text | `IsNull` | `NotNull` |
| --- | ---: | ---: | ---: |
| Present, non-null value | According to the operator | `false` | `true` |
| Present, explicit null | `false` | `true` | `false` |
| Missing | `false` | `false` | `false` |

`NEQ(field, value)` matches only present, non-null unequal values. It is not equivalent to `NoneOf(EQ(field, value))`: the latter is the complement of the EQ match set and therefore includes explicit-null and missing records.

Standard operators describe scalar, single-valued fields. Backend-specific array, multi-valued, nested, or existence behavior belongs in an adapter-native `Expr` unless a separate standard operator defines it.

See [Predicate semantics](docs/semantics.md) for the normalization and truth rules in one place.

## Empty and nullable membership

Normalization gives collection edge cases deterministic meanings:

```text
In(field, empty)    = false
NotIn(field, empty) = true
```

For a one-level pointer slice, `In(field, []*T{&a, nil, &b})` becomes `AnyOf(In(field, []T{a, b}), IsNull(field))`. An all-nil input becomes `IsNull`. `NotIn` rejects any nil pointer element instead of emitting a backend `NOT IN (..., NULL)` expression. Nested pointer element types and nil-like values inside `[]any` are invalid.

Input order and duplicate values are preserved. If an empty collection should mean “do not filter,” omit the node explicitly with `when.NotEmpty`.

## Capabilities and Requirements

`Compiler.Capabilities` is called once by `NewFactory`; the Factory stores an immutable value snapshot. A Compiler's capability commitment must remain stable for its lifetime.

`Predicate.Requirements` is calculated after normalization. For example:

- Empty `In` and `NotIn` become constants and no longer require their membership operators.
- Nullable `In` with non-null values requires both `OperatorIn` and `OperatorIsNull`.
- An all-nil `In` requires only `OperatorIsNull`.
- `Native` requires `FeatureNativeCondition`.
- `Expr` requires `FeatureNativeExpression`.

`Factory.Compile` validates the Predicate and locates the first missing capability in stable tree order before it calls the Compiler. Capabilities do not prove that a particular field or value is valid. A Compiler may implement `FieldCapabilityResolver` for optional field-level discovery, but compile-time validation remains authoritative.

## Native and Expr

`Native(C)` adds an already-formed final condition as a direct child of the implicit root. It cannot be nested or negated. If `C` is a slice, core shallow-clones its top-level backing array. Supporting it requires `FeatureNativeCondition` and includes correct conjunction with other root children.

`Expr(E)` places an adapter-native expression at the root or inside any Boolean Group. Core treats `E` as opaque: it does not classify, clone, validate, rewrite, or prove that a value is a Boolean filter. Supporting it requires `FeatureNativeExpression`.

The caller and Compiler are responsible for the backend validity, Boolean meaning, parameterization, escaping, dialect or server compatibility, and mutable state of Native and Expr payloads. Do not place untrusted raw query text in either escape hatch.

## Snapshot and ownership

`Predicate` snapshots have immutable topology, stable child order, and private core-owned backing storage. Later Builder calls do not modify earlier snapshots. Repeated snapshots of an unchanged Builder are structurally independent and deterministic.

Core takes ownership only where its API explicitly says so:

- Membership input is shallow-cloned when the Builder or Group method is called.
- A top-level `[]byte` comparison value is cloned.
- A top-level slice passed to `Native` is shallow-cloned.
- Node children and path segments are never exposed as mutable slices.

The following remain borrowed:

- Field payloads and their nested references.
- Pointer, map, function, channel, and nested-reference comparison values.
- Objects referenced by membership elements.
- Nested references inside a Native condition.
- The complete Expr payload, including a top-level slice.

Do not mutate borrowed payloads while a Predicate may be read or compiled, especially across goroutines. Weave does not perform general reflection-based deep copies.

## Concurrency

| Type | Concurrency contract |
| --- | --- |
| `Factory` | `New`, `Capabilities`, and `Compile` may be called concurrently. |
| `Compiler` | Must be request-stateless and safe for concurrent `Compile` and `Capabilities` calls. |
| `Builder` | Mutable and not safe for concurrent use. Use one per request or construction flow. |
| `Group` | Mutable, callback-scoped, and not safe for concurrent use. |
| `Predicate` and node views | Safe for concurrent reads when borrowed payloads are not mutated. |
| `OperatorSet` and `FeatureSet` | Immutable value types and safe to copy. |

The maximum legal node depth is `MaxPredicateDepth` (128), with the implicit root at depth zero. Exceeding it returns `ErrDepthLimit` rather than recursing without a bound.

## Errors and diagnostics

Use `errors.Is` for classifications and `errors.As` for `*weave.Error` details:

```go
if errors.Is(err, weave.ErrUnsupportedOperator) {
    // The configured Compiler does not implement the required operator.
}

var detail *weave.Error
if errors.As(err, &detail) {
    log.Printf("phase=%s path=%s", detail.Phase, detail.Path)
}
```

Construction and normalization errors do not match `ErrCompile`. Preflight, Compiler validation, and emission errors match both `ErrCompile` and their specific classification. A compile failure always returns the zero value of `C`.

The default `Error` string may include classifications, a normalized node path, operator or feature identifiers, and Go types. It does not format field values, query values, Native or Expr payloads, `Origin`, or the text of its lower-level `Cause`. A caller that explicitly unwraps `Cause` enters the underlying Compiler's diagnostic boundary.

## Writing a Compiler

Compiler implementations consume the sealed tree returned by `Predicate.Root`. They must preserve Boolean structure, validate all supported node payloads, return explicit unsupported or applicability errors, and never silently drop or approximate a node. See [Writing a Compiler](docs/compiler.md).

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for local verification commands and contribution expectations. Security issues should follow [SECURITY.md](SECURITY.md). User-visible changes are recorded in [CHANGELOG.md](CHANGELOG.md).

## License

Weave is licensed under the [Apache License 2.0](LICENSE).
