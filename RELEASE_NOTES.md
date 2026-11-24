# Release Notes: 1.0.0

Status: prepared, not yet tagged or published.

## Highlights

- The backend is now a layered Go service with real GraphQL parsing, schema validation, and query limits.
- Storage is pluggable, with an in-memory default and optional SQLite persistence.
- The API is safe by default: loopback binding, an exact origin allowlist, and a required write token on non-loopback addresses.
- The frontend is split into API, hook, and component modules on React 19 and Vite 8.
- Automated tests, a committed report from a real run, and a least-privilege CI workflow are included.

## Upgrade notes

1. Install Go 1.26.11 or newer and Node.js 24.
2. Start the backend with `go run ./cmd/server` from `backend/`.
3. Update API clients to read the GraphQL `errors` array.
4. Set `ALLOWED_ORIGINS` for any origin other than local Vite.
5. Set `API_WRITE_TOKEN` before binding to a non-loopback address.
6. To persist data, apply `db/schema.sql` and set `DATABASE_PATH`.

See [CHANGELOG.md](CHANGELOG.md) for the full list of changes.
