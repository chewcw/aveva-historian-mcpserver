package dataserver

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func setupTestServer(t *testing.T) (*Store, *httptest.Server) {
	t.Helper()
	store := NewStore(5 * time.Minute)
	mux := http.NewServeMux()
	s := &server{store: store, logger: slog.Default()}
	mux.HandleFunc("GET /resources", s.handleList)
	mux.HandleFunc("GET /resources/{id}", s.handleGet)
	mux.HandleFunc("POST /resources/{id}/pin", s.handlePin)
	mux.HandleFunc("DELETE /resources/{id}", s.handleDelete)
	ts := httptest.NewServer(recoveryMiddleware(slog.Default(), mux))
	t.Cleanup(ts.Close)
	return store, ts
}

func TestGetResourcesList(t *testing.T) {
	store, ts := setupTestServer(t)
	store.Put("alpha", "", nil, nil)
	store.Put("beta", "", nil, nil)

	resp, err := http.Get(ts.URL + "/resources")
	if err != nil {
		t.Fatalf("GET /resources: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var list []ResourceSummary
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
}

func TestGetResourceByID(t *testing.T) {
	store, ts := setupTestServer(t)
	columns := []Field{{Name: "col1"}, {Name: "col2"}}
	rows := [][]string{{"a", "1"}, {"b", "2"}, {"c", "3"}}
	res, _ := store.Put("test", "application/json", columns, rows)

	resp, err := http.Get(ts.URL + "/resources/" + res.ID.String())
	if err != nil {
		t.Fatalf("GET /resources/{id}: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	page := body["page"].(map[string]any)
	rowsResult := page["rows"].([]any)
	if len(rowsResult) != 3 {
		t.Errorf("row count = %d, want 3", len(rowsResult))
	}
}

func TestGetResourceNotFound(t *testing.T) {
	_, ts := setupTestServer(t)
	resp, err := http.Get(ts.URL + "/resources/" + uuid.New().String())
	if err != nil {
		t.Fatalf("GET /resources/{id}: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestGetResourcePagination(t *testing.T) {
	store, ts := setupTestServer(t)
	rows := make([][]string, 250)
	for i := range rows {
		rows[i] = []string{fmt.Sprintf("row-%d", i+1)}
	}
	res, _ := store.Put("paging", "text/plain", []Field{{Name: "data"}}, rows)

	// First page
	resp, err := http.Get(ts.URL + "/resources/" + res.ID.String())
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	defer resp.Body.Close()
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	page := body["page"].(map[string]any)
	rowsResult := page["rows"].([]any)
	firstCursor := page["cursor"]
	hasMore := page["hasMore"].(bool)
	if len(rowsResult) != 100 {
		t.Errorf("first page len = %d, want 100", len(rowsResult))
	}
	if !hasMore {
		t.Error("expected hasMore=true")
	}

	// Second page
	resp2, err := http.Get(fmt.Sprintf("%s/resources/%s?cursor=%v", ts.URL, res.ID.String(), firstCursor))
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	defer resp2.Body.Close()
	json.NewDecoder(resp2.Body).Decode(&body)
	page2 := body["page"].(map[string]any)
	rows2 := page2["rows"].([]any)
	if len(rows2) != 100 {
		t.Errorf("second page len = %d, want 100", len(rows2))
	}
}

func TestPinAndDeleteViaHTTP(t *testing.T) {
	store, ts := setupTestServer(t)
	res, _ := store.Put("test", "", nil, nil)

	// Pin with maxAge
	resp, err := http.Post(ts.URL+"/resources/"+res.ID.String()+"/pin",
		"application/json", strings.NewReader(`{"maxAge": "15m"}`))
	if err != nil {
		t.Fatalf("POST pin: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	r, _ := store.Get(res.ID)
	if !r.Pinned {
		t.Error("expected pinned=true")
	}
	if *r.MaxAge != 15*time.Minute {
		t.Errorf("MaxAge = %v, want 15m", *r.MaxAge)
	}

	// Delete
	req, err := http.NewRequest("DELETE", ts.URL+"/resources/"+res.ID.String(), nil)
	if err != nil {
		t.Fatalf("DELETE new request: %v", err)
	}
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE req: %v", err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE status = %d, want 204", delResp.StatusCode)
	}
	_, ok := store.Get(res.ID)
	if ok {
		t.Error("resource still exists after DELETE")
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	ts := httptest.NewServer(recoveryMiddleware(slog.Default(), mux))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/panic")
	if err != nil {
		t.Fatalf("GET /panic: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
}
