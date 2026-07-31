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

// Event represents an event or alarm record from the Historian Events endpoint.
type Event struct {
	ID                       string `json:"id"`
	EventTime                string `json:"eventtime"`
	Type                     string `json:"type"`
	ReceivedTime             string `json:"receivedtime"`
	SourceName               string `json:"source_name"`
	Namespace                string `json:"namespace"`
	IsAlarm                  *bool  `json:"isalarm,omitempty"`
	Priority                 *int   `json:"priority,omitempty"`
	Severity                 *int   `json:"severity,omitempty"`
	Comment                  string `json:"comment,omitempty"`
	EventTimeUTCOffsetMins   *int   `json:"eventtimeutcoffsetmins,omitempty"`
	ValueString              string `json:"valuestring,omitempty"`
	PreviousValueString      string `json:"previousvaluestring,omitempty"`
	InTouchType              string `json:"intouchtype,omitempty"`
	AlarmID                  string `json:"alarm_id,omitempty"`
	AlarmClass               string `json:"alarm_class,omitempty"`
	AlarmType                string `json:"alarm_type,omitempty"`
	AlarmCondition           string `json:"alarm_condition,omitempty"`
	AlarmState               string `json:"alarm_state,omitempty"`
	AlarmInAlarm             *bool  `json:"alarm_inalarm,omitempty"`
	AlarmAcknowledged        *bool  `json:"alarm_acknowledged,omitempty"`
	AlarmIsSilenced          *bool  `json:"alarm_issilenced,omitempty"`
	AlarmIsShelved           *bool  `json:"alarm_isshelved,omitempty"`
	AlarmDurationMs          *int   `json:"alarm_durationms,omitempty"`
	AlarmLimitString         string `json:"alarm_limitstring,omitempty"`
	AlarmOriginationTime     string `json:"alarm_originationtime,omitempty"`
	AlarmValueString         string `json:"alarm_valuestring,omitempty"`
	AlarmUnackDuration       *int   `json:"alarm_unackdurationms,omitempty"`
	AlarmShelveStartTimeUTC  string `json:"alarm_shelvestarttimeutc,omitempty"`
	AlarmShelveEndTimeUTC    string `json:"alarm_shelveendtimeutc,omitempty"`
	AlarmShelveReason        string `json:"alarm_shelvereason,omitempty"`
	AlarmShelveUserLogin     string `json:"alarm_shelveuserlogin,omitempty"`
	ProviderNodeName         string `json:"provider_nodename,omitempty"`
	ProviderSystem           string `json:"provider_system,omitempty"`
	ProviderApplicationName  string `json:"provider_applicationname,omitempty"`
	ProviderSystemVersion    string `json:"provider_systemversion,omitempty"`
	SourceProcessVariable    string `json:"source_processvariable,omitempty"`
	SourceConditionVariable  string `json:"source_conditionvariable,omitempty"`
	SourceObject             string `json:"source_object,omitempty"`
	SourceHierarchicalObject string `json:"source_hierarchicalobject,omitempty"`
	SourceArea               string `json:"source_area,omitempty"`
	SourceHierarchicalArea   string `json:"source_hierarchicalarea,omitempty"`
}
