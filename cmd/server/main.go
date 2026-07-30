package main

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

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	logger := logging.New(cfg.LogLevel, cfg.LogFilePath)
	client := historian.New(cfg.BaseURL, cfg.Username, cfg.Password, cfg.APIPathPrefix, logger)

	// Start data server
	store := dataserver.NewStore(cfg.DataServerDefaultTTL)
	dataServerCtx, stopDataServer := context.WithCancel(context.Background())
	defer stopDataServer()
	go func() {
		if err := dataserver.Start(dataServerCtx, store, cfg.DataServerBind, cfg.DataServerPort, logger); err != nil {
			logger.Warn("data server exited", "error", err)
		}
	}()
	dataserver.RunGC(dataServerCtx, store, cfg.DataServerGCInterval, logger)

	server := mcpserver.NewServer(cfg, client, store, logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	logger.Info("starting server", "name", cfg.ServerName, "version", cfg.ServerVersion)
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
