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

func DownloadWithVersion(tenantID, spaceID, fileID, versionNumber string) string {
	return fmt.Sprintf("/org/%s/space/%s/download/%s?version=%s", tenantID, spaceID, fileID, url.QueryEscape(versionNumber))
}

func DownloadInlineWithVersion(tenantID, spaceID, fileID, versionNumber string) string {
	return fmt.Sprintf("/org/%s/space/%s/download/%s?inline=1&version=%s", tenantID, spaceID, fileID, url.QueryEscape(versionNumber))
}
