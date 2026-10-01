package mcp

type DocumentTypeData struct {
	DocumentTypeSummary
	Attributes []DocumentTypeAttributeData `json:"attributes"`
}
