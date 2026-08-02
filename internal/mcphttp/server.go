package mcphttp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chewcw/aveva-historian-mcpserver/internal/config"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// minJWTSecretLen guards against trivially guessable signing keys.
const minJWTSecretLen = 32

// ValidateConfig checks MCP HTTP settings that are only required when the
// http transport is selected. Returns nil when valid.
func ValidateConfig(cfg *config.Config) error {
	if len(cfg.MCPHTTPJWTSecret) < minJWTSecretLen {
		return fmt.Errorf("MCP_HTTP_JWT_SECRET must be at least %d chars for http transport", minJWTSecretLen)
	}
	// Browsers reject a wildcard Allow-Origin paired with credentials; it also
	// defeats the origin allowlist. Refuse the combination at config time.
	if cfg.MCPCORSAllowCreds && strings.TrimSpace(cfg.MCPCORSOrigins) == "*" {
		return fmt.Errorf("MCP_CORS_ALLOW_CREDENTIALS cannot be true when MCP_CORS_ORIGINS is \"*\"")
	}
	// Fail fast: a missing clients file would only surface after the listener
	// is up. Check existence here so the operator sees the error at startup.
	if _, err := os.Stat(cfg.MCPClientsFile); err != nil {
		return fmt.Errorf("MCP_CLIENTS_FILE %s: %w", cfg.MCPClientsFile, err)
	}
	return nil
}

// Start begins the MCP Streamable HTTP server. Blocks until ctx is cancelled
// or the server fails. Callers should run it as a goroutine.
func Start(ctx context.Context, cfg *config.Config, server *sdkmcp.Server, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("source", "mcphttp.Server")

	clients, err := LoadClients(cfg.MCPClientsFile)
	if err != nil {
		return fmt.Errorf("mcp http: %w", err)
	}

	origins := splitOrigins(cfg.MCPCORSOrigins)
	cors := func(next http.Handler) http.Handler {
		return corsMiddleware(origins, cfg.MCPCORSAllowCreds, next)
	}

	sdkHandler := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server {
		return server
	}, &sdkmcp.StreamableHTTPOptions{Logger: logger})

	tok := newTokenHandler(clients, cfg.MCPHTTPJWTSecret, cfg.MCPHTTPJWTIssuer, cfg.MCPHTTPJWTAudience)

	mux := http.NewServeMux()
	mux.Handle("POST /mcp/token", cors(tok))
	mux.Handle("/mcp", cors(requireAuth(cfg.MCPHTTPJWTIssuer, cfg.MCPHTTPJWTAudience, cfg.MCPHTTPJWTSecret, sdkHandler)))

	addr := net.JoinHostPort(cfg.MCPHTTPBind, strconv.Itoa(cfg.MCPHTTPPort))
	hs := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		hs.Shutdown(shutdownCtx)
	}()

	logger.Info("MCP HTTP server starting", "addr", addr)
	if err := hs.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("mcp http server: %w", err)
	}
	return nil
}
