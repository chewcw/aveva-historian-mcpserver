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

func TestEnergyUsageHappyPath(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[{"FQN":"CDE.OEE","StartDateTime":"2024-01-01T00:00:00Z","EndDateTime":"2024-01-01T01:00:00Z","Integral":120.5,"Average":80.1,"Maximum":95.5,"Minimum":60.2,"PercentGood":99.1,"Count":3600}]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	store := dataserver.NewStore(time.Minute)
	handler := handleEnergyUsage(client, store, "http://localhost:8199", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"fqn": "CDE.OEE", "start_date_time": "2024-01-01T00:00:00Z", "end_date_time": "2024-01-02T00:00:00Z",
		}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(res.Messages) != 2 { // preview + resource link
		t.Fatalf("want 2 messages, got %d", len(res.Messages))
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("message 0 not TextContent: %#v", res.Messages[0].Content)
	}
	if !strings.Contains(tc.Text, "120.5") {
		t.Errorf("preview missing integral value: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "(default)") {
		t.Errorf("preview missing default slice flag: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "Total integral") {
		t.Errorf("preview missing total: %s", tc.Text)
	}
	rl, ok := res.Messages[1].Content.(*mcp.ResourceLink)
	if !ok || !strings.HasPrefix(rl.URI, "http://localhost:8199/resources/") {
		t.Errorf("missing resource link: %#v", res.Messages[1].Content)
	}
}

func TestEnergyUsageMissingRequired(t *testing.T) {
	client := historian.New("http://unused", "", "", "", nil)
	handler := handleEnergyUsage(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "Missing required argument: fqn") {
		t.Errorf("unexpected message: %#v", res.Messages[0].Content)
	}
}

func TestEnergyUsageBadResolution(t *testing.T) {
	client := historian.New("http://unused", "", "", "", nil)
	handler := handleEnergyUsage(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"fqn": "CDE.OEE", "start_date_time": "2024-01-01T00:00:00Z",
			"end_date_time": "2024-01-02T00:00:00Z", "resolution_ms": "0",
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

func TestEnergyUsageStartAfterEnd(t *testing.T) {
	client := historian.New("http://unused", "", "", "", nil)
	handler := handleEnergyUsage(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"fqn": "CDE.OEE", "start_date_time": "2024-01-02T00:00:00Z", "end_date_time": "2024-01-01T00:00:00Z",
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
