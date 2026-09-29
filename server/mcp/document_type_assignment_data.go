package mcp

type DocumentTypeAssignmentData struct {
	FileID       string               `json:"file_id"`
	DocumentType *DocumentTypeSummary `json:"document_type,omitempty"`
}
