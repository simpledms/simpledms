package download

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	previewconversion "github.com/simpledms/simpledms/db/enttenant/previewconversion"
	tenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	previewmodel "github.com/simpledms/simpledms/model/tenant/previewconversion"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type Preview struct {
	infra *common.Infra
}

func NewPreview(infra *common.Infra) *Preview {
	return &Preview{infra: infra}
}

func (qq *Preview) PDFInlineHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	return qq.streamPDF(rw, req, ctx, false)
}

func (qq *Preview) PDFDownloadHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	return qq.streamPDF(rw, req, ctx, true)
}

func (qq *Preview) OriginalSourceHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	filex, source, err := qq.source(ctx, req.URL.Query().Get("version"), req.PathValue("file_id"))
	if err != nil {
		return err
	}
	if filex.Data.IsDirectory {
		return e.NewHTTPErrorf(http.StatusBadRequest, "cannot preview directories")
	}

	extension := strings.ToLower(filepath.Ext(source.Filename))
	mimeType := strings.ToLower(strings.TrimSpace(strings.Split(source.MimeType, ";")[0]))
	if extension != ".html" && extension != ".htm" && extension != ".xhtml" &&
		mimeType != "text/html" && mimeType != "application/xhtml+xml" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "original source preview is only available for HTML files")
	}

	openedFile, err := qq.infra.FileSystem().OpenFile(ctx, storedfilemodel.NewStoredFile(source))
	if err != nil {
		return err
	}
	defer func() {
		_ = openedFile.Close()
	}()

	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.Header().Set("Content-Disposition", "inline")
	rw.Header().Set("X-Content-Type-Options", "nosniff")
	rw.Header().Set("Content-Security-Policy", "sandbox")
	rw.WriteHeader(http.StatusOK)
	if _, err := io.Copy(rw, openedFile); err != nil {
		return e.NewHTTPErrorf(http.StatusInternalServerError, "could not read file")
	}
	return nil
}

func (qq *Preview) streamPDF(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	attachment bool,
) error {
	filex, source, err := qq.source(ctx, req.URL.Query().Get("version"), req.PathValue("file_id"))
	if err != nil {
		return err
	}
	if filex.Data.IsDirectory {
		return e.NewHTTPErrorf(http.StatusBadRequest, "cannot download directories")
	}

	conversion, err := ctx.TenantCtx().TTx.PreviewConversion.Query().
		Where(
			previewconversion.SourceStoredFileID(source.ID),
			previewconversion.StatusEQ(previewmodel.Ready),
			previewconversion.PreviewStoredFileIDNotNil(),
		).
		Only(ctx)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusNotFound, "PDF preview is not available")
		}
		return err
	}

	previewContext := tenantprivacy.DecisionContext(ctx, tenantprivacy.Allow)
	preview, err := ctx.TenantCtx().TTx.StoredFile.Get(previewContext, *conversion.PreviewStoredFileID)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusNotFound, "PDF preview is not available")
		}
		return err
	}
	if preview.CopiedToFinalDestinationAt == nil {
		return e.NewHTTPErrorf(http.StatusNotFound, "PDF preview is not available")
	}

	openedFile, err := qq.infra.FileSystem().OpenFile(ctx, storedfilemodel.NewStoredFile(preview))
	if err != nil {
		return err
	}
	defer func() {
		_ = openedFile.Close()
	}()

	filename := preview.Filename
	if filename == "" {
		filename = "preview.pdf"
	}
	rw.Header().Set("Content-Type", "application/pdf")
	rw.Header().Set("X-Content-Type-Options", "nosniff")
	if preview.Size > 0 {
		rw.Header().Set("Content-Length", strconv.FormatInt(preview.Size, 10))
	}
	if attachment {
		rw.Header().Set("Content-Disposition", fmt.Sprintf(
			"attachment; filename=\"%s\"; filename*=UTF-8''%s",
			url.QueryEscape(filename),
			url.QueryEscape(filename),
		))
	} else {
		rw.Header().Set("Content-Disposition", "inline")
	}
	rw.WriteHeader(http.StatusOK)
	if _, err := io.Copy(rw, openedFile); err != nil {
		return e.NewHTTPErrorf(http.StatusInternalServerError, "could not read file")
	}
	return nil
}

func (qq *Preview) source(
	ctx ctxx.Context,
	versionNumber string,
	fileID string,
) (*filemodel.File, *enttenant.StoredFile, error) {
	filex := qq.infra.FileRepo.GetX(ctx, fileID)
	if filex.Data.IsDirectory {
		return filex, nil, e.NewHTTPErrorf(http.StatusBadRequest, "cannot preview directories")
	}
	if versionNumber == "" {
		return filex, filex.CurrentVersion(ctx).Data, nil
	}

	versionInt, err := strconv.Atoi(versionNumber)
	if err != nil {
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, "invalid version number")
	}
	version, err := filex.Data.QueryFileVersions().
		Where(fileversion.VersionNumber(versionInt)).
		WithStoredFile().
		Only(ctx)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return nil, nil, e.NewHTTPErrorf(http.StatusNotFound, "version not found")
		}
		return nil, nil, err
	}
	return filex, version.Edges.StoredFile, nil
}
