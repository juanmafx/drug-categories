# Test Report

- Generated (UTC): 2026-11-24T22:55:02Z
- Commit base: bbf7a6a
- Go: go1.26.11
- Node: v24.22.0

| Suite | Passed | Failed | Exit code |
| --- | --- | --- | --- |
| Backend (go test -race, including subtests) | 91 | 0 | 0 |
| Frontend (Vitest) | 14 | 0 | 0 |

## Backend package coverage

```text
ok  	github.com/juanmafx/drug-categories/backend/internal/config	coverage: 100.0% of statements
ok  	github.com/juanmafx/drug-categories/backend/internal/domain	coverage: 100.0% of statements
ok  	github.com/juanmafx/drug-categories/backend/internal/graphqlapi	coverage: 96.9% of statements
ok  	github.com/juanmafx/drug-categories/backend/internal/httpapi	coverage: 95.5% of statements
ok  	github.com/juanmafx/drug-categories/backend/internal/store	coverage: 91.0% of statements
```
