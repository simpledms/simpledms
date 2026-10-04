package browse

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/ui/util"
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
		return e.NewHTTPErrorf(http.StatusBadRequest, "File is not a folder.")
	}

	state := autil.StateX[ListDirPartialState](rw, req)

	// commented on 28.01.2026; if reactivated, should be Replace
	// rw.Header().Set("HX-Push-Url", route.BrowseWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, dirx.PublicID.String()))

	// TODO is this a good idea, or would just targeting #details be better instead of custom header?
	//		custom header is more meaningful...
	if req.Header.Get("Close-Details") != "" {
		rw.Header().Set("HX-Retarget", "#details")
		rw.Header().Set("HX-Reswap", "morph:outerHTML")
		return qq.infra.Renderer().Render(rw, ctx, &widget.DetailsWithSheet{})
	}

	browsePage, err := qq.widget(req, ctx, state, dirx)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not render widget.")
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
	renderFullPage := req.Header.Get("HX-Request") == ""

	if renderFullPage {
		viewx = partial2.NewBase(widget.T("Files"), viewx)
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

	var fabs []*widget.FloatingActionButton

	fabs = append(fabs, &widget.FloatingActionButton{
		Icon: "upload_file",
		HTMXAttrs: widget.HTMXAttrs{
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
		Child: []widget.IWidget{
			widget.NewIcon("upload_file"),
			widget.T("Upload file"),
		},
	})

	if ctx.SpaceCtx().Space.IsFolderMode {
		fabs = append(fabs, &widget.FloatingActionButton{
			FABSize: widget.FABSizeSmall,
			FABType: widget.FABTypeSecondary,
			Icon:    "create_new_folder",
			HTMXAttrs: qq.actions.MakeDirCmd.ModalLinkAttrs(
				qq.actions.MakeDirCmd.Data(dir.PublicID.String(), ""),
				"#"+qq.actions.ListDirPartial.WrapperID(),
			),
			Child: []widget.IWidget{
				widget.NewIcon("create_new_folder"),
				widget.T("Create folder"),
			},
		})
	}

	var content widget.IWidget = listDetailLayout
	if state.ActiveSideSheet == "" {
		content = &widget.View{
			Children: []widget.IWidget{
				listDetailLayout,
				qq.defaultSideSheetTrigger(dir.PublicID.String()),
			},
		}
	}

	mainLayout := &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "browse", fabs),
		Content:    content,
	}
	return mainLayout, nil
}

// defaultSideSheetTrigger opens the document type filter by default where side sheets fit
// beside the content (lg, 1200px, see Dialog). It is rendered only by the page, not by
// ListDirPartial, so list and chip refreshes don't reopen a sheet the user closed. The stable
// ID lets morph retain it on folder navigation, so load doesn't fire again there either.
func (qq *BrowsePage) defaultSideSheetTrigger(currentDirID string) *widget.Container {
	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: "browseDefaultSideSheetTrigger",
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxTrigger:     "load[window.matchMedia('(min-width: 1200px)').matches]",
			HxPost:        qq.actions.DocumentTypeFilterDialogPartial.Endpoint(),
			HxVals:        util.JSON(qq.actions.DocumentTypeFilterDialogPartial.Data(currentDirID)),
			LoadInPopover: true,
		},
	}
}
