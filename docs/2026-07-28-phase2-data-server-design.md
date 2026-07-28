# Phase 2: Data Server Design

**Date**: 2026-07-28
**Status**: Approved for implementation

## Objective

Provide out-of-band full-dataset retrieval for the AVEVA Historian MCP Server. MCP tools return ≤100-row previews inline; the data server serves the complete result set via HTTP with cursor-based pagination, TTL-based GC, and pin/delete lifecycle.

## Architecture

```
┌──────────────────────┐  stdio  ┌──────────────┐
│   AI Agent (Claude)  │◄───────►│  MCP Server   │
│                      │         │  (stdio)      │
└──────────────────────┘         └──────┬─────────┘
                                        │ push full result to store
                                        │ return ResourceURI in DualToolResult
                                        ▼
                               ┌──────────────────┐
                               │   Data Server     │  HTTP 127.0.0.1:{DATA_SERVER_PORT}
                               │  (goroutine)      │◄──── AI agent follows ResourceURI
                               │  - Store          │       for full dataset pages
                               │  - GC goroutine   │
                               └──────────────────┘
```

- In-process goroutine, same binary. Start in `main()` before `server.Run()`.
- Binds to `DATA_SERVER_BIND` (default `127.0.0.1`), port `DATA_SERVER_PORT` (default `8199`).
- `ResourceLink.URI` = `http://{bind}:{port}/resources/{uuid}`
- GC sweep goroutine runs alongside, driven by `DATA_SERVER_GC_INTERVAL` (default 1 min).

## Data Model

```go
type Field struct {
    Name string   // "FQN", "DateTime", "Value"
    Type string   // "string", "number", "datetime" — hint for client
}

type Row struct {
    ID    int64    // auto-increment per resource, 1-based, stable
    Cells []string // stringified values, order matches Fields
}

type Resource struct {
    ID        uuid.UUID
    Label     string
    MimeType  string           // "text/csv" (MVP), "application/json" (future)
    CreatedAt time.Time
    TTL       time.Duration
    Pinned    bool
    MaxAge    *time.Duration   // nil = indefinite pin

    Columns  []Field
    Rows     []Row
    RowCount int
}

type Store struct {
    mu    sync.RWMutex
    items map[uuid.UUID]*Resource
    ttl   time.Duration        // global default from config
    clock func() time.Time     // injectable for tests
}
```

- Rows stored as `[]Row`. No mutation after insert except pin/delete.
- Row.ID enables cursor pagination: `WHERE Row.ID > N` ORDER BY ID ASC.
- `Field.Type` optional hint. MCP handler knows types from Historian metadata.

## HTTP API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/resources` | List all resources (summaries) |
| GET | `/resources/{id}` | First page (default 100 rows) |
| GET | `/resources/{id}?cursor=N` | Next page after Row.ID = N |
| POST | `/resources/{id}/pin` | Pin resource, optional `{"maxAge":"15m"}` |
| DELETE | `/resources/{id}` | Delete resource |

### GET /resources

Returns `[]ResourceSummary`:
```json
[
    {
        "id": "abc-123",
        "label": "ProcessValues: Machine1_Pressure",
        "rowCount": 8542,
        "createdAt": "2026-07-28T12:00:00Z",
        "ttl": "5m0s",
        "pinned": false,
        "expiresAt": "2026-07-28T12:05:00Z"
    }
]
```

### GET /resources/{id}[?cursor=N]

Returns first or next page. Default limit 100. `cursor=0` treated as first page (same as no cursor).

```json
{
    "resource": {
        "id": "abc-123",
        "label": "ProcessValues: Machine1_Pressure",
        "columns": [
            {"name": "FQN", "type": "string"},
            {"name": "DateTime", "type": "datetime"},
            {"name": "Value", "type": "number"}
        ],
        "rowCount": 8542
    },
    "page": {
        "rows": [
            {"id": 1, "cells": ["CDE158.Pressure", "2026-01-15T10:00:00Z", "42.5"]}
        ],
        "cursor": 100,
        "hasMore": true
    }
}
```

### POST /resources/{id}/pin

Body optional: `{"maxAge": "15m"}` — override TTL. Omit body → indefinite pin. Returns 200.

### DELETE /resources/{id}

Returns 204 on success, 404 if not found.

