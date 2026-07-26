package endpoints

import (
	"context"
	"strings"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
)

func TestGetTagsQuery(t *testing.T) {
	var rawQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer ts.Close()

	client := historian.New(ts.URL, "u", "p", nil)
	groups := []types.FilterGroupDef{
		{
			And: []types.FilterCondition{
				{Field: "FQN", Operator: "startswith", Value: "CDE"},
			},
		},
	}
	result, err := GetTags(context.Background(), client, groups, 10, 0)
	if err != nil {
		t.Fatalf("GetTags failed: %v (rawQuery=%q)", err, rawQuery)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if rawQuery == "" {
		t.Error("rawQuery is empty")
	}
	if !strings.Contains(rawQuery, "$filter=") {
		t.Errorf("expected $filter= in query, got %q", rawQuery)
	}
}
