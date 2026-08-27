# Changelog

This file records notable user-visible changes to Weave. It follows the structure of [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and uses [Semantic Versioning](https://semver.org/) for tagged releases.

## [Unreleased]

### Changed

- Frozen Group method calls now record `ErrInvalidState` before evaluating inclusion predicates, `enabled` values, or nested Scopes.
- The canonical `compilertest` scenarios now cover the empty identities of all four Boolean Logic forms explicitly.

## [0.1.0-alpha.1] - 2026-08-26

### Added

- Backend-neutral predicate construction with typed concrete generic methods.
- Comparison, membership, numeric range, null, and literal-text operators.
- Explicit `AllOf`, `AnyOf`, `NoneOf`, and `NotAllOf` Boolean groups.
- Structurally immutable normalized Predicate snapshots and sealed read-only node views.
- Compiler capabilities, normalized Predicate requirements, and Factory preflight.
- Structured, redacted, location-aware errors.
- Go 1.27 compile-contract fixtures, fuzz targets, race coverage, and benchmark baselines.
- Backend-neutral `compilertest` records and harness callbacks covering semantic match sets, capabilities, redacted validation errors, backend-owned condition inspection, and repeat/concurrency stability.
- Read-only `compilertest.Scenarios` for executable consumers, including canonical expected-ID variants when a backend materializes missing values as null.

[Unreleased]: https://github.com/imbrooklyn/weave/compare/v0.1.0-alpha.1...HEAD
[0.1.0-alpha.1]: https://github.com/imbrooklyn/weave/releases/tag/v0.1.0-alpha.1
