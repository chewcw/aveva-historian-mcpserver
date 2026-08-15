package prompts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestTagProfilerHappyPath(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "FQN+eq") && !strings.Contains(r.URL.RawQuery, "FQN%20eq") {
			t.Errorf("query missing FQN filter: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[{"FQN":"CDE.OEE","TagName":"OEE","TagType":"Analog","EngUnit":"%","Source":"CDE","Description":"Overall equipment effectiveness"}]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	handler := handleTagProfiler(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{"fqn": "CDE.OEE"}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(res.Messages) != 1 { // store=nil → no resource link
		t.Fatalf("want 1 message, got %d", len(res.Messages))
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("message content = %#v", res.Messages[0].Content)
	}
	if !strings.Contains(tc.Text, "CDE.OEE") || !strings.Contains(tc.Text, "Overall equipment effectiveness") {
		t.Errorf("preview missing fields: %s", tc.Text)
	}
}

func TestTagProfilerMissingFqn(t *testing.T) {
	client := historian.New("http://unused", "", "", "", nil)
	handler := handleTagProfiler(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "Missing required argument: fqn") {
		t.Errorf("unexpected error message: %#v", res.Messages[0].Content)
	}
}

func TestTagProfilerNoMatch(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[]}`))
	}))
	defer mock.Close()
	client := historian.New(mock.URL, "", "", "", nil)
	handler := handleTagProfiler(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{"fqn": "NOPE"}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "No tag matched") {
		t.Errorf("unexpected message: %#v", res.Messages[0].Content)
	}
}

func TestTagProfilerEndpointError(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer mock.Close()
	client := historian.New(mock.URL, "", "", "", nil)
	handler := handleTagProfiler(client, nil, "", discardLogger(), 1_048_576)
	if _, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{"fqn": "CDE.OEE"}},
	}); err == nil {
		t.Fatal("expected error from failing endpoint")
	}
}
