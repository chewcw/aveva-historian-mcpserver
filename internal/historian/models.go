package historian

// ODataResponse wraps an OData JSON envelope with a generic value array.
type ODataResponse[T any] struct {
	Context string `json:"@odata.context"`
	Count   *int   `json:"@odata.count,omitempty"`
	Value   []T    `json:"value"`
}

// Tag represents a historian tag/point.
type Tag struct {
	TagName           string   `json:"TagName"`
	FQN               string   `json:"FQN"`
	TagType           string   `json:"TagType"`
	EngUnit           string   `json:"EngUnit,omitempty"`
	Source            string   `json:"Source,omitempty"`
	EngUnitMax        *float64 `json:"EngUnitMax,omitempty"`
	EngUnitMin        *float64 `json:"EngUnitMin,omitempty"`
	InterpolationType string   `json:"InterpolationType,omitempty"`
	MessageOff        string   `json:"MessageOff,omitempty"`
	MessageOn         string   `json:"MessageOn,omitempty"`
	Alias             string   `json:"Alias,omitempty"`
	Description       string   `json:"Description,omitempty"`
	Minutely          string   `json:"Minutely@odata.navigationLink,omitempty"`
	Hourly            string   `json:"Hourly@odata.navigationLink,omitempty"`
	Daily             string   `json:"Daily@odata.navigationLink,omitempty"`
}

// ProcessValue represents a timestamped process value.
type ProcessValue struct {
	FQN        string   `json:"FQN"`
	DateTime   string   `json:"DateTime"`
	Value      *float64 `json:"Value,omitempty"`
	OpcQuality *int     `json:"OpcQuality,omitempty"`
	Text       *string  `json:"Text,omitempty"`
	Unit       string   `json:"Unit,omitempty"`
}

// AnalogSummaryValue represents an analog summary over a time range.
type AnalogSummaryValue struct {
	FQN           string   `json:"FQN"`
	StartDateTime string   `json:"StartDateTime"`
	EndDateTime   string   `json:"EndDateTime"`
	RetrievalMode string   `json:"RetrievalMode,omitempty"`
	Resolution    *int     `json:"Resolution,omitempty"`
	SliceBy       string   `json:"SliceBy,omitempty"`
	SliceByValue  string   `json:"SliceByValue,omitempty"`
	OPCQuality    *int     `json:"OPCQuality,omitempty"`
	PercentGood   *float64 `json:"PercentGood,omitempty"`
	First         *float64 `json:"First,omitempty"`
	FirstDateTime string   `json:"FirstDateTime,omitempty"`
	Last          *float64 `json:"Last,omitempty"`
	LastDateTime  string   `json:"LastDateTime,omitempty"`
	Minimum       *float64 `json:"Minimum,omitempty"`
	MinDateTime   string   `json:"MinDateTime,omitempty"`
	Maximum       *float64 `json:"Maximum,omitempty"`
	MaxDateTime   string   `json:"MaxDateTime,omitempty"`
	Average       *float64 `json:"Average,omitempty"`
	StdDev        *float64 `json:"StandardDeviation,omitempty"`
	Integral      *float64 `json:"Integral,omitempty"`
	Count         *int     `json:"Count,omitempty"`
}
