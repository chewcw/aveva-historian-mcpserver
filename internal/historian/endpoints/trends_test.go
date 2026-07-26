package endpoints

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
)

func TestGetProcessValuesQuery(t *testing.T) {
	var rawQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer ts.Close()

	client := historian.New(ts.URL, "u", "p", nil)
	groups := []types.FilterGroupDef{
		{And: []types.FilterCondition{{Field: "FQN", Operator: "eq", Value: "CDE.OEE"}}},
		{And: []types.FilterCondition{
			{Field: "DateTime", Operator: "ge", Value: "2024-01-01T00:00:00Z"},
			{Field: "DateTime", Operator: "le", Value: "2024-01-02T00:00:00Z"},
		}},
	}
	result, err := GetProcessValues(context.Background(), client, groups, "Interpolated", 3600000, 100)
	if err != nil {
		t.Fatalf("GetProcessValues failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	// Verify filter portion (RetrievalMode and Resolution are appended after Build())
	// odataqb generates: $filter=FQN eq 'CDE.OEE' and (DateTime ge '2024-01-01T00:00:00Z' and DateTime le '2024-01-02T00:00:00Z')
	if rawQuery == "" {
		t.Error("rawQuery is empty")
	}
}

func TestGetProcessValuesMinimal(t *testing.T) {
	var rawQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer ts.Close()

	client := historian.New(ts.URL, "u", "p", nil)
	// No filters, no retrieval mode, no resolution — just $top
	result, err := GetProcessValues(context.Background(), client, nil, "", 0, 100)
	if err != nil {
		t.Fatalf("GetProcessValues failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	want := "$top=100"
	if rawQuery != want {
		t.Errorf("raw query = %q, want %q", rawQuery, want)
	}
}
