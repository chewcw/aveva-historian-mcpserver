package endpoints

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
)

func TestGetAnalogSummaryQuery(t *testing.T) {
	var rawQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer ts.Close()

	client := historian.New(ts.URL, "u", "p", nil)
	result, err := GetAnalogSummary(context.Background(), client,
		"CDE.OEE", "2024-01-01T00:00:00Z", "2024-01-02T00:00:00Z", 3600000, 100)
	if err != nil {
		t.Fatalf("GetAnalogSummary failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	want := "$filter=FQN+eq+%27CDE.OEE%27+and+StartDateTime+ge+2024-01-01T00%3A00%3A00Z+and+EndDateTime+le+2024-01-02T00%3A00%3A00Z&Resolution=3600000&$top=100"
	if rawQuery != want {
		t.Errorf("raw query = %q, want %q", rawQuery, want)
	}
}

func TestGetAnalogSummaryMinimal(t *testing.T) {
	var rawQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer ts.Close()

	client := historian.New(ts.URL, "u", "p", nil)
	result, err := GetAnalogSummary(context.Background(), client,
		"CDE.OEE", "2024-01-01T00:00:00Z", "2024-01-02T00:00:00Z", 0, 100)
	if err != nil {
		t.Fatalf("GetAnalogSummary failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	want := "$filter=FQN+eq+%27CDE.OEE%27+and+StartDateTime+ge+2024-01-01T00%3A00%3A00Z+and+EndDateTime+le+2024-01-02T00%3A00%3A00Z&$top=100"
	if rawQuery != want {
		t.Errorf("raw query = %q, want %q", rawQuery, want)
	}
}
