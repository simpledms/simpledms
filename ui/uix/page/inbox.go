package page

import (
	"github.com/simpledms/simpledms/action/inbox"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/httpx"
)

type Inbox struct {
	infra   *common.Infra
	actions *inbox.Actions

	// inboxDirInfo *ent.FileInfo
}

func NewInbox(infra *common.Infra, actions *inbox.Actions) *Inbox {
	return &Inbox{
		infra,
		actions,
		// infra.UnsafeDB().FileInfo.Query().Where(fileinfo.FullPath(infra.InboxPath())).OnlyX(context.Background()),
	}
}

func (qq *Inbox) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	fabs := []*wx.FloatingActionButton{
		{
			Icon: "upload_file",
			HTMXAttrs: wx.HTMXAttrs{
				HxPost: qq.actions.Browse.FileUploadDialog.Endpoint(),
				HxVals: util.JSON(qq.actions.Browse.FileUploadDialog.Data(
					ctx.SpaceCtx().SpaceRootDir().PublicID.String(),
					true,
				)),
				LoadInPopover: true,
			},
			Child: []wx.IWidget{
				wx.NewIcon("upload_file"),
				wx.T("Upload file"),
			},
		},
	}

	var viewx renderable.Renderable
	viewx = &wx.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, "inbox", fabs),
		Content:    qq.actions.Page.WidgetHandler(rw, req, ctx, ""),
	}

	renderFullPage := false
	if req.Header.Get("HX-Request") == "" {
		renderFullPage = true
	}

	if renderFullPage {
		viewx = partial2.NewBase(wx.T("Inbox"), viewx)
	}

	return qq.infra.Renderer().Render(rw, ctx, viewx)
}
