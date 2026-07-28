package endpoints

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
)

func TestGetTagsQuery(t *testing.T) {
	m := NewMockServer(t, `{"value":[]}`)
	groups := []types.FilterGroupDef{
		{
			And: []types.FilterCondition{
				{Field: "FQN", Operator: "startswith", Value: "CDE"},
			},
		},
		{
			And: []types.FilterCondition{
				{Field: "RolloverValue", Operator: "eq", Value: 250.5},
				{Field: "MessageOff", Operator: "eq", Value: "LOW"},
				{Field: "MessageOn", Operator: "eq", Value: "HIGH"},
				{Field: "TagType", Operator: "eq", Value: "Analog"},
			},
		},
	}
	top := 10
	result, err := GetTags(context.Background(), m.Client, groups, types.QueryOptions{Top: &top})
	if err != nil {
		t.Fatalf("GetTags failed: %v (rawQuery=%q)", err, m.RawQuery)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if m.RawQuery == "" {
		t.Error("rawQuery is empty")
	}
	if !strings.Contains(m.RawQuery, "$filter=") {
		t.Errorf("expected $filter= in query, got %q", m.RawQuery)
	}
	decoded, _ := url.QueryUnescape(m.RawQuery)
	if !strings.Contains(decoded, "RolloverValue eq 250.5") {
		t.Errorf("expected RolloverValue filter in query, got %q", m.RawQuery)
	}
	if !strings.Contains(decoded, "MessageOff eq 'LOW'") {
		t.Errorf("expected MessageOff filter in query, got %q", m.RawQuery)
	}
	if !strings.Contains(decoded, "MessageOn eq 'HIGH'") {
		t.Errorf("expected MessageOn filter in query, got %q", m.RawQuery)
	}
	if !strings.Contains(decoded, "TagType eq 'Analog'") {
		t.Errorf("expected TagType filter in query, got %q", m.RawQuery)
	}
}
