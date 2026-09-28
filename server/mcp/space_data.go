package mcp

type SpaceData struct {
	TenantID        string `json:"tenant_id"`
	TenantName      string `json:"tenant_name"`
	SpaceID         string `json:"space_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	FolderMode      bool   `json:"folder_mode"`
	RootDirectoryID string `json:"root_directory_id"`
	ReadOnly        bool   `json:"read_only"`
}
