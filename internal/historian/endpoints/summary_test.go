package endpoints

import (
	"context"
	"net/url"
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
		{And: []types.FilterCondition{
			{Field: "Resolution", Operator: "eq", Value: 3600000},
		}},
	}
	top := 100
	result, err := GetAnalogSummary(context.Background(), m.Client, groups, types.QueryOptions{Top: &top})
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
	top := 10
	result, err := GetAnalogSummary(context.Background(), m.Client, nil, types.QueryOptions{Top: &top})
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
	top := 100
	groups := []types.FilterGroupDef{
		{And: []types.FilterCondition{
			{Field: "RetrievalMode", Operator: "eq", Value: "Full"},
			{Field: "Resolution", Operator: "eq", Value: 3600000},
			{Field: "SliceBy", Operator: "eq", Value: "CDE.OEE,CDE.OEE2"},
			{Field: "SliceByValue", Operator: "eq", Value: "Active"},
			{Field: "OPCQuality", Operator: "eq", Value: 192},
			{Field: "PercentGood", Operator: "eq", Value: 95.0},
		}},
	}

	result, err := GetAnalogSummary(context.Background(), m.Client, groups, types.QueryOptions{Top: &top})
	if err != nil {
		t.Fatalf("GetAnalogSummary failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	decoded, _ := url.QueryUnescape(m.RawQuery)
	if !strings.Contains(decoded, "RetrievalMode eq 'Full'") {
		t.Errorf("expected RetrievalMode filter in query, got %q", m.RawQuery)
	}
	if !strings.Contains(decoded, "SliceBy eq 'CDE.OEE,CDE.OEE2'") {
		t.Errorf("expected SliceBy filter in query, got %q", m.RawQuery)
	}
	if !strings.Contains(decoded, "SliceByValue eq 'Active'") {
		t.Errorf("expected SliceByValue filter in query, got %q", m.RawQuery)
	}
	if !strings.Contains(decoded, "OPCQuality eq 192") {
		t.Errorf("expected OPCQuality filter in query, got %q", m.RawQuery)
	}
	if !strings.Contains(decoded, "PercentGood eq 95") {
		t.Errorf("expected PercentGood filter in query, got %q", m.RawQuery)
	}
}

func TestGetAnalogSummaryPartialExtra(t *testing.T) {
	m := NewMockServer(t, `{"value":[]}`)
	top := 10
	groups := []types.FilterGroupDef{
		{And: []types.FilterCondition{
			{Field: "SliceBy", Operator: "eq", Value: "CDE.OEE"},
		}},
	}

	result, err := GetAnalogSummary(context.Background(), m.Client, groups, types.QueryOptions{Top: &top})
	if err != nil {
		t.Fatalf("GetAnalogSummary failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	decoded, _ := url.QueryUnescape(m.RawQuery)
	if !strings.Contains(decoded, "SliceBy eq 'CDE.OEE'") {
		t.Errorf("expected SliceBy filter in query, got %q", m.RawQuery)
	}
	if strings.Contains(decoded, "RetrievalMode") {
		t.Errorf("did not expect RetrievalMode in query, got %q", m.RawQuery)
	}
	if strings.Contains(decoded, "OPCQuality") {
		t.Errorf("did not expect OPCQuality in query, got %q", m.RawQuery)
	}
	if strings.Contains(decoded, "PercentGood") {
		t.Errorf("did not expect PercentGood in query, got %q", m.RawQuery)
	}
}
