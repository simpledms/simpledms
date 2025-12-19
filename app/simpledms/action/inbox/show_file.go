package inbox

import (
	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/model"
	"github.com/simpledms/simpledms/app/simpledms/ui/partial"
	"github.com/simpledms/simpledms/app/simpledms/ui/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ShowFileData struct {
	FileID string
}

type ShowFile struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

type ShowFileState struct {
	// ListFilesState
	ActiveTab string `url:"tab,omitempty"`
}

func NewShowFile(infra *common.Infra, actions *Actions) *ShowFile {
	return &ShowFile{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("show-file"),
			true,
		),
	}
}

func (qq *ShowFile) Data(fileID string) *ShowFileData {
	return &ShowFileData{
		FileID: fileID,
	}
}

func (qq *ShowFile) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ShowFileData](rw, req, ctx)
	if err != nil {
		return err
	}

	state, err := autil.FormData[PageState](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, state, filex),
	)
}

func (qq *ShowFile) WidgetHandler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context, filex *model.File) *wx.DetailsWithSheet {
	state := autil.StateX[PageState](rw, req)
	return qq.Widget(ctx, state, filex)
}

func (qq *ShowFile) Widget(
	ctx ctxx.Context,
	state *PageState,
	filex *model.File,
) *wx.DetailsWithSheet {
	showFileTabs := qq.actions.ShowFileTabs.Widget(
		ctx,
		state,
		filex.Data.PublicID.String(),
		filex,
	)
	return &wx.DetailsWithSheet{
		AppBar: partial.NewFullscreenDialogAppBar(
			wx.Tuf("%s", filex.Data.Name),
			route.InboxRootWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
			[]wx.IWidget{
				&wx.IconButton{
					// TODO other icon if already open or hide...
					Icon: "description", // right_panel_open, clarify, tune, description, info, ...?
					HTMXAttrs: wx.HTMXAttrs{
						DialogID: qq.SideSheetID(),
					},
				},
			},
		),
		Child: &wx.Column{
			Children: &wx.FilePreview{
				FileURL:  route.DownloadInline(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String()),
				Filename: filex.Data.Name,
				MimeType: filex.CurrentVersion(ctx).Data.MimeType,
			},
		},
		SideSheet: &wx.Dialog{
			Widget: wx.Widget[wx.Dialog]{
				ID: qq.SideSheetID(),
			},
			Headline:                        wx.T("Details"),
			IsOpenOnLoadOnExtraLargeScreens: true,
			// allows for quick back and forth on mobile devices
			KeepInDOMOnClose: true,
			Layout:           wx.DialogLayoutSideSheet,
			Child:            showFileTabs,
		},
	}
}

func (qq *ShowFile) SideSheetID() string {
	return "inboxShowFileSideSheet"
}
