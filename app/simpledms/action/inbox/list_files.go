package inbox

import (
	"fmt"
	"log"
	"strings"

	"entgo.io/ent/dialect/sql"

	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/enttenant"
	"github.com/simpledms/simpledms/app/simpledms/enttenant/file"
	"github.com/simpledms/simpledms/app/simpledms/event"
	"github.com/simpledms/simpledms/app/simpledms/renderable"
	"github.com/simpledms/simpledms/app/simpledms/ui/partial"
	"github.com/simpledms/simpledms/app/simpledms/ui/route"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ListFilesData struct {
	SelectedFileID string
}

type ListFilesState struct {
	SearchQuery string `url:"q,omitempty"`
	// used in JS, thus don't change URL and as param name below
	ActiveSideSheet string `url:"side_sheet,omitempty"`
	SortBy          string `url:"sort_by,omitempty"` // TODO enum
}

type ListFiles struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewListFiles(
	infra *common.Infra,
	actions *Actions,
) *ListFiles {
	return &ListFiles{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("list-files"),
			true,
		),
	}
}

func (qq *ListFiles) Data(selectedFileID string) *ListFilesData {
	return &ListFilesData{
		SelectedFileID: selectedFileID,
	}
}

func (qq *ListFiles) WrapperID() string {
	return "listDirWrapper"
}

func (qq *ListFiles) FileListID() string {
	return "fileList"
}
func (qq *ListFiles) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[ListFilesData](rw, req, ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	// necessary because when filterChip is clicked, the info is send as form
	state, err := autil.FormData[PageState](rw, req, ctx)
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

func (qq *ListFiles) WidgetHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	selectedFileID string,
) *wx.ListDetailLayout {
	state := autil.StateX[PageState](rw, req)

	return qq.Widget(
		ctx,
		state,
		selectedFileID,
	)
}

// TODO return error?
// TODO pass in enttenant.File as argument instead of path? how to handle breadcrumbs?
// TODO ListDirData instead of path?
func (qq *ListFiles) Widget(
	ctx ctxx.Context,
	state *PageState,
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
				fmt.Sprintf("input from:#search delay:100ms"),
				event.SearchQueryUpdated.HandlerWithModifier("delay:100ms"),
				event.FileUploaded.Handler(),
				event.ZIPArchiveUnzipped.Handler(), // TODO necessary?
				event.FileDeleted.Handler(),
				event.FileUpdated.Handler(),
				event.SortByUpdated.HandlerWithModifier("delay:100ms"), // TODO delay necessary?
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

func (qq *ListFiles) filesList(
	ctx ctxx.Context,
	state *PageState,
	data *ListFilesData,
) renderable.Renderable {
	var fileListItems []wx.IWidget

	searchResultQuery := qq.filesQuery(ctx, state)
	// TODO .Limit(25) // needs hint if enabled
	children := searchResultQuery.AllX(ctx)

	for _, child := range children {
		if child.IsDirectory {
			continue
		}

		fileListItems = append(fileListItems, qq.actions.FileListItem.Widget(
			ctx,
			// route.InboxWithState(state),
			route.Inbox,
			child,
			child.PublicID.String() == data.SelectedFileID,
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
						HxPost: qq.actions.Browse.FileUploadDialog.Endpoint(),
						HxVals: util.JSON(qq.actions.Browse.FileUploadDialog.Data(
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
	}
}

// LIMIT must be applied by caller
func (qq *ListFiles) filesQuery(ctx ctxx.Context, state *PageState) *enttenant.FileQuery {
	searchResultQuery := ctx.TenantCtx().TTx.File.Query().
		WithParent().
		WithChildren() // necessary to count children
	/*Where(func(qs *sql.Selector) {
		// subquery to select all files in search scope
		fileInfoView := sql.Table(fileinfo.Table)
		qs.Where(
			sql.In(
				qs.C(file.FieldID),
				sql.Select(fileInfoView.C(fileinfo.FieldFileID)).
					From(fileInfoView).
					Where(sql.And(
						sqljson.ValueContains(fileInfoView.C(fileinfo.FieldPath), qq.inboxDir.ID),
						sql.NEQ(fileInfoView.C(fileinfo.FieldFileID), qq.inboxDir.ID),
					)),
			),
		)
	})*/

	searchResultQuery = searchResultQuery.Where(
		file.SpaceID(ctx.SpaceCtx().Space.ID),
		file.IsInInbox(true),
		/*file.HasSpaceAssignmentWith(
			spacefileassignment.SpaceID(ctx.SpaceCtx().Space.ID),
			spacefileassignment.IsInInbox(true),
		),*/
	)

	if state.SearchQuery != "" {
		// TODO necessary if not full text search? probably not
		// searchQuerySanitized := sqlutil.FTSSafeAndQuery(state.SearchQuery, 300)

		searchResultQuery = searchResultQuery.Where(
			file.NameContains(state.SearchQuery),
		)
	}

	// TODO use filesearch view instead and order by rank?
	switch state.SortBy {
	case "name":
		searchResultQuery = searchResultQuery.Order(file.ByName())
	case "oldestFirst":
		searchResultQuery = searchResultQuery.Order(file.ByCreatedAt())
	case "newestFirst":
		fallthrough
	default:
		searchResultQuery = searchResultQuery.Order(file.ByCreatedAt(sql.OrderDesc()))
	}
	// searchResultQuery = searchResultQuery.Order(file.ByName())

	return searchResultQuery
}

func (qq *ListFiles) appBar(ctx ctxx.Context, state *PageState) *wx.AppBar {
	return &wx.AppBar{
		Leading:          wx.NewIcon("inbox"),
		LeadingAltMobile: partial.NewMainMenu(ctx),
		Title:            wx.T("Inbox"),
		Actions: []wx.IWidget{
			&wx.IconButton{
				Icon:     "sort",
				Children: NewSortListContextMenu(qq.actions).Widget(ctx, &state.ListFilesState),
			},
		},
		Search: &wx.Search{
			Widget: wx.Widget[wx.Search]{
				ID: "search",
			},
			Name:           "SearchQuery",
			Value:          state.SearchQuery,
			SupportingText: wx.Tf("Search in «Inbox»"),
		},
	}
}
