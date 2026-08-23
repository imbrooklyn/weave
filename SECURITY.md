# Security Policy

## Supported versions

The repository does not currently declare a tagged support line. Security fixes are prepared on the default branch.

## Report a vulnerability

Please report suspected vulnerabilities privately through [GitHub Security Advisories](https://github.com/imbrooklyn/weave/security/advisories/new). Do not open a public issue for a vulnerability that has not been coordinated.

Include enough information to reproduce and assess the issue without including production credentials or unrelated personal data:

- Affected package, API, and revision or version.
- Security impact and realistic attack conditions.
- Minimal reproduction or test case.
- Whether `Native` or `Expr` is involved.
- Any known mitigation.

Maintainers will use the private advisory to confirm scope, coordinate a fix, and prepare disclosure. Public details should be released only after affected users have a reasonable remediation path.

## Security boundaries

Weave constructs and compiles predicates; it does not execute queries. Compiler implementations must keep ordinary fields and values in the backend's safe identifier, parameter, or typed-expression mechanisms. Error messages must not expose query values, field values, Native payloads, Expr payloads, or credentials.

`Native(C)` and `Expr(E)` are explicit escape hatches. Core does not prove their backend validity, Boolean meaning, parameterization, escaping, or immutability. Passing untrusted raw query text through either API is outside the safety guarantees of standard operators.

Mutating borrowed payloads while a Predicate is read or compiled can create races or change backend behavior. Follow the ownership and concurrency contracts in the [README](README.md).
