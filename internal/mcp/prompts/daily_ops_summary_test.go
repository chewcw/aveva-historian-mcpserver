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

// Mock routes by path suffix: the client builds "<baseURL>/<prefix>/<endpoint>"
// with prefix "/" even when empty, so paths arrive as "//Events"/"//AnalogSummary".
func TestDailyOpsSummaryWithFqns(t *testing.T) {
	var eventsCalls, summaryCalls int
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/Events"):
			eventsCalls++
			w.Write([]byte(`{"value":[
				{"id":"e1","eventtime":"2024-01-01T10:00:00Z","type":"Alarm.Set","severity":1,"isalarm":true,"source_name":"P-101A","namespace":"Area1"}
			]}`))
		case strings.HasSuffix(r.URL.Path, "/AnalogSummary"):
			summaryCalls++
			w.Write([]byte(`{"value":[{"FQN":"CDE.OEE","StartDateTime":"2024-01-01T00:00:00Z","EndDateTime":"2024-01-01T23:59:59Z","Minimum":5.0,"Maximum":25.0,"Average":15.0,"StandardDeviation":2.5,"Integral":54000.0,"Count":3600}]}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	store := dataserver.NewStore(time.Minute)
	handler := handleDailyOpsSummary(client, store, "http://localhost:8199", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"date": "2024-01-01", "fqns": "CDE.OEE",
		}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if eventsCalls != 1 || summaryCalls != 1 {
		t.Errorf("calls: events=%d summary=%d, want 1 each", eventsCalls, summaryCalls)
	}
	if len(res.Messages) != 3 { // preview + events link + summary link
		t.Fatalf("want 3 messages, got %d", len(res.Messages))
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("message 0 not TextContent: %#v", res.Messages[0].Content)
	}
	if !strings.Contains(tc.Text, "## Daily Ops Summary: 2024-01-01") {
		t.Errorf("preview missing header: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "CDE.OEE") || !strings.Contains(tc.Text, "54000") {
		t.Errorf("preview missing per-tag stats: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "Alarm.Set") {
		t.Errorf("preview missing event digest: %s", tc.Text)
	}
}

func TestDailyOpsSummaryNoFqnsHint(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[{"id":"e1","eventtime":"2024-01-01T10:00:00Z","type":"Alarm.Set","severity":1,"isalarm":true,"source_name":"P-101A","namespace":"Area1"}]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	store := dataserver.NewStore(time.Minute) // real store so the events link message exists
	handler := handleDailyOpsSummary(client, store, "http://localhost:8199", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{"date": "2024-01-01"}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(res.Messages) != 2 { // preview + events link only
		t.Fatalf("want 2 messages, got %d", len(res.Messages))
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "run tag_explorer to discover tags") {
		t.Errorf("preview missing drill-down hint: %#v", res.Messages[0].Content)
	}
}

func TestDailyOpsSummaryDefaultDate(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[{"id":"e1","eventtime":"2024-01-01T10:00:00Z","type":"Alarm.Set","severity":1,"isalarm":true,"source_name":"P-101A","namespace":"Area1"}]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	handler := handleDailyOpsSummary(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "(today, default)") {
		t.Errorf("preview missing today default flag: %#v", res.Messages[0].Content)
	}
}

func TestDailyOpsSummaryBadDate(t *testing.T) {
	client := historian.New("http://unused", "", "", "", nil)
	handler := handleDailyOpsSummary(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{"date": "01/01/2024"}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "date must be YYYY-MM-DD") {
		t.Errorf("unexpected message: %#v", res.Messages[0].Content)
	}
}

func TestDailyOpsSummaryFiltersReachQuery(t *testing.T) {
	var rawQuery string
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[{"id":"e1","eventtime":"2024-01-01T10:00:00Z","type":"Alarm.Set","severity":1,"isalarm":true,"source_name":"P-101A","namespace":"Area1"}]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	handler := handleDailyOpsSummary(client, nil, "", discardLogger(), 1_048_576)
	if _, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"date":      "2024-01-01",
			"severity":  "1",
			"namespace": "Area1",
		}},
	}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	for _, want := range []string{"Severity", "Namespace"} {
		if !strings.Contains(rawQuery, want) {
			t.Errorf("query missing %s: %s", want, rawQuery)
		}
	}
}

func TestDailyOpsSummaryNoEventsSingleMessage(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	store := dataserver.NewStore(time.Minute) // real store; empty events must not produce a link
	handler := handleDailyOpsSummary(client, store, "http://localhost:8199", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{"date": "2024-01-01"}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(res.Messages) != 1 { // preview only — empty events push is skipped, resultMessage drops empty refs
		t.Fatalf("want 1 message, got %d", len(res.Messages))
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "No events in the given period") {
		t.Errorf("unexpected message: %#v", res.Messages[0].Content)
	}
}
