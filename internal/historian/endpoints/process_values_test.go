package endpoints

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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
	extra := ProcessValuesParams{
		RetrievalMode: strPtr("Interpolated"),
		ResolutionMS:  intPtr(3600000),
	}
	result, err := GetProcessValues(context.Background(), client, groups, 100, extra)
	if err != nil {
		t.Fatalf("GetProcessValues failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	if rawQuery == "" {
		t.Error("rawQuery is empty")
	}
	if !strings.Contains(rawQuery, "RetrievalMode=Interpolated") {
		t.Errorf("rawQuery missing RetrievalMode=Interpolated: %s", rawQuery)
	}
	if !strings.Contains(rawQuery, "Resolution=3600000") {
		t.Errorf("rawQuery missing Resolution=3600000: %s", rawQuery)
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
	// No filters, no params — just $top
	result, err := GetProcessValues(context.Background(), client, nil, 100, ProcessValuesParams{})
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

func strPtr(s string) *string { return &s }

func intPtr(i int) *int { return &i }
