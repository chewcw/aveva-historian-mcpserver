package prompts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chewcw/aveva-historian-mcpserver/internal/dataserver"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestTagExplorerHappyPath(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.RawQuery
		if !strings.Contains(q, "TagType+eq") && !strings.Contains(q, "TagType%20eq") {
			t.Errorf("query missing TagType filter: %s", q)
		}
		if !strings.Contains(q, "search") {
			t.Errorf("query missing search: %s", q)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[{"FQN":"CDE.OEE","TagName":"OEE","TagType":"Analog","EngUnit":"%","Source":"CDE"},{"FQN":"CDE.FLOW","TagName":"Flow","TagType":"Analog","EngUnit":"L/min","Source":"CDE"}]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	store := dataserver.NewStore(time.Minute)
	handler := handleTagExplorer(client, store, "http://localhost:8199", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"search": "OEE", "tag_type": "Analog", "top": "50",
		}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(res.Messages) != 2 { // preview + resource link
		t.Fatalf("want 2 messages, got %d", len(res.Messages))
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "CDE.OEE") || !strings.Contains(tc.Text, "CDE.FLOW") {
		t.Errorf("preview missing rows: %#v", res.Messages[0].Content)
	}
	rl, ok := res.Messages[1].Content.(*mcp.ResourceLink)
	if !ok || !strings.HasPrefix(rl.URI, "http://localhost:8199/resources/") {
		t.Errorf("missing resource link: %#v", res.Messages[1].Content)
	}
}

func TestTagExplorerEmpty(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[]}`))
	}))
	defer mock.Close()
	client := historian.New(mock.URL, "", "", "", nil)
	handler := handleTagExplorer(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{"search": "zzz"}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "No tags matched") {
		t.Errorf("unexpected message: %#v", res.Messages[0].Content)
	}
}

func TestTagExplorerTopClamped(t *testing.T) {
	var gotTop string
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTop = r.URL.Query().Get("$top")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[{"FQN":"CDE.OEE"}]}`))
	}))
	defer mock.Close()
	client := historian.New(mock.URL, "", "", "", nil)
	handler := handleTagExplorer(client, nil, "", discardLogger(), 1_048_576)
	if _, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{"top": "5000"}},
	}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if gotTop != "1000" {
		t.Errorf("$top = %q, want clamped 1000", gotTop)
	}
}
