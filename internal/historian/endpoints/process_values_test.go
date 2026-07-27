package endpoints

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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
		{And: []types.FilterCondition{
			{Field: "RetrievalMode", Operator: "eq", Value: "Interpolated"},
			{Field: "Resolution", Operator: "eq", Value: 3600000},
		}},
	}
	top := 100
	result, err := GetProcessValues(context.Background(), client, groups, types.QueryOptions{Top: &top})
	if err != nil {
		t.Fatalf("GetProcessValues failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	if !strings.Contains(rawQuery, "$filter") {
		t.Errorf("rawQuery missing $filter: %s", rawQuery)
	}
	decoded, _ := url.QueryUnescape(rawQuery)
	if !strings.Contains(decoded, "RetrievalMode eq 'Interpolated'") {
		t.Errorf("rawQuery missing RetrievalMode filter: %s", rawQuery)
	}
	if !strings.Contains(decoded, "Resolution eq 3600000") {
		t.Errorf("rawQuery missing Resolution filter: %s", rawQuery)
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
	top := 100
	result, err := GetProcessValues(context.Background(), client, nil, types.QueryOptions{Top: &top})
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
