package mcphttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	testSecret   = "0123456789abcdef0123456789abcdef"
	testIssuer   = "test-issuer"
	testAudience = "test-audience"
)

func testClient() Client {
	return Client{ClientID: "web", Scopes: []string{"read"}, Enabled: true}
}

func TestIssueAndValidateToken(t *testing.T) {
	c := testClient()
	token, err := IssueToken(c, testSecret, testIssuer, testAudience, time.Hour)
	if err != nil {
		t.Fatalf("IssueToken() error = %v", err)
	}
	claims, err := validateToken(token, testSecret, testIssuer, testAudience)
	if err != nil {
		t.Fatalf("validateToken() error = %v", err)
	}
	if claims.Subject != "web" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "web")
	}
	if len(claims.Scopes) != 1 || claims.Scopes[0] != "read" {
		t.Errorf("Scopes = %v, want [read]", claims.Scopes)
	}
	if claims.Issuer != testIssuer {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, testIssuer)
	}
}

func TestValidateTokenRejectsWrongSecret(t *testing.T) {
	c := testClient()
	token, err := IssueToken(c, testSecret, testIssuer, testAudience, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateToken(token, "another-secret-0123456789abcdef", testIssuer, testAudience); err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestValidateTokenRejectsWrongIssuer(t *testing.T) {
	c := testClient()
	token, err := IssueToken(c, testSecret, testIssuer, testAudience, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateToken(token, testSecret, "other-issuer", testAudience); err == nil {
		t.Fatal("expected error for wrong issuer")
	}
}

func TestValidateTokenRejectsExpired(t *testing.T) {
	c := testClient()
	token, err := IssueToken(c, testSecret, testIssuer, testAudience, -time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateToken(token, testSecret, testIssuer, testAudience); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestRequireAuthAcceptsValidToken(t *testing.T) {
	c := testClient()
	token, err := IssueToken(c, testSecret, testIssuer, testAudience, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	handler := requireAuth(testIssuer, testAudience, testSecret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(claimsCtxKey{}).(*Claims)
		if !ok {
			http.Error(w, "no claims", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(claims.Subject))
	}))
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "web" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "web")
	}
}

func TestRequireAuthRejectsMissingToken(t *testing.T) {
	handler := requireAuth(testIssuer, testAudience, testSecret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestRequireAuthRejectsGarbageToken(t *testing.T) {
	handler := requireAuth(testIssuer, testAudience, testSecret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer not.a.jwt")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
