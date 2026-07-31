# aveva-historian-mcpserver

MCP server (Go) that bridges AI agents to AVEVA Historian time-series data. Agents discover tags, read process values, and pull analog summaries through standard MCP tools, backed by NTLM-authenticated access to the AVEVA Historian REST API.

## How it works

Historian datasets can be far larger than an LLM context window. This server uses a dual-response pattern:

1. **Preview** — each MCP tool returns at most 100 rows for LLM reasoning, plus a `ResourceLink` URI.
2. **Full dataset** — the link points to a built-in HTTP data server that serves the complete result out-of-band, with TTL-based lifecycle management.

The data server keeps full payloads out of MCP tool results and out of the model context; clients fetch them directly when needed.

## Features

- MCP stdio transport (works with Claude Desktop, Cursor, Copilot, and any MCP client)
- NTLM authentication against AVEVA Historian REST API
- OData query building for tags, process values, and analog summaries
- Keyset-cursor data server with TTL, GC sweep, and pin/delete resource lifecycle
- Structured logging via `log/slog`
- Config via `.env` with validation

## Requirements

- Go 1.26.5+
- Access to an AVEVA Historian server with valid credentials

## Build

```bash
go build -o bin/aveva-historian-mcp ./cmd/server
```

Cross-compile for Windows (common for NTLM environments):

```bash
GOOS=windows GOARCH=amd64 go build -o bin/aveva-historian-mcp.exe ./cmd/server
```

## Configuration

Copy `.env.example` to `.env` and fill in credentials.

| Variable | Required | Default | Description |
|---|---|---|---|
| `AVEVA_HISTORIAN_BASE_URL` | yes | — | Historian REST API base URL (e.g. `http://historian-server:32569`) |
| `AVEVA_HISTORIAN_USERNAME` | yes | — | NTLM username |
| `AVEVA_HISTORIAN_PASSWORD` | yes | — | NTLM password |
| `AVEVA_HISTORIAN_API_PATH_PREFIX` | no | `/Historian/v2` | API path prefix |
| `AVEVA_HISTORIAN_RESULT_BYTE_LIMIT` | no | `1048576` | Max inline result bytes before falling back to `ResourceLink` |
| `MCP_SERVER_NAME` | no | `aveva-historian-mcp` | Server name reported during MCP initialize |
| `MCP_SERVER_VERSION` | no | `dev` | Server version reported during MCP initialize |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, or `error` |
| `LOG_FILE_PATH` | no | — | Optional log file; empty logs to stderr only |
| `DATA_SERVER_BIND` | no | `127.0.0.1` | Data server bind address |
| `DATA_SERVER_PORT` | no | `8199` | Data server port |
| `DATA_SERVER_DEFAULT_TTL` | no | `5m` | Default resource lifetime (e.g. `5m`, `1h`) |
| `DATA_SERVER_GC_INTERVAL` | no | `1m` | GC sweep interval for expired resources |

Never commit a real `.env` — keep credentials out of version control.

## Usage

Add the server to your MCP client config. For Claude Desktop (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "aveva-historian": {
      "command": "/absolute/path/to/bin/aveva-historian-mcp",
      "args": [],
      "env": {
        "AVEVA_HISTORIAN_BASE_URL": "http://historian-server:32569",
        "AVEVA_HISTORIAN_USERNAME": "your-user",
        "AVEVA_HISTORIAN_PASSWORD": "your-password"
      }
    }
  }
}
```

For clients that launch from a working directory, ensure the server process can read `.env` (config is loaded from the current working directory), or set all variables in the client env block.

## Tools

| Tool | Description |
|---|---|
| `get_tags` | List/search historian tags with filters: `top`, `skip`, `orderby`, `count`, `search`, `source`, and tag property filters (`tag_type`, `tag_name`, `description`, `eng_unit`, `eng_unit_min`, `eng_unit_max`, `interpolation_type`, `integral_divisor`, `rollover_value`, `message_on`, `message_off`, `filters`) |
| `read_process_values` | Time-series process values for a tag over a time range. Optional `retrieval_mode` (Average, Cyclic, Integral, Minimum, Maximum, BestFit, Delta, Interpolated, Slope, Counter, Full) and `resolution_ms` |
| `read_analog_summary` | Analog summary statistics per cycle over a time range: Min, Max, Avg, StdDev, Integral, Count, First, Last, OPCQuality, PercentGood, with timestamps |

Large results return a `ResourceLink` (≤100-row preview inline); fetch the link from the data server for the full dataset.

## Data server

Embedded HTTP server serving resource payloads. Endpoints:

| Method | Path | Description |
|---|---|---|
| `GET` | `/resources` | List resources |
| `GET` | `/resources/{id}` | Fetch a resource payload |
| `POST` | `/resources/{id}/pin` | Prevent TTL expiry (pin the resource) |
| `DELETE` | `/resources/{id}` | Delete a resource immediately |

Expired resources are removed by the background GC sweep (`DATA_SERVER_GC_INTERVAL`).

## Development

```bash
# Test (no cache)
go test ./... -count=1

# Lint
go vet ./...

# Run (stdio — pipe into an MCP client)
go run ./cmd/server
```

Conventions: wrap every error with context (`fmt.Errorf("what: %w", err)`), pass `context.Context` first in any I/O function, no `init()` or package-level mutable state, no offset pagination for time-series data. Tests mock Historian with `httptest.NewServer` — no external dependencies.

## Project structure

```
├── cmd/server/main.go           # Entry point: config, wiring, serve
├── internal/
│   ├── config/                  # .env → typed Config struct
│   ├── historian/               # NTLM HTTP client + OData envelope parsing
│   │   └── endpoints/           # One file per API group (tags, process values, summary)
│   ├── mcp/                     # MCP server setup + tool registration
│   │   └── tools/               # One file per MCP tool handler
│   ├── dataserver/              # Out-of-band REST server: store, HTTP, TTL/GC lifecycle
│   ├── logging/                 # slog setup
│   └── types/                   # Shared types (filter, query options, resource)
├── docs/                        # Spec, plans, AVEVA API references
├── .env.example                 # Config template (no real secrets)
└── go.mod
```

## Documentation

- `docs/spec.md` — canonical project spec (read before starting implementation work)
- `docs/adr/` — architecture decision records
- `docs/aveva-*.md` — AVEVA Historian API references
