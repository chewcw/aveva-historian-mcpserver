package endpoints

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
)

func TestGetTagsQuery(t *testing.T) {
	var rawQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer ts.Close()

	client := historian.New(ts.URL, "u", "p", nil)
	result, err := GetTags(context.Background(), client, "startswith(FQN,'CDE')", 10, 0)
	if err != nil {
		t.Fatalf("GetTags failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	want := "$filter=startswith(FQN,'CDE')&$top=10&$skip=0"
	if rawQuery != want {
		t.Errorf("raw query = %q, want %q", rawQuery, want)
	}
}
