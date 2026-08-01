package cli

import (
	"strings"
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/config"
)

func TestTransportHTTPRejected(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"serve", "--transport", "http"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for http transport")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Fatalf("expected not-yet-implemented error, got: %v", err)
	}
}

func TestTransportInvalidRejected(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"serve", "--transport", "tcp"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid transport")
	}
	if !strings.Contains(err.Error(), "invalid transport") {
		t.Fatalf("expected invalid transport error, got: %v", err)
	}
}

func TestTransportStdioPassesGate(t *testing.T) {
	clearRequiredEnv(t)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"serve", "--transport", "stdio"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error: stdio transport should proceed to config load")
	}
	// Reaching the config error proves the stdio value passed the transport gate.
	if !strings.Contains(err.Error(), "missing required config") {
		t.Fatalf("expected config error, got: %v", err)
	}
}

func TestApplyFlagOverrides(t *testing.T) {
	cfg := &config.Config{DataServerPort: 9999, DataServerBind: "0.0.0.0", LogLevel: "info"}
	applyFlagOverrides(cfg, serveOptions{Port: 8080, LogLevel: "debug"})
	if cfg.DataServerPort != 8080 {
		t.Errorf("DataServerPort = %d, want 8080", cfg.DataServerPort)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.DataServerBind != "0.0.0.0" {
		t.Errorf("DataServerBind = %q, want untouched 0.0.0.0", cfg.DataServerBind)
	}
}

func TestApplyFlagOverridesZeroValuesNoop(t *testing.T) {
	cfg := &config.Config{DataServerPort: 9999, DataServerBind: "0.0.0.0", LogLevel: "info"}
	applyFlagOverrides(cfg, serveOptions{})
	if cfg.DataServerPort != 9999 || cfg.DataServerBind != "0.0.0.0" || cfg.LogLevel != "info" {
		t.Fatalf("zero-value options changed config: %+v", cfg)
	}
}

func TestServeFlagParsing(t *testing.T) {
	clearRequiredEnv(t)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"serve", "--bind", "0.0.0.0", "--port", "9000", "--log-level", "debug"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from config load with no credentials")
	}
	// The flags must not be rejected by parsing — the error must come from config.
	if !strings.Contains(err.Error(), "missing required config") {
		t.Fatalf("expected config error after successful flag parse, got: %v", err)
	}
}

func TestFlagOverridesEnv(t *testing.T) {
	t.Setenv("AVEVA_HISTORIAN_BASE_URL", "http://historian.example")
	t.Setenv("AVEVA_HISTORIAN_USERNAME", "testuser")
	t.Setenv("AVEVA_HISTORIAN_PASSWORD", "testpass")
	t.Setenv("DATA_SERVER_PORT", "9999")
	t.Setenv("DATA_SERVER_BIND", "10.0.0.1")

	cfg, err := loadConfig(serveOptions{Port: 8080})
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.DataServerPort != 8080 {
		t.Errorf("DataServerPort = %d, want 8080 (flag wins over env)", cfg.DataServerPort)
	}
	if cfg.DataServerBind != "10.0.0.1" {
		t.Errorf("DataServerBind = %q, want 10.0.0.1 (env default preserved when flag unset)", cfg.DataServerBind)
	}
}
