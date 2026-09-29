package mcp

type TagAssignmentData struct {
	FileID           string `json:"file_id"`
	TagID            string `json:"tag_id"`
	DirectlyAssigned bool   `json:"directly_assigned"`
	Resolved         bool   `json:"resolved"`
}
