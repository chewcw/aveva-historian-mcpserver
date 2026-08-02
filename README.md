# aveva-historian-mcpserver

MCP server (Go) that connects AI assistants to AVEVA Historian time-series data. Query tags, process values, and analog summaries through standard MCP tools, with NTLM-authenticated access to the AVEVA Historian REST API.

## How it works

Historian datasets can be far larger than a chat model's context window. This server uses a dual-response pattern:

1. **Preview** — each MCP tool returns at most 100 rows for the model to reason about, plus a `ResourceLink` URI.
2. **Full dataset** — the link points to a built-in HTTP data server that serves the complete result separately, with TTL-based cleanup.

Full payloads never enter the chat context; clients fetch them directly from the data server when needed.

## Features

- MCP stdio transport (works with Claude Desktop, Cursor, and other MCP clients)
- MCP Streamable HTTP transport (`--transport http`) with JWT HS256 auth, OAuth2-style token endpoint, and CORS
- NTLM authentication against AVEVA Historian REST API
- OData queries for tags, process values, and analog summaries
- Data server with TTL, automatic cleanup, and pin/delete resource lifecycle
- Structured logging via `log/slog`

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

The server reads configuration from environment variables. A template with all options is provided in `.env.example`.

| Variable | Required | Default | Description |
|---|---|---|---|
| `AVEVA_HISTORIAN_BASE_URL` | yes | — | Historian REST API base URL (e.g. `http://historian-server:32569`) |
| `AVEVA_HISTORIAN_USERNAME` | yes | — | NTLM username |
| `AVEVA_HISTORIAN_PASSWORD` | yes | — | NTLM password |
| `AVEVA_HISTORIAN_API_PATH_PREFIX` | no | `/Historian/v2` | API path prefix |
| `AVEVA_HISTORIAN_RESULT_BYTE_LIMIT` | no | `1048576` | Max inline result bytes before falling back to `ResourceLink` |
| `MCP_SERVER_NAME` | no | `aveva-historian-mcp` | Server name reported during MCP initialization |
| `MCP_SERVER_VERSION` | no | `dev` | Server version reported during MCP initialization |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, or `error` |
| `LOG_FILE_PATH` | no | — | Optional log file; empty logs to stderr only |
| `DATA_SERVER_BIND` | no | `127.0.0.1` | Data server bind address |
| `DATA_SERVER_PORT` | no | `8199` | Data server port |
| `DATA_SERVER_DEFAULT_TTL` | no | `5m` | Default resource lifetime (e.g. `5m`, `1h`) |
| `DATA_SERVER_GC_INTERVAL` | no | `1m` | Cleanup interval for expired resources |
| `MCP_HTTP_BIND` | no | `127.0.0.1` | MCP HTTP bind address |
| `MCP_HTTP_PORT` | no | `8200` | MCP HTTP port |
| `MCP_HTTP_JWT_SECRET` | no | — | JWT signing secret for the HTTP transport (required when using it; min 32 chars) |
| `MCP_HTTP_JWT_ISSUER` | no | `aveva-historian-mcp` | JWT `iss` claim |
| `MCP_HTTP_JWT_AUDIENCE` | no | `aveva-historian-mcp` | JWT `aud` claim |
| `MCP_CLIENTS_FILE` | no | `./clients.json` | Client credentials file for the HTTP transport |
| `MCP_CORS_ORIGINS` | no | `*` | Comma-separated allowed CORS origins |
| `MCP_CORS_ALLOW_CREDENTIALS` | no | `false` | Allow credentialed CORS requests |

## Command-line interface

The binary ships a Cobra CLI. Bare invocation (no subcommand) runs the server — existing MCP client configs keep working unchanged.

| Command | Description |
|---|---|
| `serve` | Run the MCP server (default when no subcommand is given) |
| `hashsecret` | Print a bcrypt hash of a client secret for `clients.json` |
| `version` | Print the build version |

`serve` flags override environment variables:

| Flag | Default | Description |
|---|---|---|
| `--transport` | `stdio` | MCP transport: `stdio` or `http` |
| `--bind` | `DATA_SERVER_BIND` | Data server bind address |
| `--port` | `DATA_SERVER_PORT` | Data server port |
| `--http-bind` | `MCP_HTTP_BIND` | MCP HTTP bind address |
| `--http-port` | `MCP_HTTP_PORT` | MCP HTTP port |
| `--log-level` | `LOG_LEVEL` | Log level (`debug`, `info`, `warn`, `error`) |

Precedence: environment variables first, then flags, then validation. Credentials (`AVEVA_HISTORIAN_*`) are environment-only.

`version` prints the version embedded at build time (default `dev`):

```bash
go build -ldflags "-X github.com/chewcw/aveva-historian-mcpserver/internal/cli.version=v1.2.3" -o bin/aveva-historian-mcp ./cmd/server
```

## Usage

Add the server to your MCP client configuration. For Claude Desktop (`claude_desktop_config.json`):

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

