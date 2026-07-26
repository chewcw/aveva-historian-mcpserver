# AVEVA Historian MCP Server

Go MCP server bridging AI agents → AVEVA Historian time-series data.

## Commands

```bash
Build:  go build -o bin/aveva-historian-mcp ./cmd/server
Run:    go run ./cmd/server
Test:   go test ./... -count=1
Lint:   go vet ./...
Single: go test -run TestName ./internal/... -count=1
```

## Code Style

```go
// Package name: lowercase, single word. One responsibility per package.
package tags

// Error wrapping: always add context. Use %w, never %v for errors.
if err := client.Get(ctx, "Tags", q.Encode(), &result); err != nil {
    return nil, fmt.Errorf("fetch tags: %w", err)
}

// Context is always first param in functions that I/O.
func GetTags(ctx context.Context, client *historian.Client, filter string, top, skip int) (*historian.ODataResponse[Tag], error) {

// Zero-value structs over constructors. Functional options for optional config.
type Client struct { Timeout time.Duration; BaseURL string }  // OK
// Option functions — NOT NewClient() inside init().

// Named returns only for exported functions that could need godoc of return shape.
```

## Imports

Group order: stdlib → internal → external, separated by blank line.

When importing a package whose name doesn't match its namespace (last path segment), always use an explicit alias for clarity:

```go
import (
    "context"
    "fmt"
    "net/http"

    "github.com/chewcw/aveva-historian-mcpserver/internal/historian"

    odataqb "github.com/chewcw/odata-query-builder"
    "github.com/google/uuid"
)
```

## Error Handling

- Wrap every error with context: `fmt.Errorf("what failed: %w", err)`
- Tool handlers return `*mcp.CallToolResult` with `IsError: true` — never panic.
- HTTP client wraps upstream errors with endpoint + params for debugging.
- No `init()` functions. No global mutable state. No `panic`/`recover` except in `main()`.

## Testing

- Tests alongside source: `foo_test.go` in same package.
- Mock Historian with `httptest.NewServer` — no external dependency.
- Test behaviors, not plumbing. Each test defends an observable contract.
- Use `-count=1` to disable test caching.

```go
func TestGetTags(t *testing.T) {
    mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`{"value": [{"FQN": "test", "TagName": "Test"}]}`))
    }))
    defer mock.Close()
    // ...
}
```

## What NOT to do

- **NEVER embed >100 rows** in MCP `CallToolResult`. Use `ResourceLink` for full datasets.
- **NEVER use offset pagination** for time-series data. Keyset cursor on `(timestamp, id)` only.

- **NEVER commit `.env`** with real credentials. Use `.env.example`.
- **NEVER add init()** or package-level mutable state.
- **NEVER use Kerberos** — NTLM only. (go.mod has krb5 dep — ignore it, unused.)
- **NEVER edit `vendor/`**, generated files, or `.codegraph/` directory.

## Git

```
Conventional commits: feat:|fix:|chore:|docs:|refactor:
Branch: type/short-description  (e.g. feat/tags-endpoint)
Squash merge only to main.
```

## Comments

- **No session-specific comments.** "Chose Option A" means nothing in 6mo — reader doesn't know Option B.
- If rationale needed, include full context: alternatives considered, tradeoff that drove choice.
- Good: `// Offset pagination safe here — Tags endpoint data is static, never appended.`
- Bad: `// Using offset instead of cursor because it's simpler.`

## Key entry points

| File | What it does |
|------|-------------|
| `cmd/server/main.go` | `main()`, wiring, serve |
| `internal/config/config.go` | `.env` → Config struct |
| `internal/historian/client.go` | NTLM HTTP + OData envelope parse |
| `internal/mcp/server.go` | MCP server + capability registration |
| `internal/mcp/tools/*.go` | One file per MCP tool handler |
| `internal/historian/endpoints/*.go` | One file per API group |
| `internal/dataserver/*.go` | REST out-of-band server |

