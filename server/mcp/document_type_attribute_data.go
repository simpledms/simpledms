package mcp

type DocumentTypeAttributeData struct {
	Type         string `json:"type"`
	Name         string `json:"name,omitempty"`
	TagID        string `json:"tag_id,omitempty"`
	PropertyID   string `json:"property_id,omitempty"`
	IsNameGiving bool   `json:"is_name_giving"`
	IsProtected  bool   `json:"is_protected"`
	IsDisabled   bool   `json:"is_disabled"`
	IsRequired   bool   `json:"is_required"`
}
