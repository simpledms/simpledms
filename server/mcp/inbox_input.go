package mcp

type InboxInput struct {
	Query   string   `json:"query,omitempty"`
	Sort    string   `json:"sort,omitempty" jsonschema:"newestFirst, oldestFirst, name, or rank"`
	Sources []string `json:"sources,omitempty"`
	Offset  int      `json:"offset,omitempty"`
	Limit   *int     `json:"limit,omitempty"`
}
