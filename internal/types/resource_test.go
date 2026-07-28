package types

import (
	"encoding/json"
	"testing"
)

func TestDualToolResultRoundTrip(t *testing.T) {
	total := 42
	orig := DualToolResult{
		Preview:     "col1,col2\n1,2",
		RowCount:    1,
		TotalCount:  &total,
		ResourceURI: "aveva://pi/records/abc",
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got DualToolResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.RowCount != orig.RowCount {
		t.Errorf("RowCount = %d, want %d", got.RowCount, orig.RowCount)
	}
	if got.ResourceURI != orig.ResourceURI {
		t.Errorf("ResourceURI = %q, want %q", got.ResourceURI, orig.ResourceURI)
	}
	if *got.TotalCount != *orig.TotalCount {
		t.Errorf("TotalCount = %d, want %d", *got.TotalCount, *orig.TotalCount)
	}
}

func TestResourceLinkFormat(t *testing.T) {
	orig := ResourceLink{
		URI:      "aveva://pi/records/xyz",
		Label:    "My Record",
		MimeType: "application/json",
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got ResourceLink
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.URI != orig.URI {
		t.Errorf("URI = %q, want %q", got.URI, orig.URI)
	}
}
