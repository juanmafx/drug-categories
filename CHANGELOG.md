# Changelog

All notable changes are documented here. The format follows Keep a Changelog, and the project uses Semantic Versioning.

## [1.0.0] - Unreleased

### Breaking

- Errors now use the GraphQL `errors` array instead of a top-level `error` string.
- Validation failures from resolvers return HTTP 200 with `errors`, instead of HTTP 400.
- Category names must be unique, ignoring ASCII case.
- The server binds to `127.0.0.1` by default and refuses non-loopback addresses without `API_WRITE_TOKEN`.
- Browser requests are limited to `ALLOWED_ORIGINS`.
- The backend entrypoint moved to `backend/cmd/server`.
- The frontend calls a same-origin `/graphql` path through a proxy.
- Node.js 24 and Go 1.26.11 are now required.

### Added

- Real GraphQL parsing and validation.
- Optional SQLite persistence and `db/schema.sql`.
- Optional bearer-token protection for mutations.
- Query token, depth, complexity, body, and rate limits, plus security headers and graceful shutdown.
- Rejection of unused, duplicate, and cyclic fragments, non-string values for `String` variables, and invisible formatting characters in category text.
- Backend and frontend test suites, a committed test report, and a CI workflow.
- Architecture, security, and release documentation.

### Changed

- Restructured the backend and frontend into nested modules.
- Upgraded to React 19 and Vite 8.

### Fixed

- A data race on the shared category list.
- Requests routed by substring match on the query text.

## [0.0.1] - 2026-03-10

- Initial Go and React application.
