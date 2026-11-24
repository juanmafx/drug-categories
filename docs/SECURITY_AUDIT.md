# Security Audit

Scope: all tracked files and the single commit of history on `main`, reviewed on 2026-11-23 during the 1.0.0 modernization. A second automated review pass on the same date tested the branch adversarially and fixed the items under "Findings from the second review". Both were AI-assisted reviews supported by automated tools, not an independent third-party audit. On 2026-11-24 the published history was rewritten to keep only the 2026 modernization commits, so the original commit reviewed here is no longer part of any branch or tag.

## Tools run

| Check | Result |
| --- | --- |
| `govulncheck` v1.10.0 on Go 1.26.11 | No vulnerabilities found |
| `npm audit` on the 1.0.0 lockfile | 0 vulnerabilities |
| `go vet`, `go test -race` | Clean |
| `gitleaks` with `--redact` on branch history | 2 hits, both synthetic test fixtures in the original commit, replaced with generated values. Resolved by the 2026-11-24 history rewrite, which removed that commit from every branch and tag. Any value that was ever published must still be treated as compromised |

## Findings fixed in 1.0.0

| Area | Previous behavior | Resolution |
| --- | --- | --- |
| Cross-origin access | Wildcard CORS on a state-changing endpoint | Exact origin allowlist |
| CSRF | Any JSON body was parsed regardless of content type | `application/json` required, disallowed origins rejected |
| Authorization | Anyone able to reach the port could write | Optional bearer token, mandatory off loopback, compared in constant time |
| Network exposure | Listened on all interfaces | Binds to loopback by default |
| Query handling | Operations were selected by substring match | Real GraphQL parsing and schema validation |
| Resource exhaustion | No limits on body, query, or request rate | Body, depth, complexity, mutation-count, and rate limits |
| Server timeouts | None | Header, read, write, and idle timeouts |
| Data race | Unsynchronized shared slice | Mutex-protected store, verified with the race detector |
| Input validation | Name presence only | Length, control-character, UTF-8, and uniqueness checks |
| Response headers | None | CSP, `nosniff`, referrer, cache, and resource policies |
| Outdated toolchain | Go 1.22 and Vite 5 | Go 1.26.11, Vite 8, React 19, Node.js 24 |

## Findings from the second review

| Severity | Finding | Resolution |
| --- | --- | --- |
| Critical | One unauthenticated request containing an unused fragment cycle overflowed the stack in the GraphQL library's validator and terminated the server. Reproduced against a local build | Cycles, unused fragments, and duplicate fragments are rejected before validation |
| Medium | Near-limit documents of nested lists or repeated variables cost over 0.6 seconds of CPU each and could return about 2,000 errors | 1,000-token budget before parsing, and at most 10 validation errors returned |
| Low | Objects and numbers sent for `String` variables were coerced into stored text | Non-string values are rejected |
| Low | Invisible, bidirectional, and line-separator characters were accepted, allowing lookalike duplicate names | Rejected by domain validation |
| Low | A whitespace-only `API_WRITE_TOKEN` satisfied the non-loopback check while making writes impossible | Tokens with whitespace or control characters are refused at startup |
| Low | The startup log reported "listening" before the port was bound | The socket is bound first, and bind failures are reported |
| Low | The committed JUnit report contained the local machine hostname | Removed by the report script |

## Known limitations

- The service does not terminate TLS, so a bearer token is only protected in transit when a TLS proxy is in front of it.
- Reads are public by design.
- There is a single shared write token, with no per-user identity, rotation, or audit log.
- Rate-limit state is per process and is keyed on the socket peer, so it is less effective behind a proxy.
- `vite preview` is not a production server, and the frontend CSP must be applied by the deployment proxy.
- Introspection stays enabled, subject to the query limits. The depth limit blocks the full standard introspection query.
- The `categories` query is not paginated, so response size grows with the number of stored categories.

## Not applicable

The project has no contact form, user accounts, passwords, or personal data, so two-factor authentication and password hashing do not apply.
