package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian/endpoints"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
)

func TestReadEvents_BoolPtrStr(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if got := boolPtrStr(nil); got != "" {
			t.Errorf("boolPtrStr(nil) = %q, want ''", got)
		}
	})
	t.Run("true", func(t *testing.T) {
		v := true
		if got := boolPtrStr(&v); got != "true" {
			t.Errorf("boolPtrStr(true) = %q, want 'true'", got)
		}
	})
	t.Run("false", func(t *testing.T) {
		v := false
		if got := boolPtrStr(&v); got != "false" {
			t.Errorf("boolPtrStr(false) = %q, want 'false'", got)
		}
	})
}

func TestReadEvents_EndToEnd(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.RawQuery
		if !strings.Contains(q, "Type%20eq") && !strings.Contains(q, "Type+eq") {
			t.Errorf("query missing Type filter: %s", q)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[{"id":"abc","eventtime":"2024-01-01T00:00:00Z","type":"Alarm.Set","severity":1}]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	ctx := context.Background()

	// Test via endpoint directly with filter conditions that match what the handler would produce
	conds := []types.FilterCondition{
		{Field: "Type", Operator: "eq", Value: "Alarm.Set"},
	}
	groups := []types.FilterGroupDef{{And: conds}}
	top := 100
	opts := types.QueryOptions{Top: &top}
	result, err := endpoints.GetEvents(ctx, client, groups, opts)
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(result.Value) != 1 {
		t.Fatalf("got %d events, want 1", len(result.Value))
	}
	if result.Value[0].Type != "Alarm.Set" {
		t.Errorf("event type = %q, want 'Alarm.Set'", result.Value[0].Type)
	}
}
