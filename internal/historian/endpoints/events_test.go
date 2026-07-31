package endpoints

import (
	"context"
	"strings"
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
)

func TestGetEvents_BasicQuery(t *testing.T) {
	m := NewMockServer(t, `{"value":[{"id":"a","eventtime":"2024-01-01T00:00:00Z","type":"Alarm.Set"}]}`)
	top := 10
	result, err := GetEvents(context.Background(), m.Client, nil, types.QueryOptions{Top: &top})
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(result.Value) != 1 {
		t.Fatalf("got %d events, want 1", len(result.Value))
	}
	if result.Value[0].ID != "a" {
		t.Errorf("id = %q, want %q", result.Value[0].ID, "a")
	}
	want := "$top=10"
	if m.RawQuery != want {
		t.Errorf("raw query = %q, want %q", m.RawQuery, want)
	}
}

func TestGetEvents_WithFilters(t *testing.T) {
	m := NewMockServer(t, `{"value":[]}`)
	groups := []types.FilterGroupDef{
		{And: []types.FilterCondition{
			{Field: "Severity", Operator: "eq", Value: 1},
		}},
	}
	result, err := GetEvents(context.Background(), m.Client, groups, types.QueryOptions{})
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if result.Value == nil {
		t.Error("expected non-nil value slice")
	}
	if !strings.Contains(m.RawQuery, "$filter") {
		t.Errorf("rawQuery missing $filter: %s", m.RawQuery)
	}
	if !strings.Contains(m.RawQuery, "Severity") {
		t.Errorf("rawQuery missing Severity: %s", m.RawQuery)
	}
}

func TestGetEvents_EmptyResult(t *testing.T) {
	m := NewMockServer(t, `{"value":[]}`)
	result, err := GetEvents(context.Background(), m.Client, nil, types.QueryOptions{})
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if result.Value == nil {
		t.Error("expected non-nil value slice")
	}
	if len(result.Value) != 0 {
		t.Errorf("got %d events, want 0", len(result.Value))
	}
}
