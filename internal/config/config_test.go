package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// envCleanup unsets all known env vars so each test starts from a clean slate.
func envCleanup() {
	os.Unsetenv("AVEVA_HISTORIAN_BASE_URL")
	os.Unsetenv("AVEVA_HISTORIAN_USERNAME")
	os.Unsetenv("AVEVA_HISTORIAN_PASSWORD")
	os.Unsetenv("MCP_SERVER_NAME")
	os.Unsetenv("MCP_SERVER_VERSION")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("LOG_FILE_PATH")
	os.Unsetenv("DATA_SERVER_BIND")
	os.Unsetenv("DATA_SERVER_PORT")
	os.Unsetenv("DATA_SERVER_DEFAULT_TTL")
	os.Unsetenv("DATA_SERVER_GC_INTERVAL")
	os.Unsetenv("MCP_HTTP_BIND")
	os.Unsetenv("MCP_HTTP_PORT")
	os.Unsetenv("MCP_HTTP_JWT_SECRET")
	os.Unsetenv("MCP_HTTP_JWT_ISSUER")
	os.Unsetenv("MCP_HTTP_JWT_AUDIENCE")
	os.Unsetenv("MCP_CLIENTS_FILE")
	os.Unsetenv("MCP_CORS_ORIGINS")
	os.Unsetenv("MCP_CORS_ALLOW_CREDENTIALS")
}

