package inbox

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/uix/partial"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FilePartialData struct {
	FileID string
}

type FilePartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

type FilePartialState struct {
	// ListFilesPartialState
	ActiveTab  string `url:"tab,omitempty"`
	PreviewTab string `url:"preview_tab,omitempty"`
}

func NewFilePartial(infra *common.Infra, actions *Actions) *FilePartial {
	return &FilePartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("file-partial"),
			true,
		),
	}
}

func (qq *FilePartial) Data(fileID string) *FilePartialData {
	return &FilePartialData{
		FileID: fileID,
	}
}

func (qq *FilePartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FilePartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	state, err := autil.FormData[InboxPageState](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	view, err := qq.Widget(ctx, state, filex)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		view,
	)
}

func (qq *FilePartial) WidgetHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	filex *filemodel.File,
) (*widget.DetailsWithSheet, error) {
	state := autil.StateX[InboxPageState](rw, req)
	return qq.Widget(ctx, state, filex)
}

func (qq *FilePartial) Widget(
	ctx ctxx.Context,
	state *InboxPageState,
	filex *filemodel.File,
) (*widget.DetailsWithSheet, error) {
	fileTabsPartial := qq.actions.FileTabsPartial.Widget(
		ctx,
		state,
		filex.Data.PublicID.String(),
		filex,
	)
	preview, hasPreviewTabs, err := qq.actions.Browse.FilePreviewPartial.PreviewWidget(
		ctx,
		filex,
		filex.CurrentVersion(ctx).Data,
		"",
		"",
		state.PreviewTab,
	)
	if err != nil {
		return nil, err
	}
	appBarActions := []widget.IWidget{
		&widget.IconButton{
			// TODO other icon if already open or hide...
			Icon:    "description", // right_panel_open, clarify, tune, description, info, ...?
			Tooltip: widget.T("Show details"),
			HTMXAttrs: widget.HTMXAttrs{
				DialogID: qq.SideSheetID(),
			},
		},
	}
	if !hasPreviewTabs {
		appBarActions = append(appBarActions, &widget.Link{
			Href: route2.Download(
				ctx.TenantCtx().TenantID,
				ctx.SpaceCtx().SpaceID,
				filex.Data.PublicID.String(),
			),
			IsNoColor: true,
			Filename:  filex.Filename(ctx),
			Child: &widget.IconButton{
				Icon:    "download",
				Tooltip: widget.T("Download"),
			},
		})
	}
	return &widget.DetailsWithSheet{
		AppBar: partial.NewFullscreenDialogAppBar(
			widget.Tuf("%s", filex.Data.Name),
			route2.InboxRootWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
			appBarActions,
		),
		Child: &widget.Column{
			Children: preview,
		},
		SideSheet: &widget.Dialog{
			Widget: widget.Widget[widget.Dialog]{
				ID: qq.SideSheetID(),
			},
			Headline:                        widget.T("Details"),
			IsOpenOnLoadOnExtraLargeScreens: true,
			// allows for quick back and forth on mobile devices
			KeepInDOMOnClose: true,
			Layout:           widget.DialogLayoutSideSheet,
			Child:            fileTabsPartial,
		},
	}, nil
}

func (qq *FilePartial) SideSheetID() string {
	return "inboxShowFileSideSheet"
}
