package historian

import (
	"encoding/json"
	"testing"
)

func TestODataResponseUnmarshalTags(t *testing.T) {
	raw := `{
		"@odata.context": "https://server/odata/$metadata#Tags",
		"@odata.count": 2,
		"value": [
			{"TagName": "SINUSOID", "FQN": "SINUSOID", "TagType": "AI", "EngUnit": "DEG C", "Source": "Manual", "EngUnitMax": 100.0, "EngUnitMin": 0.0, "InterpolationType": "Linear", "IntegralDivisor": 1.0, "RolloverValue": 999.9, "MessageOff": "LOW", "MessageOn": "HIGH", "Alias": "SINE", "Location": "Lab", "Description": "Sine wave tag"}
		]
	}`
	var resp ODataResponse[Tag]
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Context != "https://server/odata/$metadata#Tags" {
		t.Errorf("Context = %q, want %q", resp.Context, "https://server/odata/$metadata#Tags")
	}
	if resp.Count == nil || *resp.Count != 2 {
		t.Errorf("Count = %v, want 2", resp.Count)
	}
	if len(resp.Value) != 1 {
		t.Fatalf("len(Value) = %d, want 1", len(resp.Value))
	}
	tag := resp.Value[0]
	if tag.TagName != "SINUSOID" {
		t.Errorf("TagName = %q, want %q", tag.TagName, "SINUSOID")
	}
	if tag.FQN != "SINUSOID" {
		t.Errorf("FQN = %q, want %q", tag.FQN, "SINUSOID")
	}
	if tag.TagType != "AI" {
		t.Errorf("TagType = %q, want %q", tag.TagType, "AI")
	}
	if tag.EngUnit != "DEG C" {
		t.Errorf("EngUnit = %q, want %q", tag.EngUnit, "DEG C")
	}
	if tag.Source != "Manual" {
		t.Errorf("Source = %q, want %q", tag.Source, "Manual")
	}
	if tag.EngUnitMax == nil || *tag.EngUnitMax != 100.0 {
		t.Errorf("EngUnitMax = %v, want 100.0", tag.EngUnitMax)
	}
	if tag.EngUnitMin == nil || *tag.EngUnitMin != 0.0 {
		t.Errorf("EngUnitMin = %v, want 0.0", tag.EngUnitMin)
	}
	if tag.InterpolationType != "Linear" {
		t.Errorf("InterpolationType = %q, want %q", tag.InterpolationType, "Linear")
	}
	if tag.MessageOff != "LOW" {
		t.Errorf("MessageOff = %q, want %q", tag.MessageOff, "LOW")
	}
	if tag.MessageOn != "HIGH" {
		t.Errorf("MessageOn = %q, want %q", tag.MessageOn, "HIGH")
	}
	if tag.Alias != "SINE" {
		t.Errorf("Alias = %q, want %q", tag.Alias, "SINE")
	}
	if tag.Description != "Sine wave tag" {
		t.Errorf("Description = %q, want %q", tag.Description, "Sine wave tag")
	}
}

func TestODataResponseUnmarshalProcessValues(t *testing.T) {
	raw := `{
		"@odata.context": "https://server/odata/$metadata#InterpolatedData",
		"value": [
			{"FQN": "SINUSOID", "DateTime": "2025-01-15T12:00:00Z", "Value": 1.23, "OpcQuality": 192, "Text": "1.23", "Unit": "DEG C"}
		]
	}`
	var resp ODataResponse[ProcessValue]
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Context != "https://server/odata/$metadata#InterpolatedData" {
		t.Errorf("Context = %q", resp.Context)
	}
	if resp.Count != nil {
		t.Errorf("Count should be nil, got %v", *resp.Count)
	}
	if len(resp.Value) != 1 {
		t.Fatalf("len(Value) = %d, want 1", len(resp.Value))
	}
	pv := resp.Value[0]
	if pv.FQN != "SINUSOID" {
		t.Errorf("FQN = %q, want %q", pv.FQN, "SINUSOID")
	}
	if pv.DateTime != "2025-01-15T12:00:00Z" {
		t.Errorf("DateTime = %q", pv.DateTime)
	}
	if pv.Value == nil || *pv.Value != 1.23 {
		t.Errorf("Value = %v, want 1.23", pv.Value)
	}
	if pv.OpcQuality == nil || *pv.OpcQuality != 192 {
		t.Errorf("OpcQuality = %v, want 192", pv.OpcQuality)
	}
	if pv.Text == nil || *pv.Text != "1.23" {
		t.Errorf("Text = %v, want 1.23", pv.Text)
	}
	if pv.Unit != "DEG C" {
		t.Errorf("Unit = %q, want %q", pv.Unit, "DEG C")
	}
}

func TestODataResponseUnmarshalAnalogSummary(t *testing.T) {
	raw := `{
		"@odata.context": "https://server/odata/$metadata#SummaryData",
		"value": [
			{
				"FQN": "SINUSOID",
				"StartDateTime": "2025-01-15T00:00:00Z",
				"EndDateTime": "2025-01-15T23:59:59Z",
				"Minimum": 0.1,
				"Maximum": 9.9,
				"Average": 5.0,
				"StandardDeviation": 2.5,
				"Count": 86400,
				"TagPath": "\\PATH\\SINUSOID"
			}
		]
	}`
	var resp ODataResponse[AnalogSummaryValue]
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Value) != 1 {
		t.Fatalf("len(Value) = %d, want 1", len(resp.Value))
	}
	sv := resp.Value[0]
	if sv.FQN != "SINUSOID" {
		t.Errorf("FQN = %q", sv.FQN)
	}
	if sv.StartDateTime != "2025-01-15T00:00:00Z" {
		t.Errorf("StartDateTime = %q", sv.StartDateTime)
	}
	if sv.EndDateTime != "2025-01-15T23:59:59Z" {
		t.Errorf("EndDateTime = %q", sv.EndDateTime)
	}
	if sv.Minimum == nil || *sv.Minimum != 0.1 {
		t.Errorf("Minimum = %v", sv.Minimum)
	}
	if sv.Maximum == nil || *sv.Maximum != 9.9 {
		t.Errorf("Maximum = %v", sv.Maximum)
	}
	if sv.Average == nil || *sv.Average != 5.0 {
		t.Errorf("Average = %v", sv.Average)
	}
	if sv.StdDev == nil || *sv.StdDev != 2.5 {
		t.Errorf("StdDev = %v", sv.StdDev)
	}
	if sv.Count == nil || *sv.Count != 86400 {
		t.Errorf("Count = %d", sv.Count)
	}
}

func TestODataResponseEmptyValue(t *testing.T) {
	raw := `{
		"@odata.context": "https://server/odata/$metadata#Tags",
		"value": []
	}`
	var resp ODataResponse[Tag]
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Context != "https://server/odata/$metadata#Tags" {
		t.Errorf("Context = %q", resp.Context)
	}
	if resp.Count != nil {
		t.Errorf("Count should be nil for absent @odata.count, got %v", *resp.Count)
	}
	if resp.Value == nil {
		t.Error("Value should be non-nil empty slice, got nil")
	}
	if len(resp.Value) != 0 {
		t.Errorf("len(Value) = %d, want 0", len(resp.Value))
	}
}
