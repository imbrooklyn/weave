# Pull request

## Summary

Describe the user-visible problem and the resulting behavior.

## Contract impact

- [ ] Operator or Boolean semantics
- [ ] Null, missing, empty collection, or empty Group behavior
- [ ] Public API or GoDoc
- [ ] Snapshot ownership or concurrency
- [ ] Capabilities, Requirements, or Compiler behavior
- [ ] Error classification or diagnostics
- [ ] No public contract change

Explain every selected item, including compatibility and security implications.

## Verification

List the exact commands you ran and their results.

- [ ] `gofmt`
- [ ] `go test ./...`
- [ ] `go test -race ./...`
- [ ] `go vet ./...`
- [ ] Relevant examples, compile fixtures, fuzz targets, or benchmarks

## Safety checklist

- [ ] Tests cover the behavior or regression.
- [ ] Errors and test output do not expose query values, field values, Native payloads, Expr payloads, or credentials.
- [ ] Runtime packages still depend only on the Go standard library.
- [ ] The change does not silently ignore or approximate an unsupported node.
- [ ] Documentation and examples match the implemented behavior.
