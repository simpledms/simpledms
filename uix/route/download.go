package route

import (
	"fmt"
)

func DownloadRoute() string {
	return "GET /org/{tenant_id}/space/{space_id}/download/{file_id}"
}

func Download(tenantID, spaceID, fileID string) string {
	return fmt.Sprintf("/org/%s/space/%s/download/%s", tenantID, spaceID, fileID)
}

func DownloadInline(tenantID, spaceID, fileID string) string {
	return fmt.Sprintf("/org/%s/space/%s/download/%s?inline=1", tenantID, spaceID, fileID)
}
