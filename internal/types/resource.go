package types

type ResourceLink struct {
	URI      string `json:"uri"`
	Label    string `json:"label,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

type DualToolResult struct {
	Preview     string `json:"preview"`
	RowCount    int    `json:"rowCount"`
	TotalCount  *int   `json:"totalCount,omitempty"`
	ResourceURI string `json:"resourceUri"`
}
