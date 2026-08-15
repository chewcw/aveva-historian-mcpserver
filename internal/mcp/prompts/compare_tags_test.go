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

func TestCompareTagsHappyPath(t *testing.T) {
	var requests int
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[{"FQN":"CDE.OEE","StartDateTime":"2024-01-01T00:00:00Z","EndDateTime":"2024-01-01T01:00:00Z","Minimum":90,"Maximum":99,"Average":95.5,"StandardDeviation":1.2,"Integral":1000,"Count":10,"First":95,"Last":96}]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	store := dataserver.NewStore(time.Minute)
	handler := handleCompareTags(client, store, "http://localhost:8199", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"fqn":             "CDE.OEE, CDE.PUMP",
			"start_date_time": "2024-01-01T00:00:00Z",
			"end_date_time":   "2024-01-01T01:00:00Z",
		}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if requests != 2 {
		t.Errorf("want 2 endpoint calls (one per FQN), got %d", requests)
	}
	if len(res.Messages) != 2 { // preview + resource link
		t.Fatalf("want 2 messages, got %d", len(res.Messages))
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "95.5") || !strings.Contains(tc.Text, "10") {
		t.Errorf("preview missing stats: %#v", res.Messages[0].Content)
	}
}

func TestCompareTagsNeedsTwoFqns(t *testing.T) {
	client := historian.New("http://unused", "", "", "", nil)
	handler := handleCompareTags(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"fqn":             "CDE.OEE",
			"start_date_time": "2024-01-01T00:00:00Z",
			"end_date_time":   "2024-01-01T01:00:00Z",
		}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "at least 2 FQNs") {
		t.Errorf("unexpected message: %#v", res.Messages[0].Content)
	}
}

func TestCompareTagsBadResolution(t *testing.T) {
	client := historian.New("http://unused", "", "", "", nil)
	handler := handleCompareTags(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"fqn":             "A, B",
			"start_date_time": "2024-01-01T00:00:00Z",
			"end_date_time":   "2024-01-01T01:00:00Z",
			"resolution_ms":   "abc",
		}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "resolution_ms must be a positive integer") {
		t.Errorf("unexpected message: %#v", res.Messages[0].Content)
	}
}

func TestCompareTagsStartAfterEnd(t *testing.T) {
	client := historian.New("http://unused", "", "", "", nil)
	handler := handleCompareTags(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"fqn":             "A, B",
			"start_date_time": "2024-01-02T00:00:00Z",
			"end_date_time":   "2024-01-01T00:00:00Z",
		}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "start_date_time must be before end_date_time") {
		t.Errorf("unexpected message: %#v", res.Messages[0].Content)
	}
}
