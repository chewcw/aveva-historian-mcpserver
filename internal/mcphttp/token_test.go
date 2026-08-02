package mcphttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func newTestTokenHandler() *tokenHandler {
	c := testClient()
	return newTokenHandler(map[string]Client{"web": c}, testSecret, testIssuer, testAudience)
}

func TestTokenEndpointBasicAuth(t *testing.T) {
	h := newTestTokenHandler()
	req := httptest.NewRequest(http.MethodPost, "/mcp/token", strings.NewReader(""))
	req.SetBasicAuth("web", "s3cret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	var resp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad JSON response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatal("access_token empty")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("token_type = %q, want Bearer", resp.TokenType)
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("expires_in = %d, want 3600", resp.ExpiresIn)
	}
	claims := &Claims{}
	if _, err := jwt.ParseWithClaims(resp.AccessToken, claims,
		func(t *jwt.Token) (any, error) { return []byte(testSecret), nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(testIssuer),
		jwt.WithAudience(testAudience),
		jwt.WithExpirationRequired(),
	); err != nil {
		t.Fatalf("returned token invalid: %v", err)
	}
	if claims.Subject != "web" {
		t.Errorf("Subject = %q, want web", claims.Subject)
	}
}

func TestTokenEndpointFormAuth(t *testing.T) {
	h := newTestTokenHandler()
	form := "client_id=web&client_secret=s3cret"
	req := httptest.NewRequest(http.MethodPost, "/mcp/token", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestTokenEndpointBadSecret(t *testing.T) {
	h := newTestTokenHandler()
	req := httptest.NewRequest(http.MethodPost, "/mcp/token", strings.NewReader(""))
	req.SetBasicAuth("web", "wrong")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid_client") {
		t.Errorf("body = %q, want invalid_client error", rec.Body.String())
	}
}

func TestTokenEndpointUnknownClient(t *testing.T) {
	h := newTestTokenHandler()
	req := httptest.NewRequest(http.MethodPost, "/mcp/token", strings.NewReader(""))
	req.SetBasicAuth("ghost", "whatever")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestTokenEndpointWrongMethod(t *testing.T) {
	h := newTestTokenHandler()
	req := httptest.NewRequest(http.MethodGet, "/mcp/token", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}
