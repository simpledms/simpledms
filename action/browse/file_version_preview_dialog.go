package browse

import (
	"net/http"
	"strconv"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileVersionPreviewDialogData struct {
	FileID        string
	VersionNumber string
	PreviewTab    string
}

type FileVersionPreviewDialog struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileVersionPreviewDialog(infra *common.Infra, actions *Actions) *FileVersionPreviewDialog {
	config := actionx.NewConfig(actions.Route("file-version-preview-dialog"), true)
	return &FileVersionPreviewDialog{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileVersionPreviewDialog) Data(fileID, versionNumber string, previewTab ...string) *FileVersionPreviewDialogData {
	activeTab := ""
	if len(previewTab) > 0 {
		activeTab = previewTab[0]
	}
	return &FileVersionPreviewDialogData{
		FileID:        fileID,
		VersionNumber: versionNumber,
		PreviewTab:    activeTab,
	}
}

func (qq *FileVersionPreviewDialog) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileVersionPreviewDialogData](rw, req, ctx)
	if err != nil {
		return err
	}

	if data.VersionNumber == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "missing version number")
	}

	versionInt, err := strconv.Atoi(data.VersionNumber)
	if err != nil {
		return e.NewHTTPErrorf(http.StatusBadRequest, "invalid version number")
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)
	versionx, err := filex.Data.QueryFileVersions().
		Where(fileversion.VersionNumber(versionInt)).
		WithStoredFile().
		Only(ctx)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusNotFound, "version not found")
		}
		return err
	}

	storedFile := versionx.Edges.StoredFile
	preview, hasPreviewTabs, err := qq.actions.FilePreviewPartial.PreviewWidget(
		ctx,
		filex,
		storedFile,
		data.VersionNumber,
		"",
		data.PreviewTab,
	)
	if err != nil {
		return err
	}
	var headerActions []widget.IWidget
	if !hasPreviewTabs {
		filename := storedFile.Filename
		if filename == "" {
			filename = filex.Data.Name
		}
		headerActions = []widget.IWidget{
			&widget.Link{
				Href:      qq.actions.FilePreviewPartial.originalDownloadURL(ctx, filex, data.VersionNumber),
				IsNoColor: true,
				Filename:  filename,
				Child: &widget.Button{
					Icon:      widget.NewIcon("download"),
					Label:     widget.T("Download"),
					StyleType: widget.ButtonStyleTypeText,
				},
			},
		}
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		&widget.Dialog{
			Layout:        widget.DialogLayoutStable,
			Width:         widget.DialogWidthWide,
			Headline:      widget.T("Version preview"),
			HeaderActions: headerActions,
			IsOpenOnLoad:  true,
			Child:         preview,
		},
	)
}