If the server runs from a working directory where `.env` is present, it loads that file automatically; otherwise set the variables in the client's `env` block.

## Tools

| Tool | Description |
|---|---|
| `get_tags` | List/search historian tags with filters: `top`, `skip`, `orderby`, `count`, `search`, `source`, and tag property filters (`tag_type`, `tag_name`, `description`, `eng_unit`, `eng_unit_min`, `eng_unit_max`, `interpolation_type`, `integral_divisor`, `rollover_value`, `message_on`, `message_off`, `filters`) |
| `read_process_values` | Time-series process values for a tag over a time range. Optional `retrieval_mode` (Average, Cyclic, Integral, Minimum, Maximum, BestFit, Delta, Interpolated, Slope, Counter, Full) and `resolution_ms` |
| `read_analog_summary` | Analog summary statistics per cycle over a time range: Min, Max, Avg, StdDev, Integral, Count, First, Last, OPCQuality, PercentGood, with timestamps |

Large results return a `ResourceLink` (up to 100 preview rows inline); fetch the link from the data server for the full dataset.

## Data server

Embedded HTTP server serving resource payloads.

| Method | Path | Description |
|---|---|---|
| `GET` | `/resources` | List resources |
| `GET` | `/resources/{id}` | Fetch a resource payload |
| `POST` | `/resources/{id}/pin` | Keep a resource alive (prevent expiration) |
| `DELETE` | `/resources/{id}` | Delete a resource immediately |

Expired resources are removed by the background cleanup loop (`DATA_SERVER_GC_INTERVAL`).

## HTTP transport

For remote or browser-based clients, run MCP over a Streamable HTTP endpoint with JWT authentication instead of stdio:

```bash
./bin/aveva-historian-mcp serve --transport http
```

This starts an HTTP server (default `127.0.0.1:8200`) with two endpoints:

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/mcp/token` | client credentials | Mint a JWT access token (OAuth2-style client-credentials exchange) |
| `POST`/`GET` | `/mcp` | `Authorization: Bearer <jwt>` | Streamable HTTP MCP endpoint (JSON-RPC messages + SSE streams) |

### Client credentials

Clients are defined in `MCP_CLIENTS_FILE` (`clients.json` by default). Secrets are stored as bcrypt hashes — never plaintext. Generate a hash with the `hashsecret` command:

```bash
./bin/aveva-historian-mcp hashsecret my-client-secret
```

Example `clients.json`:

```json
{
  "clients": [
    {
      "client_id": "web",
      "client_secret_hash": "$2a$10$...",
      "scopes": ["read"],
      "enabled": true
    }
  ]
}
```

Only clients with `"enabled": true` can authenticate. Malformed entries (invalid bcrypt hash, duplicate `client_id`, missing file) are hard errors at startup.

### Getting a token

The token endpoint accepts HTTP Basic auth or form-encoded credentials and returns a JWT valid for 1 hour:

```bash
# Basic auth
curl -u web:my-client-secret -X POST http://127.0.0.1:8200/mcp/token

# Form-encoded
curl -X POST http://127.0.0.1:8200/mcp/token \
  -d 'client_id=web&client_secret=my-client-secret'
```

Response:

```json
{"access_token":"eyJ...","token_type":"Bearer","expires_in":3600}
```

Call the MCP endpoint with the token:

```bash
curl http://127.0.0.1:8200/mcp \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
```

### Configuration notes

- `MCP_HTTP_JWT_SECRET` is required for the HTTP transport and must be at least 32 characters. Generate one with `openssl rand -hex 32`.
- `MCP_CORS_ORIGINS` defaults to `*`. With `MCP_CORS_ALLOW_CREDENTIALS=true` the wildcard is rejected at startup — credentials require an explicit origin list.
- The data server (port 8199) is unchanged; `ResourceLink` payloads work the same over HTTP.
- `--transport http` requires `MCP_HTTP_JWT_SECRET` and a valid `clients.json`; both are validated before the listener starts.

## Development

```bash
# Run tests
go test ./... -count=1

# Lint
go vet ./...

# Run (stdio)
go run ./cmd/server

# Check version
go run ./cmd/server version
```

## Project structure

```
├── cmd/server/main.go           # Entry point: thin wrapper around internal/cli
├── internal/
│   ├── cli/                     # Cobra commands: serve, version, hashsecret, flag handling
│   ├── config/                  # Environment → typed config
│   ├── historian/               # NTLM HTTP client + OData response parsing
│   │   └── endpoints/           # One file per API group (tags, process values, summary)
│   ├── mcp/                     # MCP server setup + tool registration
│   │   └── tools/               # One file per MCP tool handler
│   ├── mcphttp/                 # HTTP transport: JWT auth, token endpoint, CORS, server
│   ├── dataserver/              # Out-of-band REST server: store, HTTP, TTL/cleanup
│   ├── logging/                 # slog setup
│   └── types/                   # Shared types (filter, query options, resource)
├── .env.example                 # Configuration template
└── go.mod
```
