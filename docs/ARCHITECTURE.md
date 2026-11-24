# Architecture

## Overview

The system has two deployable parts: a Go backend that serves GraphQL over HTTP, and a static React frontend. In development and preview, Vite proxies `/graphql` to the backend so the browser talks to a single origin.

```text
+---------+   POST /graphql   +-----------+   +------------+   +--------+
| Browser | ----------------> | httpapi   |-->| graphqlapi |-->| domain |
| (React) |                   | transport |   | schema     |   | rules  |
+---------+                   +-----------+   +-----+------+   +--------+
                                                    |
                                                    v
                                              +-----------+
                                              | store     |
                                              | memory or |
                                              | SQLite    |
                                              +-----------+
```

## Backend modules

| Package | Responsibility | Depends on |
| --- | --- | --- |
| `cmd/server` | Wiring, server timeouts, graceful shutdown | all internal packages |
| `internal/config` | Parse environment, refuse unsafe configurations | standard library |
| `internal/domain` | `Category`, input validation, name folding | standard library |
| `internal/store` | `Repository` interface with memory and SQLite implementations | `domain` |
| `internal/graphqlapi` | Schema, resolvers, query limits, error masking | `domain`, `store` |
| `internal/httpapi` | Origin policy, content type, body limits, auth, rate limit, headers | `graphqlapi` |

Dependencies point inward. `domain` imports nothing from the project, and storage can be swapped behind `Repository`.

## Request lifecycle

1. `httpapi` applies security headers and the origin allowlist.
2. The per-client rate limit and the content type are checked.
3. The body is decoded with size limits.
4. `graphqlapi` parses the document and requires exactly one operation.
5. Depth, complexity, and mutation-count limits run before schema validation, so that hostile documents are rejected cheaply.
6. The document is validated against the schema.
7. Mutations are refused unless the caller has write access.
8. Resolvers call `domain` validation and then the repository.
9. Errors outside the domain are logged and replaced by a generic message.

## GraphQL API

```graphql
type Category {
  id: Int!
  name: String!
  description: String!
}

type Query {
  categories: [Category!]!
}

type Mutation {
  createCategory(name: String!, description: String): Category!
}
```

## Persistence

`db/schema.sql` defines the `categories` table for SQLite, with constraints mirroring the domain rules: length bounds, trimmed names, and a case-insensitive unique name. The `NOCASE` collation folds ASCII only, and `domain.NameKey` matches that behavior so both stores agree.

The in-memory store is the default and is intended for local use. SQLite runs in WAL mode with a busy timeout. Schema changes are applied explicitly by an operator.

## Frontend modules

| Path | Responsibility |
| --- | --- |
| `src/api/client.js` | GraphQL transport and error normalization |
| `src/hooks/useCategories.js` | Loading, error, and list state |
| `src/components/CategoryForm.jsx` | Controlled input form |
| `src/components/CategoryTable.jsx` | List, loading, and empty states |
| `src/App.jsx` | Page composition |

## Recommended deployment

Serve `frontend/dist` and proxy `/graphql` from one TLS-terminating reverse proxy, keeping the backend on a loopback or private address. Set `API_WRITE_TOKEN`, set `ALLOWED_ORIGINS` to the public origin, and apply the Content Security Policy from `frontend/vite.config.js` at the proxy, because `vite preview` is not a production server. Enforce per-client rate limits at the proxy, since the backend sees only the proxy address.

For multiple backend replicas, replace SQLite with a networked database behind the `Repository` interface and move rate limiting out of process.
