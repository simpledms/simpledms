package mcp

type MetadataListInput struct {
	Offset int  `json:"offset,omitempty" jsonschema:"Result offset"`
	Limit  *int `json:"limit,omitempty" jsonschema:"Maximum results, default 50 and maximum 100"`
}
