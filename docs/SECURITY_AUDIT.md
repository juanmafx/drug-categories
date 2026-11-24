# Security Audit

Scope: all tracked files and the single commit of history on `main`, reviewed on 2026-11-23 during the 1.0.0 modernization. This was a self-review supported by automated tools, not an independent audit.

## Tools run

| Check | Result |
| --- | --- |
| `govulncheck` v1.10.0 on Go 1.26.11 | No vulnerabilities found |
| `npm audit` on the 1.0.0 lockfile | 0 vulnerabilities |
| `go vet`, `go test -race` | Clean |
| Manual search of tracked files and history for credentials | None found |

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

## Known limitations

- The service does not terminate TLS, so a bearer token is only protected in transit when a TLS proxy is in front of it.
- Reads are public by design.
- There is a single shared write token, with no per-user identity, rotation, or audit log.
- Rate-limit state is per process and is keyed on the socket peer, so it is less effective behind a proxy.
- `vite preview` is not a production server, and the frontend CSP must be applied by the deployment proxy.
- Introspection stays enabled, subject to the query limits.

## Not applicable

The project has no contact form, user accounts, passwords, or personal data, so two-factor authentication and password hashing do not apply.
