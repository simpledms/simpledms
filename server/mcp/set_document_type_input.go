package mcp

type SetDocumentTypeInput struct {
	FileID         string `json:"file_id" jsonschema:"Public file identifier"`
	DocumentTypeID string `json:"document_type_id" jsonschema:"Public document-type identifier"`
}
