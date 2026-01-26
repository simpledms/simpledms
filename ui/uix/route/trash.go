package route

import "fmt"

func TrashRoute() string {
	return "GET /org/{tenant_id}/space/{space_id}/trash/"
}

func TrashRouteWithSelection() string {
	return "GET /org/{tenant_id}/space/{space_id}/trash/file/{file_id}"
}

func TrashDownloadRoute() string {
	return "GET /org/{tenant_id}/space/{space_id}/trash/download/{file_id}"
}

func TrashActionsRoute() string {
	return "/trash/"
}

func TrashRoot(tenantID, spaceID string) string {
	return fmt.Sprintf("/org/%s/space/%s/trash/", tenantID, spaceID)
}

func TrashFile(tenantID, spaceID, fileID string) string {
	return fmt.Sprintf("/org/%s/space/%s/trash/file/%s", tenantID, spaceID, fileID)
}

func TrashFileWithState(data any) func(string, string, string) string {
	return WithState(TrashFile, data)
}

func TrashDownload(tenantID, spaceID, fileID string) string {
	return fmt.Sprintf("/org/%s/space/%s/trash/download/%s", tenantID, spaceID, fileID)
}

func TrashDownloadInline(tenantID, spaceID, fileID string) string {
	return fmt.Sprintf("/org/%s/space/%s/trash/download/%s?inline=1", tenantID, spaceID, fileID)
}