### Errors

```json
{"error": "resource not found"}
```

With appropriate HTTP status (404, 400 for bad cursor, 409 for already pinned, 405 for invalid operations).

## Lifecycle & GC

- Default TTL: 5 min (configurable via `DATA_SERVER_DEFAULT_TTL` env var).
- GC runs every `DATA_SERVER_GC_INTERVAL` (default 1 min).
- Pin marks `Pinned=true`. If `MaxAge` set, GC checks inequality.
- Re-pin with a new `maxAge` replaces the previous MaxAge value.
- Re-pin without request body makes the pin indefinite (clears MaxAge).
- Delete bypasses pin — immediate removal.
- GC goroutine shares `context.Context` with server for graceful shutdown.

```
Created ──→ 5 min TTL ──→ Swept (if unpinned)
               │
               └── pinned ──→ MaxAge or until DELETE
```

## Cursor Strategy

Auto-increment int64 per resource (1-based). Forward-only.
- `cursor=N` → rows where `Row.ID > N`, LIMIT 100.
- `cursor` in response = `Row.ID` of last row in page (0 if empty).
- `hasMore` = true when there are rows beyond the page.

## Store Methods

```go
func (s *Store) Put(label, mimeType string, columns []Field, rows [][]string) (*Resource, error)
func (s *Store) Get(id uuid.UUID) (*Resource, bool)
func (s *Store) Page(id uuid.UUID, cursor int64, limit int) (rows []Row, nextCursor int64, hasMore bool)
func (s *Store) List() []ResourceSummary
func (s *Store) Pin(id uuid.UUID, maxAge *time.Duration) error
func (s *Store) Delete(id uuid.UUID) error
func (s *Store) Sweep(now time.Time) int
```

## Configuration (new env vars)

| Variable | Default | Description |
|----------|---------|-------------|
| `DATA_SERVER_BIND` | `127.0.0.1` | Bind address |
| `DATA_SERVER_PORT` | `8199` | HTTP port |
| `DATA_SERVER_DEFAULT_TTL` | `5m` | Default resource TTL (Go duration) |
| `DATA_SERVER_GC_INTERVAL` | `1m` | GC sweep interval (Go duration) |

## Failure Mode: Data Server Unavailable

If the data server fails to bind (port in use, bad address), MCP server still starts:
- MCP tool handlers log a warning and operate without ResourceURI in DualToolResult.
- Preview-only mode — no out-of-band full dataset available.
- The error is logged at startup and on each resource push attempt.

## Integration with MCP Tools

When an MCP tool handler (e.g. `read_process_values`) fetches data:

1. Fetch full result from Historian via existing endpoint function.
2. Format first 100 rows as preview text → `DualToolResult.Preview`.
3. Push full result to data server store → get `*Resource` back.
4. Set `DualToolResult.ResourceURI` = `http://{bind}:{port}/resources/{resource.ID}`.

Data server does NOT re-query Historian. Consistency between preview and full set is guaranteed.

## Files

```
internal/dataserver/
├── server.go        # HTTP router, handlers, middleware (recovery, logging)
├── store.go         # Store, Resource, Row, Field types + methods
├── lifecycle.go     # GC goroutine, Sweep
└── server_test.go   # Tests via httptest
```

## Testing Strategy

| Scope | Method | What |
|-------|--------|------|
| Store unit | `store_test.go` | Put/Get/Page/List/Pin/Delete/Sweep |
| HTTP handlers | `server_test.go` | httptest.Server, full request/response round-trip |
| Cursor edge cases | In store tests | Empty store, last page, cursor past end, negative cursor |
| GC | `lifecycle_test.go` | Sweep with clock injection, pinned vs unpinned, MaxAge expiry |

## Edge Cases

- **Cursor past end**: `Page` returns empty rows, `hasMore=false`, `cursor=0`.
- **Negative cursor**: treated as first page (same as no cursor).
- **Empty rows**: valid resource with 0 rows. Returned normally.
- **Concurrent pin + delete**: Delete wins. Returns 404.
- **Double pin**: Returns 200 (idempotent). May update MaxAge.
- **GC during page read**: RLock on Store.Get prevents mid-read sweep.
- **Data server on wrong port**: MCP handler logs error, returns error in DualToolResult with no ResourceURI.
