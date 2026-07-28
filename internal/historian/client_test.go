package historian

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientGetSuccess(t *testing.T) {
	var gotPath, gotAccept string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[{"Id":1,"TagName":"T1","FQN":"T1","TagType":"AI"}]}`))
	}))
	defer ts.Close()

	c := New(ts.URL, "user", "pass", "/Historian/v2", nil)
	var resp ODataResponse[Tag]
	if err := c.Get(context.Background(), "Tags", "$top=10", &resp); err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if gotPath != "/Historian/v2/Tags" {
		t.Errorf("path = %q, want /Historian/v2/Tags", gotPath)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept header = %q, want application/json", gotAccept)
	}
	if len(resp.Value) != 1 || resp.Value[0].TagName != "T1" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestClientGetNon2xx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer ts.Close()

	c := New(ts.URL, "user", "pass", "/Historian/v2", nil)
	var dest json.RawMessage
	err := c.Get(context.Background(), "Tags", "", &dest)
	if err == nil {
		t.Fatal("expected error for 400, got nil")
	}
	t.Logf("got expected error: %v", err)
}

func TestClientGetCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// wait for context cancellation
		<-r.Context().Done()
	}))
	defer ts.Close()

	c := New(ts.URL, "user", "pass", "/Historian/v2", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	var dest json.RawMessage
	err := c.Get(ctx, "Tags", "", &dest)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	t.Logf("got expected error: %v", err)
}
