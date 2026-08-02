package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/chewcw/aveva-historian-mcpserver/internal/config"
	"github.com/chewcw/aveva-historian-mcpserver/internal/dataserver"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/logging"
	mcpserver "github.com/chewcw/aveva-historian-mcpserver/internal/mcp"
	mcphttp "github.com/chewcw/aveva-historian-mcpserver/internal/mcphttp"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// serveOptions holds flag values for the serve command. Zero values mean
// "not set" and leave the corresponding config field untouched.
type serveOptions struct {
	Transport string
	Bind      string
	Port      int
	HTTPBind  string
	HTTPPort  int
	LogLevel  string
}

func newServeCmd() *cobra.Command {
	var opts serveOptions
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the MCP server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.Transport, "transport", "stdio", "Supported MCP transport: stdio or http")
	cmd.Flags().StringVar(&opts.Bind, "bind", "", "data server bind address (overrides DATA_SERVER_BIND)")
	cmd.Flags().IntVar(&opts.Port, "port", 0, "data server port (overrides DATA_SERVER_PORT)")
	cmd.Flags().StringVar(&opts.HTTPBind, "http-bind", "", "MCP HTTP bind address (overrides MCP_HTTP_BIND)")
	cmd.Flags().IntVar(&opts.HTTPPort, "http-port", 0, "MCP HTTP port (overrides MCP_HTTP_PORT)")
	cmd.Flags().StringVar(&opts.LogLevel, "log-level", "", "log level (overrides LOG_LEVEL)")
	return cmd
}

// runServe validates transport, loads env config, applies flag overrides,
// then starts the data server and MCP server. This is the body of the old
// main() moved here so both bare invocation and `serve` share it.
func runServe(ctx context.Context, opts serveOptions) error {
	transport := opts.Transport
	if transport == "" {
		transport = "stdio"
	}
	switch transport {
	case "stdio":
	case "http":
	default:
		return fmt.Errorf("invalid transport %q (want stdio or http)", transport)
	}

	cfg, err := loadConfig(opts)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	logger := logging.New(cfg.LogLevel, cfg.LogFilePath)
	client := historian.New(cfg.BaseURL, cfg.Username, cfg.Password, cfg.APIPathPrefix, logger)

	store := dataserver.NewStore(cfg.DataServerDefaultTTL)
	dataServerCtx, stopDataServer := context.WithCancel(context.Background())
	defer stopDataServer()
	go func() {
		if err := dataserver.Start(dataServerCtx, store, cfg.DataServerBind, cfg.DataServerPort, logger); err != nil {
			logger.Warn("data server exited", "error", err)
		}
	}()
	dataserver.RunGC(dataServerCtx, store, cfg.DataServerGCInterval, logger)

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	logger.Info("starting server", "name", cfg.ServerName, "version", cfg.ServerVersion)
	if transport == "http" {
		if err := mcphttp.ValidateConfig(cfg); err != nil {
			return err
		}
		server := mcpserver.NewServer(cfg, client, store, logger)
		if err := mcphttp.Start(ctx, cfg, server, logger); err != nil {
			return fmt.Errorf("mcp http server: %w", err)
		}
		return nil
	}
	server := mcpserver.NewServer(cfg, client, store, logger)
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	return nil
}

// loadConfig loads env config then applies flag overrides. Env wins over
// defaults, flags win over env, validation happens inside config.Load.
func loadConfig(opts serveOptions) (*config.Config, error) {
	cfg, err := config.Load(".env")
	if err != nil {
		return nil, err
	}
	applyFlagOverrides(cfg, opts)
	return cfg, nil
}

// applyFlagOverrides overwrites cfg fields with non-zero flag values.
// Pure function: unit-testable without the cobra harness.
func applyFlagOverrides(cfg *config.Config, opts serveOptions) {
	if opts.Bind != "" {
		cfg.DataServerBind = opts.Bind
	}
	if opts.Port != 0 {
		cfg.DataServerPort = opts.Port
	}
	if opts.HTTPBind != "" {
		cfg.MCPHTTPBind = opts.HTTPBind
	}
	if opts.HTTPPort != 0 {
		cfg.MCPHTTPPort = opts.HTTPPort
	}
	if opts.LogLevel != "" {
		cfg.LogLevel = opts.LogLevel
	}
}
