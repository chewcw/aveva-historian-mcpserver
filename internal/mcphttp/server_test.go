package mcphttp

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chewcw/aveva-historian-mcpserver/internal/config"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func writeValidClientsFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "clients.json")
	if err := os.WriteFile(path, []byte(`{"clients":[
		{"client_id":"web","client_secret_hash":"`+testClientHash+`","scopes":["read"],"enabled":true}
	]}`), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidateConfigSecretTooShort(t *testing.T) {
	cfg := &config.Config{MCPHTTPJWTSecret: "short"}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected error for short secret")
	}
}

func TestValidateConfigOK(t *testing.T) {
	cfg := &config.Config{MCPHTTPJWTSecret: testSecret}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}
}

func TestStartRejectsBadClientsFile(t *testing.T) {
	cfg := &config.Config{
		MCPHTTPJWTSecret:   testSecret,
		MCPHTTPJWTIssuer:   testIssuer,
		MCPHTTPJWTAudience: testAudience,
		MCPClientsFile:     filepath.Join(t.TempDir(), "missing.json"),
	}
	srv := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "test"}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := Start(ctx, cfg, srv, nil)
	if err == nil {
		t.Fatal("expected error for missing clients file")
	}
	if !strings.Contains(err.Error(), "clients") {
		t.Errorf("error = %q, want clients file mention", err.Error())
	}
}

// TestStartServesTokenAndRejectsUnauthenticatedMCP exercises the full mux:
// token endpoint works, /mcp rejects without a token, and accepts with one.
func TestStartServesTokenAndRejectsUnauthenticatedMCP(t *testing.T) {
	// Pick a free port.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	cfg := &config.Config{
		MCPHTTPBind:        "127.0.0.1",
		MCPHTTPPort:        port,
		MCPHTTPJWTSecret:   testSecret,
		MCPHTTPJWTIssuer:   testIssuer,
		MCPHTTPJWTAudience: testAudience,
		MCPClientsFile:     writeValidClientsFile(t),
		MCPCORSOrigins:     "*",
	}
	srv := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "test"}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	var startErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		startErr = Start(ctx, cfg, srv, nil)
	}()

	base := "http://127.0.0.1:" + strconv.Itoa(port)
	// Wait for readiness.
	var healthy bool
	for i := 0; i < 100; i++ {
		resp, err := http.Get(base + "/mcp/token")
		if err == nil {
			resp.Body.Close()
			healthy = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !healthy {
		cancel()
		wg.Wait()
		t.Fatalf("server did not become ready; startErr=%v", startErr)
	}

	// Unauthenticated /mcp POST must be rejected.
	req, _ := http.NewRequest(http.MethodPost, base+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated /mcp status = %d, want 401", resp.StatusCode)
	}

	// Token endpoint issues a JWT.
	tokReq, _ := http.NewRequest(http.MethodPost, base+"/mcp/token", strings.NewReader(""))
	tokReq.SetBasicAuth("web", "s3cret")
	tokResp, err := http.DefaultClient.Do(tokReq)
	if err != nil {
		t.Fatal(err)
	}
	defer tokResp.Body.Close()
	if tokResp.StatusCode != http.StatusOK {
		t.Fatalf("token status = %d, want 200", tokResp.StatusCode)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokResp.Body).Decode(&tok); err != nil {
		t.Fatal(err)
	}

	// Authenticated /mcp POST reaches the SDK handler (404 = session-less
	// streamable handler response for a POST without Mcp-Session-Id on a
	// stateless-capable handler; anything other than 401 proves auth passed).
	authReq, _ := http.NewRequest(http.MethodPost, base+"/mcp", strings.NewReader(`{}`))
	authReq.Header.Set("Content-Type", "application/json")
	authReq.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	authResp, err := http.DefaultClient.Do(authReq)
	if err != nil {
		t.Fatal(err)
	}
	authResp.Body.Close()
	if authResp.StatusCode == http.StatusUnauthorized {
		t.Fatal("authenticated /mcp request got 401")
	}

	cancel()
	wg.Wait()
}
