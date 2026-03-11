package browse

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type BrowsePage struct {
	infra   *common.Infra
	actions *Actions
}

func NewBrowsePage(infra *common.Infra, actions *Actions) *BrowsePage {
	return &BrowsePage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *BrowsePage) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	dirIDStr := req.PathValue("dir_id")
	var dirx *enttenant.File

	if dirIDStr == "" {
		// set root
		dirx = ctx.SpaceCtx().SpaceRootDir()
	} else {
		dirx = ctx.SpaceCtx().Space.QueryFiles().Where(file.PublicID(entx.NewCIText(dirIDStr))).OnlyX(ctx)
	}

	if !dirx.IsDirectory {
		return e.NewHTTPErrorf(http.StatusBadRequest, "file is not a directory")
	}

	state := autil.StateX[ListDirPartialState](rw, req)

	// commented on 28.01.2026; if reactivated, should be Replace
	// rw.Header().Set("HX-Push-Url", route.BrowseWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, dirx.PublicID.String()))

	// TODO is this a good idea, or would just targeting #details be better instead of custom header?
	//		custom header is more meaningful...
	if req.Header.Get("Close-Details") != "" {
		rw.Header().Set("HX-Retarget", "#details")
		rw.Header().Set("HX-Reswap", "innerHTML")
		return qq.infra.Renderer().Render(rw, ctx, &wx.View{})
	}

	browsePage, err := qq.widget(req, ctx, state, dirx)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "could not render widget")
	}

	qq.render(rw, req, ctx, browsePage)
	return nil
}

func (qq *BrowsePage) render(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	viewx renderable.Renderable,
) {
	renderFullPage := false
	if req.Header.Get("HX-Request") == "" {
		renderFullPage = true
	}

	if renderFullPage {
		viewx = partial2.NewBase(wx.T("Files"), viewx)
	}

	qq.infra.Renderer().RenderX(rw, ctx, viewx)
}

func (qq *BrowsePage) widget(
	req *httpx.Request,
	ctx ctxx.Context,
	state *ListDirPartialState,
	dir *enttenant.File,
) (renderable.Renderable, error) {
	listDetailLayout := qq.actions.ListDirPartial.Widget(
		ctx,
		state,
		dir.PublicID.String(),
		"",
	)

	var fabs []*wx.FloatingActionButton

	fabs = append(fabs, &wx.FloatingActionButton{
		Icon: "upload_file",
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:        qq.actions.FileUploadDialogPartial.Endpoint(),
			HxVals:        util.JSON(qq.actions.FileUploadDialogPartial.Data(dir.PublicID.String(), false)),
			LoadInPopover: true,
		},
		/*
			HTMXAttrs: qq.actions.UploadFileCmd.ModalLinkAttrs(
				qq.actions.UploadFileCmd.Data(dir.PublicID.String(), "", false),
				"#"+qq.actions.ListDirPartial.WrapperID(),
			),
		*/
		Child: []wx.IWidget{
			wx.NewIcon("upload_file"),
			wx.T("Upload file"),
		},
	})

	if ctx.SpaceCtx().Space.IsFolderMode {
		fabs = append(fabs, &wx.FloatingActionButton{
			FABSize: wx.FABSizeSmall,
			Icon:    "create_new_folder",
			HTMXAttrs: qq.actions.MakeDirCmd.ModalLinkAttrs(
				qq.actions.MakeDirCmd.Data(dir.PublicID.String(), ""),
				"#"+qq.actions.ListDirPartial.WrapperID(),
			),
			Child: []wx.IWidget{
				wx.NewIcon("create_new_folder"),
				wx.T("Create directory"),
			},
		})
	}

	mainLayout := &wx.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "browse", fabs),
		Content:    listDetailLayout,
	}
	return mainLayout, nil
}
