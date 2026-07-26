package endpoints

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
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
	groups := []types.FilterGroupDef{
		{And: []types.FilterCondition{{Field: "FQN", Operator: "eq", Value: "CDE.OEE"}}},
		{And: []types.FilterCondition{
			{Field: "StartDateTime", Operator: "ge", Value: "2024-01-01T00:00:00Z"},
			{Field: "EndDateTime", Operator: "le", Value: "2024-01-02T00:00:00Z"},
		}},
	}
	result, err := GetAnalogSummary(context.Background(), client, groups, 3600000, 100)
	if err != nil {
		t.Fatalf("GetAnalogSummary failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	if rawQuery == "" {
		t.Error("rawQuery is empty")
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
	result, err := GetAnalogSummary(context.Background(), client, nil, 0, 10)
	if err != nil {
		t.Fatalf("GetAnalogSummary failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	want := "$top=10"
	if rawQuery != want {
		t.Errorf("raw query = %q, want %q", rawQuery, want)
	}
}
