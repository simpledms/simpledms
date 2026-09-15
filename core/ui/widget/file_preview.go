package widget

import (
	"strings"

	"github.com/simpledms/simpledms/util/mimetypex"
)

type FilePreview struct {
	Widget[FilePreview]
	FileURL  string
	Filename string
	// Base64Data template.HTMLAttr
	MimeType         string
	HideDownloadLink bool
}

// not in use as of 2024.10.01 because of switch to FileURL instead of Base64 encoded data
func (qq *FilePreview) GetMimeTypeForData() string {
	// value could be `text/plain; charset=utf-8`, in this case
	// space gets percent encoded in HTML and breaks the output,
	// thus remove all spaces; not sure if it does any harm for type
	// attribute, thus just in data string
	return strings.ReplaceAll(qq.GetMimeType(), " ", "")
}

// GetMimeType returns the effective MIME type used by the preview element.
func (qq *FilePreview) GetMimeType() string {
	return mimetypex.Resolve(qq.MimeType, qq.Filename)
}

func (qq *FilePreview) GetFilename() string {
	return qq.Filename
	// return filepath.Base(qq.FileURL)
}

func (qq *FilePreview) IsImage() bool {
	return strings.HasPrefix(qq.GetMimeType(), "image/")
}

func (qq *FilePreview) IsVideo() bool {
	return strings.HasPrefix(qq.GetMimeType(), "video/")
}

func (qq *FilePreview) IsAudio() bool {
	return strings.HasPrefix(qq.GetMimeType(), "audio/")
}

func (qq *FilePreview) IsPreviewable() bool {
	// TODO add more; for example go files
	mimeType := qq.GetMimeType()
	return strings.HasPrefix(mimeType, "text/") || strings.HasPrefix(mimeType, "application/pdf")
}

// TODO impl preview for archives
func (qq *FilePreview) IsArchive() bool {
	// TODO add more
	return strings.HasPrefix(qq.GetMimeType(), "application/zip")
}

func (qq *FilePreview) IsBinary() bool {
	return strings.HasPrefix(qq.GetMimeType(), "application/octet-stream")
}

func (qq *FilePreview) IsPDF() bool {
	return strings.HasPrefix(qq.GetMimeType(), "application/pdf")
}
