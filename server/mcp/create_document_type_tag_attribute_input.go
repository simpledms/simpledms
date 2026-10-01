package mcp

type CreateDocumentTypeTagAttributeInput struct {
	DocumentTypeID string `json:"document_type_id"`
	TagID          string `json:"tag_id"`
	Name           string `json:"name"`
	IsNameGiving   bool   `json:"is_name_giving,omitempty"`
}
