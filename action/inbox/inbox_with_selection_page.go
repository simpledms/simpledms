package inbox

import (
	"net/http"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type InboxWithSelectionPage struct {
	infra   *common.Infra
	actions *Actions

	// inboxDirInfo *ent.FileInfo
}

func NewInboxWithSelectionPage(infra *common.Infra, actions *Actions) *InboxWithSelectionPage {
	return &InboxWithSelectionPage{
		infra:   infra,
		actions: actions,
		// inboxDirInfo: infra.UnsafeDB().FileInfo.Query().Where(fileinfo.FullPathEqualFold(infra.InboxPath())).OnlyX(context.Background()),
	}
}

func (qq *InboxWithSelectionPage) Handler(
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

	content := qq.actions.InboxPage.WidgetHandler(rw, req, ctx, filex.PublicID.String())
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
				HxPost: qq.actions.Browse.FileUploadDialogPartial.Endpoint(),
				HxVals: util.JSON(qq.actions.Browse.FileUploadDialogPartial.Data(
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
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "inbox", fabs),
		Content:    content, // TODO pass in filex?
	}

	renderFullPage := false
	if req.Header.Get("HX-Request") == "" {
		renderFullPage = true
	}

	if renderFullPage {
		viewx = partial2.NewBase(wx.T("Inbox"), viewx)
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		viewx,
	)
}
