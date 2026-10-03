package common

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/mimetypex"
)

func StreamDownload(
	infra *common.Infra,
	ctx ctxx.Context,
	rw httpx.ResponseWriter,
	req *httpx.Request,
	filex *filemodel.File,
	currentVersion *storedfilemodel.StoredFile,
) error {
	if filex.Data.IsDirectory {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Folders cannot be downloaded.")
	}

	f, err := infra.FileSystem().OpenFile(ctx, currentVersion)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "")
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Println(err)
		}
	}()

	isInline := req.URL.Query().Get("inline") == "1"
	if isInline {
		rw.Header().Set("Content-Disposition", "inline")
	} else {
		rw.Header().Set("Content-Disposition", fmt.Sprintf(
			"attachment; filename=\"%s\"; filename*=UTF-8''%s",
			url.QueryEscape(filex.Data.Name),
			url.QueryEscape(filex.Data.Name),
		))
	}

	mimeType := mimetypex.Resolve(currentVersion.Data.MimeType, currentVersion.Data.Filename)
	rw.Header().Set("Content-Type", mimeType)
	SetDownloadSecurityHeaders(rw.Header(), mimeType, isInline)

	rw.WriteHeader(http.StatusOK)
	_, err = io.Copy(rw, f)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "")
	}

	return nil
}

// SetDownloadSecurityHeaders must be used by every handler that serves uploaded bytes. It
// prevents active content, such as HTML or SVG scripts, from running with the viewer's
// session on the application origin.
func SetDownloadSecurityHeaders(header http.Header, mimeType string, isInline bool) {
	header.Set("X-Content-Type-Options", "nosniff")

	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	// Chromium refuses to render PDFs in sandboxed documents. The PDF viewer does not
	// execute document scripts with access to the application origin.
	if isInline && mediaType == "application/pdf" {
		return
	}
	header.Set("Content-Security-Policy", "sandbox")
}
