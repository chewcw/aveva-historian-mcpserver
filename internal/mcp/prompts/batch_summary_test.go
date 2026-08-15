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

func TestBatchSummaryHappyPath(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[
			{"FQN":"CDE.OEE","StartDateTime":"2024-01-01T08:00:00Z","EndDateTime":"2024-01-01T09:00:00Z","First":10.0,"Last":20.0,"Minimum":5.0,"Maximum":25.0,"Average":15.0,"StandardDeviation":2.5,"Integral":54000.0,"Count":3600,"SliceByValue":"batch-42"}
		]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	store := dataserver.NewStore(time.Minute)
	handler := handleBatchSummary(client, store, "http://localhost:8199", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"fqn": "CDE.OEE", "start_date_time": "2024-01-01T00:00:00Z",
			"end_date_time": "2024-01-02T00:00:00Z", "slice_by": "CDE.OEE",
			"slice_by_value": "batch-42",
		}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(res.Messages) != 2 {
		t.Fatalf("want 2 messages, got %d", len(res.Messages))
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "## Batch Summary: CDE.OEE") {
		t.Errorf("preview missing header: %#v", res.Messages[0].Content)
	}
	if !strings.Contains(tc.Text, "54000") || !strings.Contains(tc.Text, "batch-42") {
		t.Errorf("preview missing data: %s", tc.Text)
	}
}

func TestBatchSummaryMissingSliceBy(t *testing.T) {
	client := historian.New("http://unused", "", "", "", nil)
	handler := handleBatchSummary(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"fqn": "CDE.OEE", "start_date_time": "2024-01-01T00:00:00Z", "end_date_time": "2024-01-02T00:00:00Z",
		}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "Missing required argument: slice_by") {
		t.Errorf("unexpected message: %#v", res.Messages[0].Content)
	}
}

func TestBatchSummaryTooManySliceBy(t *testing.T) {
	client := historian.New("http://unused", "", "", "", nil)
	handler := handleBatchSummary(client, nil, "", discardLogger(), 1_048_576)
	fqns := "A,B,C,D,E,F,G,H,I,J,K" // 11
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"fqn": "CDE.OEE", "start_date_time": "2024-01-01T00:00:00Z",
			"end_date_time": "2024-01-02T00:00:00Z", "slice_by": fqns,
		}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "Too many FQNs") {
		t.Errorf("unexpected message: %#v", res.Messages[0].Content)
	}
}
