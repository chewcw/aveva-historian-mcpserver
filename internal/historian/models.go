package historian

// ODataResponse wraps an OData JSON envelope with a generic value array.
type ODataResponse[T any] struct {
	Context string `json:"@odata.context"`
	Count   *int   `json:"@odata.count,omitempty"`
	Value   []T    `json:"value"`
}

// Tag represents a historian tag/point.
type Tag struct {
	ID          int    `json:"Id"`
	TagName     string `json:"TagName"`
	FQN         string `json:"FQN"`
	TagType     string `json:"TagType"`
	Unit        string `json:"Unit,omitempty"`
	Description string `json:"Description,omitempty"`
}

// ProcessValue represents a timestamped process value.
type ProcessValue struct {
	FQN      string  `json:"FQN"`
	DateTime string  `json:"DateTime"`
	Value    float64 `json:"Value,omitempty"`
	Quality  string  `json:"Quality,omitempty"`
	TagPath  string  `json:"TagPath,omitempty"`
}

// AnalogSummaryValue represents an analog summary over a time range.
type AnalogSummaryValue struct {
	FQN            string   `json:"FQN"`
	StartDateTime  string   `json:"StartDateTime"`
	EndDateTime    string   `json:"EndDateTime"`
	RetrievalMode  string   `json:"RetrievalMode,omitempty"`
	Resolution     *int     `json:"Resolution,omitempty"`
	SliceBy        string   `json:"SliceBy,omitempty"`
	SliceByValue   string   `json:"SliceByValue,omitempty"`
	OPCQuality     *int     `json:"OPCQuality,omitempty"`
	PercentGood    *float64 `json:"PercentGood,omitempty"`
	First          *float64 `json:"First,omitempty"`
	FirstDateTime  string   `json:"FirstDateTime,omitempty"`
	Last           *float64 `json:"Last,omitempty"`
	LastDateTime   string   `json:"LastDateTime,omitempty"`
	Minimum        *float64 `json:"Minimum,omitempty"`
	MinDateTime    string   `json:"MinDateTime,omitempty"`
	Maximum        *float64 `json:"Maximum,omitempty"`
	MaxDateTime    string   `json:"MaxDateTime,omitempty"`
	Average        *float64 `json:"Average,omitempty"`
	StdDev         *float64 `json:"StandardDeviation,omitempty"`
	Integral       *float64 `json:"Integral,omitempty"`
	Count          *int     `json:"Count,omitempty"`
}
