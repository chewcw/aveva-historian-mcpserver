package config

import (
	"os"
	"path/filepath"
	"testing"
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
