package browse

import (
	"net/http"
	"os"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	previewconversion "github.com/simpledms/simpledms/db/enttenant/previewconversion"
	"github.com/simpledms/simpledms/internal/gotenberg"
	previewmodel "github.com/simpledms/simpledms/model/tenant/previewconversion"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type RetryPDFPreviewData struct {
	CurrentDirID  string
	FileID        string
	VersionNumber string
}

type RetryPDFPreviewCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewRetryPDFPreviewCmd(infra *common.Infra, actions *Actions) *RetryPDFPreviewCmd {
	return &RetryPDFPreviewCmd{
		infra:   infra,
		actions: actions,
		Config:  actionx.NewConfig(actions.Route("retry-pdf-preview-cmd"), false),
	}
}

func (qq *RetryPDFPreviewCmd) Data(currentDirID, fileID, versionNumber string) *RetryPDFPreviewData {
	return &RetryPDFPreviewData{
		CurrentDirID:  currentDirID,
		FileID:        fileID,
		VersionNumber: versionNumber,
	}
}

func (qq *RetryPDFPreviewCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[RetryPDFPreviewData](rw, req, ctx)
	if err != nil {
		return err
	}
	if !gotenberg.IsValidGotenbergURL(strings.TrimSpace(os.Getenv("SIMPLEDMS_GOTENBERG_URL"))) {
		return e.NewHTTPErrorf(http.StatusBadRequest, "PDF preview conversion is not configured")
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)
	source, err := qq.actions.FilePreviewPartial.versionSource(ctx, filex, data.VersionNumber)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusNotFound, "version not found")
		}
		return err
	}
	conversion, err := ctx.TenantCtx().TTx.PreviewConversion.Query().
		Where(previewconversion.SourceStoredFileID(source.ID)).
		Only(ctx)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusNotFound, "PDF preview is not available")
		}
		return err
	}
	if conversion.Status != previewmodel.Failed {
		return e.NewHTTPErrorf(http.StatusBadRequest, "PDF preview is not ready to retry")
	}

	err = ctx.TenantCtx().TTx.PreviewConversion.UpdateOneID(conversion.ID).
		ClearPreviewStoredFileID().
		SetStatus(previewmodel.Pending).
		SetRetryCount(0).
		ClearLastAttemptedAt().
		ClearNextAttemptAt().
		ClearProcessingStartedAt().
		ClearFailureCategory().
		Exec(ctx)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", "previewStatusChanged")
	rw.AddRenderables(widget.NewSnackbarf("PDF preview generation queued."))
	return nil
}
