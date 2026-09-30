package mcp

type CreateDocumentTypePropertyAttributeInput struct {
	DocumentTypeID string `json:"document_type_id"`
	PropertyID     string `json:"property_id"`
	IsNameGiving   bool   `json:"is_name_giving,omitempty"`
}
