# Predicate Semantics

This document collects the backend-neutral behavior that a Weave Compiler must preserve. The [README](../README.md) provides the API overview and construction examples.

## Predicate domain

`Predicate[C, E]` is bound to the Factory and Compiler type domain that created it. A Factory rejects zero Predicates and Predicates created by another Factory, even when their type arguments are identical.

The implicit root is always `LogicAllOf`. An empty root is true. Child order, explicit Group nesting, and origin order remain deterministic through normalization.

## Inclusion is not validation

`when.Predicate[T]`, `when.PairPredicate[A, B]`, and `enabled ...bool` decide whether a construction call exists in the tree. An omitted call still consumes its `Origin.Sequence`, but it does not produce a node and does not undergo field, value, or range validation.

A value predicate sequence runs from left to right, calls each evaluated predicate once, and stops after false. A nil predicate is an `ErrInvalidPredicate` construction error. Predicate panics are caller program errors and are not recovered.

An enabled Group with a nil Scope is invalid. A disabled Group is omitted without invoking its Scope. A Scope's Group freezes before normal return or panic propagation.

## Standard leaf truth table

Standard comparison, membership, range, and literal-text operators describe scalar, single-valued fields.

| Field state | Ordinary value operator | `IsNull` | `NotNull` |
| --- | ---: | ---: | ---: |
| Present, non-null | According to the operator | `false` | `true` |
| Present, explicit null | `false` | `true` | `false` |
| Missing | `false` | `false` | `false` |

“Ordinary value operator” includes `EQ`, `NEQ`, `LT`, `LTE`, `GT`, `GTE`, `In`, `NotIn`, `Between`, `Contains`, `HasPrefix`, and `HasSuffix`.

The table defines a two-valued match set. A backend with SQL null, document missing fields, indexed-value absence, or another truth model must totalize its generated expression to this behavior.

### NEQ is not NOT EQ

`NEQ(field, value)` is false for explicit null and missing fields. `NoneOf(EQ(field, value))` is the Boolean complement of the EQ match set, so it is true for explicit null and missing fields. A Compiler must not rewrite one as the other.

### Nil-like ordinary values

A typed nil ordinary comparison value is invalid and is not rewritten to `IsNull` or `NotNull`. Null intent must be explicit. Field values that are nil-like are also invalid before Compiler field recognition.

## Numeric ranges

`Between(field, lower, upper)` accepts a single static type satisfying `when.Number` and represents a closed interval:

```text
field >= lower AND field <= upper
```

Equal bounds are valid. Enabled inverted bounds return `ErrInvalidRange`; bounds are never exchanged automatically. A float32, float64, or named floating-point NaN bound returns `ErrInvalidValue`.

Use explicit `GTE` with `LTE` or `LT` for non-numeric ranges and for cases where only the backend can define ordering.

## Literal text

`Contains`, `HasPrefix`, and `HasSuffix` accept literal text. Their value is not a SQL LIKE pattern, regular expression, wildcard query, or full-text query. A Compiler must use its backend's parameterization and escaping rules.

Weave does not claim common case sensitivity, Unicode normalization, locale, collation, analyzer, or tokenizer behavior. A Compiler must reject an operator it cannot implement exactly for its configured backend and field.

## Boolean groups

```text
AllOf(A, B)    = A AND B
AnyOf(A, B)    = A OR B
NoneOf(A, B)   = NOT (A OR B)
NotAllOf(A, B) = NOT (A AND B)
```

Empty groups use Boolean identities:

| Logic | Empty value |
| --- | ---: |
| `AllOf` | `true` |
| `AnyOf` | `false` |
| `NoneOf` | `true` |
| `NotAllOf` | `false` |

An enabled Group remains present when every child is omitted. It then becomes the corresponding empty constant. This is different from disabling the Group itself.

Explicit Groups are not flattened, reordered, or deduplicated. Normalization does not apply De Morgan rewrites and does not move Native or Expr payloads.

## Membership normalization

### Empty input

```text
In(field, [])    -> false
NotIn(field, []) -> true
```

The normalized constants no longer require `OperatorIn` or `OperatorNotIn`.

### Nullable In

Nullable lowering applies only when the static slice element type is exactly one pointer level:

```text
In(field, []*T{&a, nil, &b})
    -> AnyOf(In(field, []T{a, b}), IsNull(field))

In(field, []*T{nil, nil})
    -> IsNull(field)

NotIn(field, []*T{&a, &b})
    -> NotIn(field, []T{a, b})
```

`NotIn` with any nil pointer is invalid. A nested pointer element such as `**T` is invalid. A nil-like runtime value inside `[]any` is invalid and never changes the tree shape through runtime-type inference.

Non-null pointer elements are dereferenced when the construction method runs. Value order and duplicates remain intact.

## Constant normalization

Normalization may:

- Replace empty memberships and empty Groups with constants.
- Lower nullable In.
- Remove true identity constants from `AllOf` and `NotAllOf` inputs.
- Remove false identity constants from `AnyOf` and `NoneOf` inputs.
- Fold a Group whose children are all constants.

Normalization does not remove a non-constant child merely because another child determines the Boolean result. This ensures that capabilities and Compiler validation still account for every non-constant node.

## Native and Expr

`Native(C)` may occur only as a direct child of the implicit root and participates in root conjunction. It cannot be nested or negated. `FeatureNativeCondition` commits a Compiler to combining every valid Native value correctly with standard root children.

`Expr(E)` may occur at the root or inside any Group and may participate in all four Boolean logics. `FeatureNativeExpression` applies only to E values that satisfy the Compiler's documented backend preconditions.

Core does not inspect or rewrite either payload. In particular, it does not prove that E is a Boolean expression. Native and Expr behavior inside their payloads is the caller's responsibility.

## Requirements after normalization

Every remaining standard leaf contributes its Operator. Native and Expr contribute their Features. Constants and Groups do not add optional requirements because constants and all four Boolean logics are mandatory Compiler behavior.

Requirements are immutable value snapshots. They answer whether a Compiler lacks a global ability, not whether a field is recognized, a value has the right type, or an operator applies to a particular field.

## Snapshot and borrowed payloads

Core owns and freezes tree topology, child storage, paths, membership storage, top-level byte-slice comparison values, and top-level Native slice storage. Views expose collections through count and index accessors.

Fields, nested references, pointer and map values, objects referenced by membership elements, nested Native references, and all Expr values remain borrowed. A Predicate is safe for concurrent reads only while callers keep those borrowed values immutable.

## Depth and deterministic diagnostics

The implicit root has depth zero. Each child edge adds one, and `MaxPredicateDepth` is 128. Construction and defensive compile validation reject depth 129 with `ErrDepthLimit`.

Normalized `NodePath` values describe current tree position. `Origin` identifies the original construction call, including when one input lowers to multiple normalized nodes. Path strings and enum strings are deterministic diagnostic text, not serialization protocols.
