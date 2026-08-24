# Changelog

This file records notable user-visible changes to Weave. It follows the structure of [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and uses [Semantic Versioning](https://semver.org/) for tagged releases.

## [Unreleased]

### Added

- Backend-neutral predicate construction with typed concrete generic methods.
- Comparison, membership, numeric range, null, and literal-text operators.
- Explicit `AllOf`, `AnyOf`, `NoneOf`, and `NotAllOf` Boolean groups.
- Structurally immutable normalized Predicate snapshots and sealed read-only node views.
- Compiler capabilities, normalized Predicate requirements, and Factory preflight.
- Structured, redacted, location-aware errors.
- Go 1.27 compile-contract fixtures, fuzz targets, race coverage, and benchmark baselines.
- Backend-neutral `compilertest` records and harness callbacks covering semantic match sets, capabilities, redacted validation errors, backend-owned condition inspection, and repeat/concurrency stability.

[Unreleased]: https://github.com/imbrooklyn/weave/commits/main
