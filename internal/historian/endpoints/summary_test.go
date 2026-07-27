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
	result, err := GetAnalogSummary(context.Background(), client, groups, 3600000, 100, AnalogSummaryParams{})
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
	result, err := GetAnalogSummary(context.Background(), client, nil, 0, 10, AnalogSummaryParams{})
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

func TestGetAnalogSummaryWithExtraParams(t *testing.T) {
	var rawQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer ts.Close()

	client := historian.New(ts.URL, "u", "p", nil)
	retrievalMode := "Full"
	sliceBy := "CDE.OEE,CDE.OEE2"
	sliceByValue := "Active"
	opcQuality := 192
	percentGood := 95.0
	extra := AnalogSummaryParams{
		RetrievalMode: &retrievalMode,
		SliceBy:       &sliceBy,
		SliceByValue:  &sliceByValue,
		OPCQuality:    &opcQuality,
		PercentGood:   &percentGood,
	}

	result, err := GetAnalogSummary(context.Background(), client, nil, 3600000, 100, extra)
	if err != nil {
		t.Fatalf("GetAnalogSummary failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	if !strings.Contains(rawQuery, "RetrievalMode=Full") {
		t.Errorf("expected RetrievalMode=Full in query, got %q", rawQuery)
	}
	if !strings.Contains(rawQuery, "SliceBy=CDE.OEE%2CCDE.OEE2") {
		t.Errorf("expected SliceBy=CDE.OEE%%2CCDE.OEE2 in query, got %q", rawQuery)
	}
	if !strings.Contains(rawQuery, "SliceByValue=Active") {
		t.Errorf("expected SliceByValue=Active in query, got %q", rawQuery)
	}
	if !strings.Contains(rawQuery, "OPCQuality=192") {
		t.Errorf("expected OPCQuality=192 in query, got %q", rawQuery)
	}
	if !strings.Contains(rawQuery, "PercentGood=95") {
		t.Errorf("expected PercentGood=95 in query, got %q", rawQuery)
	}
}

func TestGetAnalogSummaryPartialExtra(t *testing.T) {
	var rawQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer ts.Close()

	client := historian.New(ts.URL, "u", "p", nil)
	sliceBy := "CDE.OEE"
	extra := AnalogSummaryParams{
		SliceBy: &sliceBy,
	}

	result, err := GetAnalogSummary(context.Background(), client, nil, 0, 10, extra)
	if err != nil {
		t.Fatalf("GetAnalogSummary failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	if !strings.Contains(rawQuery, "SliceBy=CDE.OEE") {
		t.Errorf("expected SliceBy=CDE.OEE in query, got %q", rawQuery)
	}
	if strings.Contains(rawQuery, "RetrievalMode") {
		t.Errorf("did not expect RetrievalMode in query, got %q", rawQuery)
	}
	if strings.Contains(rawQuery, "OPCQuality") {
		t.Errorf("did not expect OPCQuality in query, got %q", rawQuery)
	}
	if strings.Contains(rawQuery, "PercentGood") {
		t.Errorf("did not expect PercentGood in query, got %q", rawQuery)
	}
}
