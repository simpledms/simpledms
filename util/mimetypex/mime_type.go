package mimetypex

import (
	"path/filepath"
	"strings"
)

// Resolve returns a stored MIME type or infers one from the filename.
func Resolve(mimeType, filename string) string {
	if strings.TrimSpace(mimeType) != "" {
		return mimeType
	}
	if strings.EqualFold(filepath.Ext(filename), ".txt") {
		return "text/plain; charset=utf-8"
	}
	return "application/octet-stream"
}
