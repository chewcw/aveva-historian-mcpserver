package endpoints

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
)

func TestGetProcessValuesQuery(t *testing.T) {
	m := NewMockServer(t, `{"value":[]}`)
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
	result, err := GetProcessValues(context.Background(), m.Client, groups, types.QueryOptions{Top: &top})
	if err != nil {
		t.Fatalf("GetProcessValues failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	if !strings.Contains(m.RawQuery, "$filter") {
		t.Errorf("rawQuery missing $filter: %s", m.RawQuery)
	}
	decoded, _ := url.QueryUnescape(m.RawQuery)
	if !strings.Contains(decoded, "RetrievalMode eq 'Interpolated'") {
		t.Errorf("rawQuery missing RetrievalMode filter: %s", m.RawQuery)
	}
	if !strings.Contains(decoded, "Resolution eq 3600000") {
		t.Errorf("rawQuery missing Resolution filter: %s", m.RawQuery)
	}
}

func TestGetProcessValuesMinimal(t *testing.T) {
	m := NewMockServer(t, `{"value":[]}`)
	// No filters, no params — just $top
	top := 100
	result, err := GetProcessValues(context.Background(), m.Client, nil, types.QueryOptions{Top: &top})
	if err != nil {
		t.Fatalf("GetProcessValues failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	want := "$top=100"
	if m.RawQuery != want {
		t.Errorf("raw query = %q, want %q", m.RawQuery, want)
	}

	if len(result.Value) != 0 {
		t.Errorf("expected 0 values, got %d", len(result.Value))
	}
}

func TestGetProcessValuesWithResults(t *testing.T) {
	m := NewMockServer(t, `{"@odata.context":"https://host/Historian/v2/$metadata#ProcessValues","value":[{"FQN":"CDE.OEE.Availability","DateTime":"2024-01-01T00:00:00Z","Value":98.5,"OpcQuality":192,"Unit":"%"},{"FQN":"CDE.OEE.Performance","DateTime":"2024-01-01T00:01:00Z","Value":null,"Unit":"%"}]}`)
	top := 100
	result, err := GetProcessValues(context.Background(), m.Client, nil, types.QueryOptions{Top: &top})
	if err != nil {
		t.Fatalf("GetProcessValues failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	if !strings.Contains(result.Context, "ProcessValues") {
		t.Errorf("Context = %q, want substring %q", result.Context, "ProcessValues")
	}

	if len(result.Value) != 2 {
		t.Fatalf("len(Value) = %d, want 2", len(result.Value))
	}

	v0 := result.Value[0]
	if v0.FQN != "CDE.OEE.Availability" {
		t.Errorf("Value[0].FQN = %q, want %q", v0.FQN, "CDE.OEE.Availability")
	}
	if v0.DateTime != "2024-01-01T00:00:00Z" {
		t.Errorf("Value[0].DateTime = %q, want %q", v0.DateTime, "2024-01-01T00:00:00Z")
	}
	if v0.Value == nil {
		t.Fatal("Value[0].Value is nil, want 98.5")
	}
	if *v0.Value != 98.5 {
		t.Errorf("Value[0].Value = %v, want 98.5", *v0.Value)
	}
	if v0.OpcQuality == nil {
		t.Fatal("Value[0].OpcQuality is nil, want 192")
	}
	if *v0.OpcQuality != 192 {
		t.Errorf("Value[0].OpcQuality = %v, want 192", *v0.OpcQuality)
	}
	if v0.Unit != "%" {
		t.Errorf("Value[0].Unit = %q, want %%", v0.Unit)
	}

	v1 := result.Value[1]
	if v1.FQN != "CDE.OEE.Performance" {
		t.Errorf("Value[1].FQN = %q, want %q", v1.FQN, "CDE.OEE.Performance")
	}
	if v1.Value != nil {
		t.Errorf("Value[1].Value = %v, want nil", *v1.Value)
	}
}

func TestGetProcessValuesHTTPError(t *testing.T) {
	m := NewMockServer(t, `{"error":{"code":"BadRequest","message":"Invalid filter"}}`)
	m.StatusCode = http.StatusBadRequest
	top := 100
	_, err := GetProcessValues(context.Background(), m.Client, nil, types.QueryOptions{Top: &top})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error = %q, want substring %q", err.Error(), "400")
	}
}

func TestGetProcessValuesBadJSON(t *testing.T) {
	m := NewMockServer(t, `this is not json`)
	top := 100
	_, err := GetProcessValues(context.Background(), m.Client, nil, types.QueryOptions{Top: &top})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("error = %q, want substring %q", err.Error(), "decode")
	}
}
