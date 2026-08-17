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

func PreviewPDFRoute() string {
	return "GET /org/{tenant_id}/space/{space_id}/preview/{file_id}/pdf"
}

func PreviewPDFDownloadRoute() string {
	return "GET /org/{tenant_id}/space/{space_id}/preview/{file_id}/pdf/download"
}

func PreviewPDF(tenantID, spaceID, fileID string) string {
	return fmt.Sprintf("/org/%s/space/%s/preview/%s/pdf", tenantID, spaceID, fileID)
}

func PreviewPDFWithVersion(tenantID, spaceID, fileID, versionNumber string) string {
	return fmt.Sprintf("%s?version=%s", PreviewPDF(tenantID, spaceID, fileID), url.QueryEscape(versionNumber))
}

func PreviewPDFDownload(tenantID, spaceID, fileID string) string {
	return fmt.Sprintf("/org/%s/space/%s/preview/%s/pdf/download", tenantID, spaceID, fileID)
}

func PreviewPDFDownloadWithVersion(tenantID, spaceID, fileID, versionNumber string) string {
	return fmt.Sprintf("%s?version=%s", PreviewPDFDownload(tenantID, spaceID, fileID), url.QueryEscape(versionNumber))
}

func OriginalSourceRoute() string {
	return "GET /org/{tenant_id}/space/{space_id}/preview/{file_id}/original"
}

func OriginalSource(tenantID, spaceID, fileID string) string {
	return fmt.Sprintf("/org/%s/space/%s/preview/%s/original", tenantID, spaceID, fileID)
}

func OriginalSourceWithVersion(tenantID, spaceID, fileID, versionNumber string) string {
	return fmt.Sprintf("%s?version=%s", OriginalSource(tenantID, spaceID, fileID), url.QueryEscape(versionNumber))
}
