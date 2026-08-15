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

func TestTrendReportHappyPath(t *testing.T) {
	var summaryHits, valuesHits int
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/AnalogSummary") {
			summaryHits++
			w.Write([]byte(`{"value":[{"FQN":"CDE.OEE","StartDateTime":"2024-01-01T00:00:00Z","EndDateTime":"2024-01-01T01:00:00Z","Minimum":90,"Maximum":99,"Average":95.5,"StandardDeviation":1.2,"Integral":1000,"Count":10,"First":95,"Last":96}]}`))
			return
		}
		valuesHits++
		w.Write([]byte(`{"value":[{"FQN":"CDE.OEE","DateTime":"2024-01-01T00:00:00Z","Value":95.5,"OpcQuality":192,"Unit":"%"}]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	store := dataserver.NewStore(time.Minute)
	handler := handleTrendReport(client, store, "http://localhost:8199", discardLogger(), 1_048_576)
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
	if summaryHits != 1 || valuesHits != 1 {
		t.Errorf("summary=%d values=%d, want 1 each", summaryHits, valuesHits)
	}
	if len(res.Messages) != 3 { // preview + summary link + values link
		t.Fatalf("want 3 messages, got %d", len(res.Messages))
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "95.5") || !strings.Contains(tc.Text, "Trend Report") {
		t.Errorf("preview missing stats: %#v", res.Messages[0].Content)
	}
}

func TestTrendReportMissingArgs(t *testing.T) {
	client := historian.New("http://unused", "", "", "", nil)
	handler := handleTrendReport(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{"fqn": "CDE.OEE"}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "Missing required argument: start_date_time") {
		t.Errorf("unexpected message: %#v", res.Messages[0].Content)
	}
}

func TestTrendReportBadMode(t *testing.T) {
	client := historian.New("http://unused", "", "", "", nil)
	handler := handleTrendReport(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"fqn":             "CDE.OEE",
			"start_date_time": "2024-01-01T00:00:00Z",
			"end_date_time":   "2024-01-01T01:00:00Z",
			"retrieval_mode":  "Bogus",
		}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "unsupported retrieval mode") {
		t.Errorf("unexpected message: %#v", res.Messages[0].Content)
	}
}

func TestTrendReportStartAfterEnd(t *testing.T) {
	client := historian.New("http://unused", "", "", "", nil)
	handler := handleTrendReport(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"fqn":             "CDE.OEE",
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
