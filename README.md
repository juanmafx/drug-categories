# Drug Categories

A small full-stack catalog for browsing and adding drug categories, built with a Go GraphQL API and a React frontend.

> This project manages category labels only. It contains no dosing, interaction, or clinical data and is not medical advice.

## Purpose

Drug Categories keeps a list of named categories, each with an optional description, and exposes it through one typed GraphQL endpoint and a single-page UI.

## Why this project exists

It is a compact reference for a cleanly layered Go and React application: a narrow domain, explicit boundaries between transport, API, domain, and storage, and defensive handling of untrusted input at each boundary.

## What it does

- Lists categories through the `categories` query.
- Creates categories through the `createCategory` mutation.
- Validates names and descriptions, and rejects case-insensitive duplicate names (ASCII case folding).
- Stores data in memory by default, or in SQLite when `DATABASE_PATH` is set.
- Does not update or delete categories.

## Architecture

```text
Browser (React) -> /graphql (Vite proxy) -> httpapi -> graphqlapi -> domain
                                                                  -> store (memory or SQLite)
```

```text
backend/
  cmd/server/          Process entrypoint and graceful shutdown
  internal/config/     Environment parsing and fail-closed checks
  internal/domain/     Category model and validation
  internal/store/      Repository interface, memory and SQLite stores
  internal/graphqlapi/ Schema, resolvers, and query limits
  internal/httpapi/    HTTP handler, origin policy, auth, rate limiting
frontend/
  src/api/             GraphQL client
  src/hooks/           Data-loading state
  src/components/      Form and table
  src/styles/          Stylesheet
  tests/               Vitest suites
db/schema.sql          SQLite schema for the optional durable store
reports/tests/         Committed output of a real test run
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for details.

## Quick start

Requires Go 1.26.11 or newer and Node.js 24 (see `.nvmrc`).

```bash
cd backend
go run ./cmd/server
```

In a second terminal:

```bash
cd frontend
npm ci --ignore-scripts
npm run dev
```

Open the URL printed by Vite, usually `http://localhost:5173`.

## Usage guide

Use the form to add a category. If the server has a write token configured, enter it in the token field. The token is held in component memory only and is never written to browser storage.

Query the API directly:

```bash
curl -s http://127.0.0.1:8080/graphql \
  -H 'Content-Type: application/json' \
  -d '{"query":"{ categories { id name description } }"}'
```

Create a category:

```bash
curl -s http://127.0.0.1:8080/graphql \
  -H 'Content-Type: application/json' \
  -d '{"query":"mutation($name:String!,$description:String){ createCategory(name:$name,description:$description){ id name } }","variables":{"name":"Antivirals","description":"Treat viral infections"}}'
```

Add `-H "Authorization: Bearer $API_WRITE_TOKEN"` when a token is configured.

### Durable storage (optional)

Without `DATABASE_PATH`, data is lost when the process exits. To persist it:

```bash
mkdir -p data
sqlite3 data/categories.db < db/schema.sql
cd backend
DATABASE_PATH=../data/categories.db go run ./cmd/server
```

The server never creates or migrates the database. It refuses to start if the schema has not been applied.

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `HTTP_ADDR` | `127.0.0.1:8080` | Listen address. |
| `ALLOWED_ORIGINS` | Local Vite dev and preview origins | Comma-separated exact browser origins. Wildcards are rejected. |
| `DATABASE_PATH` | empty | SQLite file path. Empty selects the in-memory store. |
| `API_WRITE_TOKEN` | empty | Bearer token for mutations, minimum 32 characters. Required on non-loopback addresses. |
| `BACKEND_URL` | `http://127.0.0.1:8080` | Frontend dev and preview proxy target. |

See [.env.example](.env.example). The server reads the process environment and does not load `.env` files itself.

## Testing

```bash
cd backend && go test -race ./...
cd frontend && npm ci --ignore-scripts && npm test
```

Regenerate the committed report from the repository root:

```bash
./scripts/generate-test-report.sh
```

Results of the latest recorded run are in [reports/tests/SUMMARY.md](reports/tests/SUMMARY.md). CI is defined in `.github/workflows/ci.yml`.

## Security notes

- Mutations require a bearer token whenever one is configured, and the server will not bind to a non-loopback address without one.
- Browser requests are checked against an exact origin allowlist, and only `application/json` is accepted.
- Request size, query depth, query complexity, and per-client request rate are bounded.
- SQL is parameterized, and internal errors are logged rather than returned to clients.
- The rate limiter keys on the socket peer address. Behind a reverse proxy all clients share one bucket, so apply limits at the proxy as well.
- TLS is not terminated by this service. Use a TLS-terminating proxy for any non-local deployment.

See [SECURITY.md](SECURITY.md) and [docs/SECURITY_AUDIT.md](docs/SECURITY_AUDIT.md).

## Project history

Originally created in 2026 and modernized in 2026.

## License

This repository does not currently include a license file.
