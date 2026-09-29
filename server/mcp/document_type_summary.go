package mcp

type DocumentTypeSummary struct {
	DocumentTypeID string `json:"document_type_id"`
	Name           string `json:"name"`
	IsProtected    bool   `json:"is_protected"`
	IsDisabled     bool   `json:"is_disabled"`
}
