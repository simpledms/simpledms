package mcp

type SearchFilesInput struct {
	Query           string                `json:"query,omitempty"`
	Sort            string                `json:"sort,omitempty" jsonschema:"newestFirst, oldestFirst, name, or rank"`
	TagIDs          []string              `json:"tag_ids,omitempty"`
	DocumentTypeID  string                `json:"document_type_id,omitempty"`
	PropertyFilters []PropertyFilterInput `json:"property_filters,omitempty"`
	Offset          int                   `json:"offset,omitempty"`
	Limit           *int                  `json:"limit,omitempty"`
}
