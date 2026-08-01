package cli

import (
	"bytes"
	"strings"
	"testing"
)

// clearRequiredEnv forces the three required credentials to empty so
// config.Load falls through to the "missing required config" error
// regardless of the host environment.
func clearRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AVEVA_HISTORIAN_BASE_URL", "")
	t.Setenv("AVEVA_HISTORIAN_USERNAME", "")
	t.Setenv("AVEVA_HISTORIAN_PASSWORD", "")
}

func TestRootHelpListsCommands(t *testing.T) {
	var out bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, want := range []string{"serve", "version"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("help output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRootBareRunsServe(t *testing.T) {
	clearRequiredEnv(t)
	cmd := NewRootCmd()
	cmd.SetArgs(nil)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from bare serve run with no credentials")
	}
	if !strings.Contains(err.Error(), "missing required config") {
		t.Fatalf("expected config error, got: %v", err)
	}
}

func TestVersionCommand(t *testing.T) {
	original := version
	version = "test-version"
	defer func() { version = original }()

	var out bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "test-version" {
		t.Fatalf("version output = %q, want %q", got, "test-version")
	}
}

func TestUnknownCommand(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"nope"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestExtraArgsRejected(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"serve", "x"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for extra positional args")
	}
}
