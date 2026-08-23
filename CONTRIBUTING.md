# Contributing to Weave

Thank you for helping improve Weave. Contributions should preserve its backend-neutral semantics, small dependency surface, and deterministic behavior.

## Requirements

- Go 1.27 or newer.
- Git.
- No external service is required for core package tests.

## Set up the repository

Clone the repository and verify the toolchain:

```sh
git clone https://github.com/imbrooklyn/weave.git
cd weave
go version
```

The `go version` output must report Go 1.27 or newer. The runtime module must continue to depend only on the Go standard library.

## Make a focused change

- Keep public identifiers, GoDoc, comments, examples, tests, scripts, and documentation in English.
- Add tests for every behavior change and regression fix.
- Preserve exact match-set, null/missing, empty collection, empty Group, ownership, and error semantics.
- Do not silently ignore unsupported nodes or replace a standard operator with an approximation.
- Keep Compiler implementations request-stateless; do not add database, session, context, logger, or transaction state to the core contract.
- Treat `Native` and `Expr` as explicit escape hatches. Do not weaken ordinary field or value safety to make them more convenient.
- Avoid new runtime dependencies. Adding one requires a concrete compatibility and maintenance justification.

## Format and verify

Run the complete local checks before opening a pull request:

```sh
go fmt ./...
go test ./...
go test -race ./...
go vet ./...
```

Run package examples explicitly:

```sh
go test -run '^Example' ./...
```

Compile-contract tests run as part of `go test ./...` and require a Go 1.27 toolchain. When changing generic method signatures, named-slice behavior, `when` predicate types, or downstream alias compatibility, update the positive and negative fixtures under `testdata/compile`. Negative fixtures must continue to assert the intended compiler diagnostic location.

For fuzz-related changes, run the affected target for an appropriate duration. A short smoke example is:

```sh
go test -run '^$' -fuzz '^FuzzBuilderConstructionSequence$' -fuzztime=10s .
```

For allocation or traversal changes, record a benchmark comparison without inventing an absolute threshold:

```sh
go test -run '^$' -bench '^Benchmark(PredicateSnapshot|FactoryCompile)' -benchmem .
```

## Runtime dependency check

Inspect runtime dependencies for both public packages:

```sh
go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' . ./when
```

The only non-standard-library paths in that output should be `github.com/imbrooklyn/weave` and `github.com/imbrooklyn/weave/when` themselves.

## Documentation expectations

Every exported symbol needs accurate GoDoc. Package documentation and runnable examples should explain semantic edge cases close to the API that exposes them. Public enum integer values are implementation details; document their named meanings and stable `String` identifiers, not their numeric ordinals.

Update README, focused documents, and the changelog when a user-visible contract changes. Links, code, commands, and version requirements must remain executable and internally consistent.

## Pull requests

A pull request should:

- Explain the user-visible problem and resulting behavior.
- Keep unrelated refactoring separate.
- Include tests that would fail without the change.
- List the verification commands that were run.
- Call out compatibility, ownership, concurrency, security, or allocation effects.
- Avoid generated artifacts, local workspaces, coverage output, profiling data, credentials, and editor files.

By contributing, you agree that your contribution is licensed under the repository's [Apache License 2.0](LICENSE).
