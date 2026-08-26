package util

import (
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/model/main/common/filesource"
)

func FileSourceLabel(source filesource.FileSource) *widget.Text {
	switch source {
	case filesource.WebInterface:
		return widget.T("Web upload")
	case filesource.PWAOSOpen:
		return widget.T("Open with")
	case filesource.URLImport:
		return widget.T("URL import")
	case filesource.WebDAV:
		return widget.T("WebDAV")
	case filesource.SystemExtraction:
		return widget.T("System extraction")
	case filesource.UnknownLegacy:
		fallthrough
	default:
		return widget.T("Unknown")
	}
}
