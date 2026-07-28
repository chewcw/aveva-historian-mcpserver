package endpoints

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
)

// MockServer wraps an httptest.Server that stubs an AVEVA Historian endpoint.
// It captures the outbound query string for test assertions.
type MockServer struct {
	*httptest.Server
	Client     *historian.Client
	RawQuery   string // captured from the last request
	StatusCode int    // HTTP status to return; 0 means 200
}

// NewMockServer creates a mock historian server responding with the given JSON body.
// Calls t.Cleanup to close the server. The Client field is ready to use.
func NewMockServer(t *testing.T, body string) *MockServer {
	t.Helper()
	m := &MockServer{}
	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.RawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		status := m.StatusCode
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	m.Client = historian.New(m.URL, "u", "p", "/Historian/v2", nil)
	t.Cleanup(m.Close)
	return m
}
