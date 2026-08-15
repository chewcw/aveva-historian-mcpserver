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

// Events JSON uses lowercase keys (id, eventtime, ...) — see historian.Event tags.
func TestAlarmReviewHappyPath(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[
			{"id":"a1","eventtime":"2024-01-01T12:00:00Z","type":"Alarm.Set","severity":1,"isalarm":true,"alarm_acknowledged":false,"source_name":"P-101A","alarm_condition":"Hi.Hi","namespace":"Area1"},
			{"id":"a2","eventtime":"2024-01-01T13:00:00Z","type":"Alarm.Clear","severity":3,"isalarm":true,"alarm_acknowledged":true,"source_name":"P-101A","alarm_condition":"Hi.Hi","namespace":"Area1"}
		]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	store := dataserver.NewStore(time.Minute)
	handler := handleAlarmReview(client, store, "http://localhost:8199", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(res.Messages) != 2 {
		t.Fatalf("want 2 messages, got %d", len(res.Messages))
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("message 0 not TextContent: %#v", res.Messages[0].Content)
	}
	if !strings.Contains(tc.Text, "## Alarm Review") {
		t.Errorf("preview missing header: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "(last 24h, now, default)") {
		t.Errorf("preview missing window defaults: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "Unacknowledged") {
		t.Errorf("preview missing unacknowledged section: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "P-101A") {
		t.Errorf("preview missing alarm source: %s", tc.Text)
	}
	rl, ok := res.Messages[1].Content.(*mcp.ResourceLink)
	if !ok || !strings.HasPrefix(rl.URI, "http://localhost:8199/resources/") {
		t.Errorf("missing resource link: %#v", res.Messages[1].Content)
	}
}

func TestAlarmReviewExplicitWindowNoDefaultNote(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[{"id":"a1","eventtime":"2024-01-01T12:00:00Z","type":"Alarm.Set","severity":2,"isalarm":true,"alarm_acknowledged":false}]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	handler := handleAlarmReview(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"start_date_time": "2024-01-01T00:00:00Z", "end_date_time": "2024-01-02T00:00:00Z",
			"severity": "2", "acknowledged": "false",
		}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("message 0 not TextContent: %#v", res.Messages[0].Content)
	}
	if strings.Contains(tc.Text, "default)") {
		t.Errorf("explicit window should have no default note: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "| 2 Major | 1 |") {
		t.Errorf("severity count missing: %s", tc.Text)
	}
}

func TestAlarmReviewBadSeverity(t *testing.T) {
	client := historian.New("http://unused", "", "", "", nil)
	handler := handleAlarmReview(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{"severity": "9"}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "severity must be 1-4") {
		t.Errorf("unexpected message: %#v", res.Messages[0].Content)
	}
}

func TestAlarmReviewNoAlarms(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	handler := handleAlarmReview(client, nil, "", discardLogger(), 1_048_576)
	res, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(tc.Text, "No alarms in the given window") {
		t.Errorf("unexpected message: %#v", res.Messages[0].Content)
	}
}

func TestAlarmReviewFiltersReachQuery(t *testing.T) {
	var rawQuery string
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[{"id":"a1","eventtime":"2024-01-01T12:00:00Z","type":"Alarm.Set","severity":2,"isalarm":true,"alarm_acknowledged":false}]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	handler := handleAlarmReview(client, nil, "", discardLogger(), 1_048_576)
	if _, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"start_date_time": "2024-01-01T00:00:00Z",
			"end_date_time":   "2024-01-02T00:00:00Z",
			"severity":        "2",
			"namespace":       "Area1",
			"acknowledged":    "false",
		}},
	}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	for _, want := range []string{"Severity", "Namespace", "Alarm_Acknowledged"} {
		if !strings.Contains(rawQuery, want) {
			t.Errorf("query missing %s: %s", want, rawQuery)
		}
	}
}

func TestAlarmReviewFiltersAbsentWhenNotGiven(t *testing.T) {
	var rawQuery string
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[{"id":"a1","eventtime":"2024-01-01T12:00:00Z","type":"Alarm.Set","severity":2,"isalarm":true,"alarm_acknowledged":false}]}`))
	}))
	defer mock.Close()

	client := historian.New(mock.URL, "", "", "", nil)
	handler := handleAlarmReview(client, nil, "", discardLogger(), 1_048_576)
	if _, err := handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{Arguments: map[string]string{
			"start_date_time": "2024-01-01T00:00:00Z",
			"end_date_time":   "2024-01-02T00:00:00Z",
		}},
	}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	for _, absent := range []string{"Severity", "Namespace", "Alarm_Acknowledged"} {
		if strings.Contains(rawQuery, absent) {
			t.Errorf("query must not contain %s: %s", absent, rawQuery)
		}
	}
}
