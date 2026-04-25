package inbox

import (
	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/uix/partial"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
)

type InboxRootPage struct {
	infra   *common.Infra
	actions *Actions

	// inboxDirInfo *ent.FileInfo
}

func NewInboxRootPage(infra *common.Infra, actions *Actions) *InboxRootPage {
	return &InboxRootPage{
		infra:   infra,
		actions: actions,
		// infra.UnsafeDB().FileInfo.Query().Where(fileinfo.FullPath(infra.InboxPath())).OnlyX(context.Background()),
	}
}

func (qq *InboxRootPage) Handler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	fabs := []*widget.FloatingActionButton{
		{
			Icon: "upload_file",
			HTMXAttrs: widget.HTMXAttrs{
				HxPost: qq.actions.Browse.FileUploadDialogPartial.Endpoint(),
				HxVals: util.JSON(qq.actions.Browse.FileUploadDialogPartial.Data(
					ctx.SpaceCtx().SpaceRootDir().PublicID.String(),
					true,
				)),
				LoadInPopover: true,
			},
			Child: []widget.IWidget{
				widget.NewIcon("upload_file"),
				widget.T("Upload file"),
			},
		},
	}

	var viewx renderable.Renderable
	viewx = &widget.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, qq.infra, "inbox", fabs),
		Content:    qq.actions.InboxPage.WidgetHandler(rw, req, ctx, ""),
	}

	renderFullPage := false
	if req.Header.Get("HX-Request") == "" {
		renderFullPage = true
	}

	if renderFullPage {
		viewx = partial.NewBase(widget.T("Inbox"), viewx)
	}

	return qq.infra.Renderer().Render(rw, ctx, viewx)
}
