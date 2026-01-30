package route

import (
	"fmt"
	"net/url"
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

func DownloadWithVersion(tenantID, spaceID, fileID, versionID string) string {
	return fmt.Sprintf("/org/%s/space/%s/download/%s?version_id=%s", tenantID, spaceID, fileID, url.QueryEscape(versionID))
}

func DownloadInlineWithVersion(tenantID, spaceID, fileID, versionID string) string {
	return fmt.Sprintf("/org/%s/space/%s/download/%s?inline=1&version_id=%s", tenantID, spaceID, fileID, url.QueryEscape(versionID))
}
