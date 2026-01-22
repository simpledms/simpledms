package inbox

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/ui/uix/partial"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ShowFilePartialData struct {
	FileID string
}

type ShowFilePartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

type ShowFilePartialState struct {
	// ListFilesPartialState
	ActiveTab string `url:"tab,omitempty"`
}

func NewShowFilePartial(infra *common.Infra, actions *Actions) *ShowFilePartial {
	return &ShowFilePartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("show-file"),
			true,
		),
	}
}

func (qq *ShowFilePartial) Data(fileID string) *ShowFilePartialData {
	return &ShowFilePartialData{
		FileID: fileID,
	}
}

func (qq *ShowFilePartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ShowFilePartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	state, err := autil.FormData[InboxPageState](rw, req, ctx)
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

func (qq *ShowFilePartial) WidgetHandler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context, filex *model.File) *wx.DetailsWithSheet {
	state := autil.StateX[InboxPageState](rw, req)
	return qq.Widget(ctx, state, filex)
}

func (qq *ShowFilePartial) Widget(
	ctx ctxx.Context,
	state *InboxPageState,
	filex *model.File,
) *wx.DetailsWithSheet {
	showFileTabs := qq.actions.ShowFileTabsPartial.Widget(
		ctx,
		state,
		filex.Data.PublicID.String(),
		filex,
	)
	return &wx.DetailsWithSheet{
		AppBar: partial.NewFullscreenDialogAppBar(
			wx.Tuf("%s", filex.Data.Name),
			route2.InboxRootWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
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
				FileURL:  route2.DownloadInline(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String()),
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

func (qq *ShowFilePartial) SideSheetID() string {
	return "inboxShowFileSideSheet"
}
