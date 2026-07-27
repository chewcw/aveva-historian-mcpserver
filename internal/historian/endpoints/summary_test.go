package endpoints

import (
	"context"
	"strings"
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
)

func TestGetAnalogSummaryQuery(t *testing.T) {
	m := NewMockServer(t, `{"value":[]}`)
	groups := []types.FilterGroupDef{
		{And: []types.FilterCondition{{Field: "FQN", Operator: "eq", Value: "CDE.OEE"}}},
		{And: []types.FilterCondition{
			{Field: "StartDateTime", Operator: "ge", Value: "2024-01-01T00:00:00Z"},
			{Field: "EndDateTime", Operator: "le", Value: "2024-01-02T00:00:00Z"},
		}},
	}
	result, err := GetAnalogSummary(context.Background(), m.Client, groups, 3600000, 100, AnalogSummaryParams{})
	if err != nil {
		t.Fatalf("GetAnalogSummary failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	if m.RawQuery == "" {
		t.Error("rawQuery is empty")
	}
}

func TestGetAnalogSummaryMinimal(t *testing.T) {
	m := NewMockServer(t, `{"value":[]}`)
	result, err := GetAnalogSummary(context.Background(), m.Client, nil, 0, 10, AnalogSummaryParams{})
	if err != nil {
		t.Fatalf("GetAnalogSummary failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	want := "$top=10"
	if m.RawQuery != want {
		t.Errorf("raw query = %q, want %q", m.RawQuery, want)
	}
}

func TestGetAnalogSummaryWithExtraParams(t *testing.T) {
	m := NewMockServer(t, `{"value":[]}`)
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

	result, err := GetAnalogSummary(context.Background(), m.Client, nil, 3600000, 100, extra)
	if err != nil {
		t.Fatalf("GetAnalogSummary failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	if !strings.Contains(m.RawQuery, "RetrievalMode=Full") {
		t.Errorf("expected RetrievalMode=Full in query, got %q", m.RawQuery)
	}
	if !strings.Contains(m.RawQuery, "SliceBy=CDE.OEE%2CCDE.OEE2") {
		t.Errorf("expected SliceBy=CDE.OEE%%2CCDE.OEE2 in query, got %q", m.RawQuery)
	}
	if !strings.Contains(m.RawQuery, "SliceByValue=Active") {
		t.Errorf("expected SliceByValue=Active in query, got %q", m.RawQuery)
	}
	if !strings.Contains(m.RawQuery, "OPCQuality=192") {
		t.Errorf("expected OPCQuality=192 in query, got %q", m.RawQuery)
	}
	if !strings.Contains(m.RawQuery, "PercentGood=95") {
		t.Errorf("expected PercentGood=95 in query, got %q", m.RawQuery)
	}
}

func TestGetAnalogSummaryPartialExtra(t *testing.T) {
	m := NewMockServer(t, `{"value":[]}`)
	sliceBy := "CDE.OEE"
	extra := AnalogSummaryParams{
		SliceBy: &sliceBy,
	}

	result, err := GetAnalogSummary(context.Background(), m.Client, nil, 0, 10, extra)
	if err != nil {
		t.Fatalf("GetAnalogSummary failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	if !strings.Contains(m.RawQuery, "SliceBy=CDE.OEE") {
		t.Errorf("expected SliceBy=CDE.OEE in query, got %q", m.RawQuery)
	}
	if strings.Contains(m.RawQuery, "RetrievalMode") {
		t.Errorf("did not expect RetrievalMode in query, got %q", m.RawQuery)
	}
	if strings.Contains(m.RawQuery, "OPCQuality") {
		t.Errorf("did not expect OPCQuality in query, got %q", m.RawQuery)
	}
	if strings.Contains(m.RawQuery, "PercentGood") {
		t.Errorf("did not expect PercentGood in query, got %q", m.RawQuery)
	}
}
