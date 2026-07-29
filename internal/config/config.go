package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	APIPathPrefix string
	BaseURL       string
	Username      string
	Password      string
	ServerName    string
	ServerVersion string
	LogLevel        string
	LogFilePath      string
	ResultByteLimit int

	DataServerBind       string        // default "127.0.0.1"
	DataServerPort       int           // default 8199
	DataServerDefaultTTL time.Duration // default 5m
	DataServerGCInterval time.Duration // default 1m
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return loadFromEnv()
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}
	return loadFromEnv()
}

func loadFromEnv() (*Config, error) {
	cfg := &Config{
		ServerName:    defaultStr(os.Getenv("MCP_SERVER_NAME"), "aveva-historian-mcp"),
		ServerVersion: defaultStr(os.Getenv("MCP_SERVER_VERSION"), "dev"),
		LogLevel:      defaultStr(os.Getenv("LOG_LEVEL"), "info"),
		LogFilePath:   os.Getenv("LOG_FILE_PATH"),
	}
	cfg.APIPathPrefix = defaultStr(os.Getenv("AVEVA_HISTORIAN_API_PATH_PREFIX"), "/Historian/v2")
	cfg.BaseURL = os.Getenv("AVEVA_HISTORIAN_BASE_URL")
	cfg.Username = os.Getenv("AVEVA_HISTORIAN_USERNAME")
	cfg.Password = os.Getenv("AVEVA_HISTORIAN_PASSWORD")

	const defaultResultByteLimit = 1_048_576
	raw := os.Getenv("AVEVA_HISTORIAN_RESULT_BYTE_LIMIT")
	if raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			cfg.ResultByteLimit = v
		} else {
			cfg.ResultByteLimit = defaultResultByteLimit
		}
	} else {
		cfg.ResultByteLimit = defaultResultByteLimit
	}

	cfg.DataServerBind = defaultStr(os.Getenv("DATA_SERVER_BIND"), "127.0.0.1")
	cfg.DataServerPort = defaultInt(os.Getenv("DATA_SERVER_PORT"), 8199)
	cfg.DataServerDefaultTTL = defaultDuration(os.Getenv("DATA_SERVER_DEFAULT_TTL"), 5*time.Minute)
	cfg.DataServerGCInterval = defaultDuration(os.Getenv("DATA_SERVER_GC_INTERVAL"), 1*time.Minute)

	var missing []string
	if cfg.BaseURL == "" {
		missing = append(missing, "AVEVA_HISTORIAN_BASE_URL")
	}
	if cfg.Username == "" {
		missing = append(missing, "AVEVA_HISTORIAN_USERNAME")
	}
	if cfg.Password == "" {
		missing = append(missing, "AVEVA_HISTORIAN_PASSWORD")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}

	if _, err := url.Parse(cfg.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid AVEVA_HISTORIAN_BASE_URL %q: %w", cfg.BaseURL, err)
	}
	return cfg, nil
}

func defaultStr(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

func defaultInt(s string, def int) int {
	if s == "" {
		return def
	}
	if v, err := strconv.Atoi(s); err == nil && v > 0 {
		return v
	}
	return def
}

func defaultDuration(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	if v, err := time.ParseDuration(s); err == nil && v > 0 {
		return v
	}
	return def
}
