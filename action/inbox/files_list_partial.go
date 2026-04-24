package inbox

import (
	"log"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FilesListPartialData struct {
	SelectedFileID string
}

type FilesListPartialState struct {
	SearchQuery string `url:"q,omitempty"`
	// used in JS, thus don't change URL and as param name below
	ActiveSideSheet string `url:"side_sheet,omitempty"`
	SortBy          string `url:"sort_by,omitempty"` // TODO enum
}

type FilesListPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewListFilesPartial(
	infra *common.Infra,
	actions *Actions,
) *FilesListPartial {
	return &FilesListPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("files-list-partial"),
			true,
		),
	}
}

func (qq *FilesListPartial) Data(selectedFileID string) *FilesListPartialData {
	return &FilesListPartialData{
		SelectedFileID: selectedFileID,
	}
}

func (qq *FilesListPartial) WrapperID() string {
	return "listDirWrapper"
}

func (qq *FilesListPartial) FileListID() string {
	return "fileList"
}
func (qq *FilesListPartial) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[FilesListPartialData](rw, req, ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	// necessary because when filterChip is clicked, the info is send as form
	state, err := autil.FormData[InboxPageState](rw, req, ctx)
	if err != nil {
		return err
	}

	hxTarget := req.URL.Query().Get("hx-target")
	if hxTarget == "#"+qq.FileListID() {
		rw.Header().Set("HX-Replace-Url", route.InboxRootWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID))

		return qq.infra.Renderer().Render(
			rw,
			ctx,
			qq.filesList(
				ctx,
				state,
				data,
			),
		)
	}

	return qq.infra.Renderer().Render(rw, ctx, qq.Widget(ctx, state, data.SelectedFileID))
}

func (qq *FilesListPartial) WidgetHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	selectedFileID string,
) *wx.ListDetailLayout {
	state := autil.StateX[InboxPageState](rw, req)

	return qq.Widget(
		ctx,
		state,
		selectedFileID,
	)
}

// TODO return error?
// TODO pass in enttenant.File as argument instead of path? how to handle breadcrumbs?
// TODO ListDirData instead of path?
func (qq *FilesListPartial) Widget(
	ctx ctxx.Context,
	state *InboxPageState,
	selectedFileID string,
) *wx.ListDetailLayout {
	var children []wx.IWidget
	var appBar *wx.AppBar

	if selectedFileID == "" {
		appBar = qq.appBar(ctx, state)
	} else {
		appBar = &wx.AppBar{
			Title:   wx.T("Inbox"),
			Leading: wx.NewIcon("inbox"),
		}
	}

	children = append(children,
		qq.filesList(
			ctx,
			state,
			qq.Data(selectedFileID),
		),
	)

	list := &wx.Column{
		Widget: wx.Widget[wx.Column]{
			ID: qq.WrapperID(),
		},
		GapYSize: wx.Gap2,
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.EndpointWithParams(actionx.ResponseWrapperNone, ""),
			HxVals:   util.JSON(qq.Data(selectedFileID)), // overrides form fields, must be added via HxInclude
			HxTarget: "#innerContent",                    // not just fileList because of sortBy selection
			HxSwap:   "innerHTML",
			HxTrigger: strings.Join([]string{
				// see comment on HTMXAttrs on ScrollableContent (FileList)
				event.SortByUpdated.HandlerWithModifier("delay:100ms"), // TODO delay necessary?
				event.FileMoved.Handler(),                              // because it also has to close details
			}, ", "),
			HxInclude: "#search,#sortBy",
		},
		Children: children,
	}
	return &wx.ListDetailLayout{
		AppBar: appBar,
		List:   list,
	}
}

func (qq *FilesListPartial) filesList(
	ctx ctxx.Context,
	state *InboxPageState,
	data *FilesListPartialData,
) renderable.Renderable {
	var fileListItems []wx.IWidget

	repos := qq.infra.SpaceFileRepoFactory().ForSpaceX(ctx)
	children := repos.Query.InboxFilesX(ctx, &filemodel.InboxFileQueryFilterDTO{
		SearchQuery: state.SearchQuery,
		SortBy:      state.SortBy,
	})

	for _, child := range children {
		if child.IsDirectory {
			continue
		}

		fileListItems = append(fileListItems, qq.actions.FileListItemPartial.Widget(
			ctx,
			// route.InboxWithState(state),
			route.Inbox,
			child,
			child.PublicID == data.SelectedFileID,
		))
	}

	var content wx.IWidget
	content = &wx.List{
		Children: fileListItems,
	}

	if len(fileListItems) == 0 {
		content = &wx.EmptyState{
			Icon:     wx.NewIcon("description"),
			Headline: wx.T("No files available yet."),
			// Description: NewText("There are no directories or files available yet, you can create"),
			Actions: []wx.IWidget{
				&wx.Link{
					HTMXAttrs: wx.HTMXAttrs{
						HxPost: qq.actions.Browse.FileUploadDialogPartial.Endpoint(),
						HxVals: util.JSON(qq.actions.Browse.FileUploadDialogPartial.Data(
							ctx.SpaceCtx().SpaceRootDir().PublicID.String(),
							true,
						)),
						LoadInPopover: true,
					},
					Child: &wx.Button{
						Icon:  wx.NewIcon("upload_file"),
						Label: wx.T("Upload file"),
					},
				},
			},
		}
	}

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.FileListID(),
		},
		Children: content,
		// must be on ScrollableContent and not directly on wx.List because otherwise page breaks
		// if a search has no results and empty state is rendered without HTMXAttrs
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.EndpointWithParams(actionx.ResponseWrapperNone, "#"+qq.FileListID()),
			HxVals:   util.JSON(data), // overrides form fields, must be added via HxInclude
			HxTarget: "#" + qq.FileListID(),
			HxSwap:   "outerHTML",
			HxTrigger: strings.Join([]string{
				// SortByUpdated is handled separately because it has to update sortby
				// context menu and the files list; sortby context menu is part of appbar and thus
				// updating the app bar while using the search input leads to flickering and
				// loss of input while typing
				event.SearchQueryUpdated.HandlerWithModifier("delay:100ms"),
				event.FileUploaded.Handler(),
				event.ZIPArchiveUnzipped.Handler(), // TODO necessary?
				event.FileDeleted.Handler(),
				event.FileUpdated.Handler(),
			}, ", "),
			HxInclude: "#search,#sortBy",
		},
	}
}
func (qq *FilesListPartial) appBar(ctx ctxx.Context, state *InboxPageState) *wx.AppBar {
	return &wx.AppBar{
		Leading:          wx.NewIcon("inbox"),
		LeadingAltMobile: partial.NewMainMenu(ctx, qq.infra),
		Title:            wx.T("Inbox"),
		Actions: []wx.IWidget{
			&wx.IconButton{
				Icon:     "sort",
				Tooltip:  wx.T("Sort files"),
				Children: NewSortListContextMenuWidget(qq.actions).Widget(ctx, &state.FilesListPartialState),
			},
		},
		Search: &wx.Search{
			Widget: wx.Widget[wx.Search]{
				ID: "search",
			},
			Name:           "SearchQuery",
			Value:          state.SearchQuery,
			SupportingText: wx.Tf("Search in «Inbox»"),
			HTMXAttrs: wx.HTMXAttrs{
				HxOn: event.SearchQueryUpdated.HxOnWithQueryParam("input", "q"),
			},
		},
	}
}
