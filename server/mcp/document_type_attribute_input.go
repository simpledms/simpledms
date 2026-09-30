package mcp

type DocumentTypeAttributeInput struct {
	DocumentTypeID string `json:"document_type_id"`
	TagID          string `json:"tag_id,omitempty"`
	PropertyID     string `json:"property_id,omitempty"`
}
