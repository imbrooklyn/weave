# Writing a Compiler

A Weave `Compiler[C, E]` validates a normalized, adapter-bound Predicate and emits a final backend condition of type `C`. This document describes the public contract implemented by the current core API.

## Interface

```go
type Compiler[C, E any] interface {
    Compile(Predicate[C, E]) (C, error)
    Capabilities() Capabilities
}
```

`C` is the final condition consumed by the caller's backend API. `E` is the carrier accepted by `Expr`. The types may be identical, but `Native(C)` remains root-only while `Expr(E)` is nestable.

## Lifecycle and concurrency

A Compiler must be request-stateless and safe for concurrent calls to `Compile` and `Capabilities`. It may hold immutable semantic configuration and read-only field metadata. It must not hold a database handle, session, context, logger, transaction, or per-request query values.

`Capabilities` must return the same value throughout the Compiler's lifetime. `NewFactory` calls it once and stores a snapshot. Compiler constructors should reject invalid semantic configuration before a Factory is made.

## Factory safeguards

The normal entry point is `Factory.Compile`, which performs:

1. Factory and Predicate identity checks.
2. Structural, ownership, path, Native-position, depth, and Requirements validation.
3. Capability preflight in stable pre-order depth-first tree order.
4. Delegation to `Compiler.Compile`.
5. Compile error normalization and zero-`C` enforcement.

A direct call to `Compiler.Compile` bypasses those safeguards. Implementations must still validate their adapter-specific field, value, operator applicability, Native, and Expr contracts defensively.

## Traversing the sealed tree

Start with `predicate.Root()`. A valid root is always an implicit `LogicAllOf` `GroupView`. Use `NodeView.Kind` or the `AsX` methods to select a sealed view:

```go
root := predicate.Root()
group, ok := root.AsGroup()
if !ok || group.Logic() != weave.LogicAllOf {
    return zero, &weave.Error{
        Code:  weave.CodeInvalidPredicate,
        Phase: weave.PhaseValidate,
    }
}

for index := 0; index < group.ChildCount(); index++ {
    child, ok := group.Child(index)
    if !ok {
        // Return a structured invalid-predicate error.
    }
    // Validate and emit child according to its sealed view.
}
```

Collection views use count and index accessors. Do not retain a NodeView to mutate payloads, and do not use reflection to reconstruct private node types.

## Mandatory Boolean behavior

Constants and all four Group logics are mandatory Compiler behavior and are not optional capabilities:

```text
AllOf(A, B)    = A AND B
AnyOf(A, B)    = A OR B
NoneOf(A, B)   = NOT (A OR B)
NotAllOf(A, B) = NOT (A AND B)
```

The core normalizes empty Groups, but a Compiler should still treat malformed input defensively and must preserve explicit non-empty nesting and precedence.

The standard leaf truth table, nullable membership rules, and literal-text contract are defined in [Predicate semantics](semantics.md). A Compiler must account for backend null, missing, unknown, array, analyzer, or pattern behavior rather than relying on superficially similar operators.

## Capabilities

List only operators and features the configured Compiler can implement exactly for their valid field domain:

```go
func (c Compiler) Capabilities() weave.Capabilities {
    return weave.Capabilities{
        Operators: weave.NewOperatorSet(
            weave.OperatorEQ,
            weave.OperatorIn,
            weave.OperatorIsNull,
        ),
        Features: weave.NewFeatureSet(
            weave.FeatureNativeExpression,
        ),
    }
}
```

Do not expose a capability that expands automatically when core adds an Operator. A new unknown Operator must be rejected explicitly.

Global capabilities do not mean that every operator applies to every field. Return `ErrOperatorNotApplicable` when a recognized field cannot use an otherwise supported operator. A Compiler may implement `FieldCapabilityResolver` for discovery, but `Compile` remains authoritative.

## Field and value validation

Core deliberately stores fields as `any`. A Compiler must distinguish:

| Condition | Classification |
| --- | --- |
| Field is unrecognized or belongs to another adapter | `ErrInvalidField` |
| Field is valid but the operator does not apply | `ErrOperatorNotApplicable` |
| Value type, nil state, precision, representation, or recognized native value is invalid | `ErrInvalidValue` |
| Compiler does not implement the operator globally | `ErrUnsupportedOperator` |
| Compiler does not implement Native or Expr globally | `ErrUnsupportedFeature` |

Do not stringify arbitrary values, perform lossy conversion, trust unvalidated field names, or silently omit an invalid node. Standard fields describe scalar, single-valued data.

## Native and Expr validation

If `FeatureNativeCondition` is declared, every C value satisfying the Compiler's documented preconditions must combine correctly with standard root children. Native values may not appear below the root.

If `FeatureNativeExpression` is declared, every E value satisfying documented preconditions must preserve its position under AND, OR, and NOT. The E carrier may contain non-Boolean values; callers are responsible for choosing a valid filter expression. A Compiler may reject invalid values it can reliably identify, but core does not require a closed Boolean wrapper.

Normal fields and values should use safe typed or parameterized backend APIs. Native and Expr are escape hatches, not a reason to place ordinary values into raw query text.

## Errors

Return the zero value of C on every failure. Prefer a `*weave.Error` with accurate metadata:

```go
return zero, &weave.Error{
    Code:      weave.CodeInvalidValue,
    Phase:     weave.PhaseValidate,
    Path:      node.Path(),
    Origin:    node.Origin(),
    Operator:  operator,
    FieldType: reflect.TypeOf(field),
    ValueType: reflect.TypeOf(value),
    Cause:     sanitizedCause,
}
```

Validation and emission errors match `ErrCompile` through `Error.Is`. Factory wraps plain or sentinel errors into the compile-stage model and discards any nonzero C returned with an error.

`Error.Error` intentionally omits query values and `Cause.Error()`. Only retain a Cause whose text is safe to expose through explicit unwrapping. Compiler errors should be deterministic; report the first error in stable pre-order depth-first traversal.

## Output and determinism

A successful Compile must account for every node. It must not return a partial condition, silently discard an unsupported node, or replace a standard operator with an approximation. Repeated compilation of the same immutable Predicate and Compiler configuration must produce semantically identical output.

Backend optimization is allowed only when it is strictly semantics-preserving. Do not reorder or rewrite Native or Expr values, rewrite `NEQ` as a negated `EQ`, or erase a non-constant node because another child appears to determine the Boolean result.
