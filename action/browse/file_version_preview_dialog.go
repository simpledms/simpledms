package browse

import (
	"net/http"
	"strconv"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileVersionPreviewDialogData struct {
	FileID        string
	VersionNumber string
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

func (qq *FileVersionPreviewDialog) Data(fileID, versionNumber string) *FileVersionPreviewDialogData {
	return &FileVersionPreviewDialogData{
		FileID:        fileID,
		VersionNumber: versionNumber,
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
	versionm := model.NewStoredFile(storedFile)
	filename := filex.Data.Name
	if versionm.Data.Filename != "" {
		filename = versionm.Data.Filename
	}
	downloadURL := route.DownloadWithVersion(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String(), data.VersionNumber)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		&wx.Dialog{
			Layout:   wx.DialogLayoutStable,
			Width:    wx.DialogWidthWide,
			Headline: wx.T("Version preview"),
			HeaderActions: []wx.IWidget{
				&wx.Link{
					Href:      downloadURL,
					IsNoColor: true,
					Filename:  filename,
					Child: &wx.Button{
						Icon:      wx.NewIcon("download"),
						Label:     wx.T("Download"),
						StyleType: wx.ButtonStyleTypeText,
					},
				},
			},
			IsOpenOnLoad: true,
			Child: &wx.FilePreview{
				FileURL:  route.DownloadInlineWithVersion(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String(), data.VersionNumber),
				Filename: filename,
				MimeType: versionm.Data.MimeType,
			},
		},
	)
}