func TestLoad_Success(t *testing.T) {
	envCleanup()
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	if err := os.WriteFile(env, []byte(
		"AVEVA_HISTORIAN_BASE_URL=http://historian:32569\n"+
			"AVEVA_HISTORIAN_USERNAME=admin\n"+
			"AVEVA_HISTORIAN_PASSWORD=secret\n"+
			"LOG_LEVEL=debug\n"+
			"LOG_FILE_PATH=test.log\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(env)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.BaseURL != "http://historian:32569" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "http://historian:32569")
	}
	if cfg.Username != "admin" {
		t.Errorf("Username = %q, want %q", cfg.Username, "admin")
	}
	if cfg.Password != "secret" {
		t.Errorf("Password = %q, want %q", cfg.Password, "secret")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.LogFilePath != "test.log" {
		t.Errorf("LogFilePath = %q, want %q", cfg.LogFilePath, "test.log")
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{"missing BASE_URL", "AVEVA_HISTORIAN_USERNAME=u\nAVEVA_HISTORIAN_PASSWORD=p\n"},
		{"missing USERNAME", "AVEVA_HISTORIAN_BASE_URL=http://h:1\nAVEVA_HISTORIAN_PASSWORD=p\n"},
		{"missing PASSWORD", "AVEVA_HISTORIAN_BASE_URL=http://h:1\nAVEVA_HISTORIAN_USERNAME=u\n"},
		{"all missing", "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envCleanup()
			dir := t.TempDir()
			env := filepath.Join(dir, ".env")
			if err := os.WriteFile(env, []byte(tt.env), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(env)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestLoad_InvalidURL(t *testing.T) {
	envCleanup()
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	if err := os.WriteFile(env, []byte(
		"AVEVA_HISTORIAN_BASE_URL=://invalid\n"+
			"AVEVA_HISTORIAN_USERNAME=u\n"+
			"AVEVA_HISTORIAN_PASSWORD=p\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(env)
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestLoad_DefaultsApplied(t *testing.T) {
	envCleanup()
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	if err := os.WriteFile(env, []byte(
		"AVEVA_HISTORIAN_BASE_URL=http://h:32569\n"+
			"AVEVA_HISTORIAN_USERNAME=u\n"+
			"AVEVA_HISTORIAN_PASSWORD=p\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(env)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ServerName != "aveva-historian-mcp" {
		t.Errorf("ServerName = %q, want %q", cfg.ServerName, "aveva-historian-mcp")
	}
	if cfg.ServerVersion != "dev" {
		t.Errorf("ServerVersion = %q, want %q", cfg.ServerVersion, "dev")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
}

func TestConfigDataServerDefaults(t *testing.T) {
	t.Setenv("AVEVA_HISTORIAN_BASE_URL", "http://localhost")
	t.Setenv("AVEVA_HISTORIAN_USERNAME", "user")
	t.Setenv("AVEVA_HISTORIAN_PASSWORD", "pass")

	cfg, err := Load(".env.nonexistent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataServerBind != "127.0.0.1" {
		t.Errorf("DataServerBind = %q, want 127.0.0.1", cfg.DataServerBind)
	}
	if cfg.DataServerPort != 8199 {
		t.Errorf("DataServerPort = %d, want 8199", cfg.DataServerPort)
	}
	if cfg.DataServerDefaultTTL != 5*time.Minute {
		t.Errorf("DataServerDefaultTTL = %v, want 5m0s", cfg.DataServerDefaultTTL)
	}
	if cfg.DataServerGCInterval != 1*time.Minute {
		t.Errorf("DataServerGCInterval = %v, want 1m0s", cfg.DataServerGCInterval)
	}
}

func TestConfigDataServerOverrides(t *testing.T) {
	t.Setenv("AVEVA_HISTORIAN_BASE_URL", "http://localhost")
	t.Setenv("AVEVA_HISTORIAN_USERNAME", "user")
	t.Setenv("AVEVA_HISTORIAN_PASSWORD", "pass")
	t.Setenv("DATA_SERVER_BIND", "0.0.0.0")
	t.Setenv("DATA_SERVER_PORT", "9090")
	t.Setenv("DATA_SERVER_DEFAULT_TTL", "10m")
	t.Setenv("DATA_SERVER_GC_INTERVAL", "30s")

	cfg, err := Load(".env.nonexistent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataServerBind != "0.0.0.0" {
		t.Errorf("DataServerBind = %q, want 0.0.0.0", cfg.DataServerBind)
	}
	if cfg.DataServerPort != 9090 {
		t.Errorf("DataServerPort = %d, want 9090", cfg.DataServerPort)
	}
	if cfg.DataServerDefaultTTL != 10*time.Minute {
		t.Errorf("DataServerDefaultTTL = %v, want 10m0s", cfg.DataServerDefaultTTL)
	}
	if cfg.DataServerGCInterval != 30*time.Second {
		t.Errorf("DataServerGCInterval = %v, want 30s", cfg.DataServerGCInterval)
	}
}

func TestConfigMCPHTTPDefaults(t *testing.T) {
	envCleanup()
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	if err := os.WriteFile(env, []byte(
		"AVEVA_HISTORIAN_BASE_URL=http://historian:32569\n"+
			"AVEVA_HISTORIAN_USERNAME=admin\n"+
			"AVEVA_HISTORIAN_PASSWORD=secret\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(env)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.MCPHTTPBind != "127.0.0.1" {
		t.Errorf("MCPHTTPBind = %q, want %q", cfg.MCPHTTPBind, "127.0.0.1")
	}
	if cfg.MCPHTTPPort != 8200 {
		t.Errorf("MCPHTTPPort = %d, want 8200", cfg.MCPHTTPPort)
	}
	if cfg.MCPHTTPJWTIssuer != "aveva-historian-mcp" {
		t.Errorf("MCPHTTPJWTIssuer = %q, want %q", cfg.MCPHTTPJWTIssuer, "aveva-historian-mcp")
	}
	if cfg.MCPHTTPJWTAudience != "aveva-historian-mcp" {
		t.Errorf("MCPHTTPJWTAudience = %q, want %q", cfg.MCPHTTPJWTAudience, "aveva-historian-mcp")
	}
	if cfg.MCPClientsFile != "./clients.json" {
		t.Errorf("MCPClientsFile = %q, want %q", cfg.MCPClientsFile, "./clients.json")
	}
	if cfg.MCPCORSOrigins != "*" {
		t.Errorf("MCPCORSOrigins = %q, want %q", cfg.MCPCORSOrigins, "*")
	}
	if cfg.MCPCORSAllowCreds {
		t.Error("MCPCORSAllowCreds = true, want false")
	}
	if cfg.MCPHTTPJWTSecret != "" {
		t.Errorf("MCPHTTPJWTSecret = %q, want empty", cfg.MCPHTTPJWTSecret)
	}
}

func TestConfigMCPHTTPOverrides(t *testing.T) {
	envCleanup()
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	if err := os.WriteFile(env, []byte(
		"AVEVA_HISTORIAN_BASE_URL=http://historian:32569\n"+
			"AVEVA_HISTORIAN_USERNAME=admin\n"+
			"AVEVA_HISTORIAN_PASSWORD=secret\n"+
			"MCP_HTTP_BIND=0.0.0.0\n"+
			"MCP_HTTP_PORT=9999\n"+
			"MCP_HTTP_JWT_SECRET=this-is-a-32+char-secret-1234567890\n"+
			"MCP_HTTP_JWT_ISSUER=my-issuer\n"+
			"MCP_HTTP_JWT_AUDIENCE=my-audience\n"+
			"MCP_CLIENTS_FILE=/etc/mcp/clients.json\n"+
			"MCP_CORS_ORIGINS=https://a.example.com,https://b.example.com\n"+
			"MCP_CORS_ALLOW_CREDENTIALS=true\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(env)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.MCPHTTPBind != "0.0.0.0" {
		t.Errorf("MCPHTTPBind = %q, want %q", cfg.MCPHTTPBind, "0.0.0.0")
	}
	if cfg.MCPHTTPPort != 9999 {
		t.Errorf("MCPHTTPPort = %d, want 9999", cfg.MCPHTTPPort)
	}
	if cfg.MCPHTTPJWTSecret != "this-is-a-32+char-secret-1234567890" {
		t.Errorf("MCPHTTPJWTSecret = %q, want set value", cfg.MCPHTTPJWTSecret)
	}
	if cfg.MCPHTTPJWTIssuer != "my-issuer" {
		t.Errorf("MCPHTTPJWTIssuer = %q, want %q", cfg.MCPHTTPJWTIssuer, "my-issuer")
	}
	if cfg.MCPHTTPJWTAudience != "my-audience" {
		t.Errorf("MCPHTTPJWTAudience = %q, want %q", cfg.MCPHTTPJWTAudience, "my-audience")
	}
	if cfg.MCPClientsFile != "/etc/mcp/clients.json" {
		t.Errorf("MCPClientsFile = %q, want %q", cfg.MCPClientsFile, "/etc/mcp/clients.json")
	}
	if cfg.MCPCORSOrigins != "https://a.example.com,https://b.example.com" {
		t.Errorf("MCPCORSOrigins = %q, want two origins", cfg.MCPCORSOrigins)
	}
	if !cfg.MCPCORSAllowCreds {
		t.Error("MCPCORSAllowCreds = false, want true")
	}
}
