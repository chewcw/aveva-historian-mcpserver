package endpoints

import (
	"context"
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
	}
	result, err := GetTags(context.Background(), m.Client, groups, 10, 0, TagsParams{
		RolloverValue: new(float64(250.5)),
		MessageOff:    new("LOW"),
		MessageOn:     new("HIGH"),
		TagType:       new("Analog"),
	})
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
	if !strings.Contains(m.RawQuery, "RolloverValue=250.5") {
		t.Errorf("expected RolloverValue in query, got %q", m.RawQuery)
	}
	if !strings.Contains(m.RawQuery, "MessageOff=LOW") {
		t.Errorf("expected MessageOff in query, got %q", m.RawQuery)
	}
	if !strings.Contains(m.RawQuery, "MessageOn=HIGH") {
		t.Errorf("expected MessageOn in query, got %q", m.RawQuery)
	}
	if !strings.Contains(m.RawQuery, "TagType=Analog") {
		t.Errorf("expected TagType in query, got %q", m.RawQuery)
	}
}
