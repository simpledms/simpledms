package page

import (
	"net/http"

	"github.com/simpledms/simpledms/action/inbox"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/renderable"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/partial"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type InboxWithSelection struct {
	infra   *common.Infra
	actions *inbox.Actions

	// inboxDirInfo *ent.FileInfo
}

func NewInboxWithSelection(infra *common.Infra, actions *inbox.Actions) *InboxWithSelection {
	return &InboxWithSelection{
		infra:   infra,
		actions: actions,
		// inboxDirInfo: infra.UnsafeDB().FileInfo.Query().Where(fileinfo.FullPathEqualFold(infra.InboxPath())).OnlyX(context.Background()),
	}
}

func (qq *InboxWithSelection) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	// TODO handle direct access

	fileIDStr := req.PathValue("file_id")
	if fileIDStr == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "No file id provided.")
	}
	filex := ctx.SpaceCtx().Space.QueryFiles().Where(file.PublicID(entx.NewCIText(fileIDStr))).OnlyX(ctx)

	// assignment := ctx.SpaceCtx().Space.QueryFileAssignment().Where(spacefileassignment.FileID(fileID64)).OnlyX(ctx)

	if !filex.IsInInbox {
		return e.NewHTTPErrorf(http.StatusBadRequest, "File is not in inbox.")
	}

	/* disabled on 26.06.2025 because done in WidgetHandler
	// necessary to preserve activeTab in ShowFile when switching file from list
	// duplicate in Browse.Handler
	if !filex.IsDirectory && req.Header.Get("HX-Request") != "" {
		activeTab := ""
		currentURL := req.Header.Get("HX-Current-URL")
		if currentURL != "" {
			currentURLx, err := url.Parse(currentURL)
			if err != nil {
				log.Println(err)
			} else {
				activeTab = currentURLx.Query().Get("tab")
				reqQuery := req.URL.Query()
				if activeTab != "" {
					reqQuery.Set("tab", activeTab)
					req.URL.RawQuery = reqQuery.Encode()
					rw.Header().Set("HX-Push-Url", req.URL.String())
				}
			}
		}
	}
	*/

	content := qq.actions.Page.WidgetHandler(rw, req, ctx, filex.PublicID.String())
	if req.Header.Get("HX-Target") == "details" {
		return qq.infra.Renderer().Render(
			rw,
			ctx,
			// TODO shouldn't process all if just Detail is necessary...
			content.Detail,
		)
	}

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
		Navigation: partial.NewNavigationRail(ctx, "inbox", fabs),
		Content:    content, // TODO pass in filex?
	}

	renderFullPage := false
	if req.Header.Get("HX-Request") == "" {
		renderFullPage = true
	}

	if renderFullPage {
		viewx = partial.NewBase(wx.T("Inbox"), viewx)
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		viewx,
	)
}
