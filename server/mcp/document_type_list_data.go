package mcp

type DocumentTypeListData struct {
	DocumentTypes []DocumentTypeSummary `json:"document_types"`
	HasMore       bool                  `json:"has_more"`
	NextOffset    *int                  `json:"next_offset,omitempty"`
}
