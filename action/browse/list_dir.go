package browse

import (
	"fmt"
	"log"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileinfo"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/filesearch"
	"github.com/simpledms/simpledms/db/enttenant/predicate"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/resolvedtagassignment"
	"github.com/simpledms/simpledms/db/enttenant/tagassignment"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/event"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/model/common/fieldtype"
	"github.com/simpledms/simpledms/renderable"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/partial"
	"github.com/simpledms/simpledms/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/sqlutil"
	"github.com/simpledms/simpledms/util/timex"
)

type ListDirData struct {
	CurrentDirID   string
	SelectedFileID string
}

type ListDirState struct {
	ListFilterTagsState
	DocumentTypeFilterState
	PropertiesFilterState

	// TODO move to dedicated struct?
	// SpaceIDs []int64 `url:"space_ids,omitempty"` // shared with DocumentTypeFilterState

	SearchQuery     string `url:"q,omitempty"`
	searchQueryRaw  string
	HideDirectories bool `url:"hide_directories,omitempty"`
	HideFiles       bool `url:"hide_files,omitempty"`
	IsRecursive     bool `url:"recursive,omitempty"`
	// FolderMode      bool   `url:"folder_mode,omitempty"`

	// used in JS, thus don't change URL and as param name below
	// TODO multiple?
	ActiveSideSheet string `url:"side_sheet,omitempty"`

	// TODO does offset belong to state? in url, but not really state...
	// Offset int `url:"offset,omitempty"`

	// Order           string `url:"order,omitempty"`

	// OpenDialog string `url:"dialog,omitempty"`

	// not sure if necessary, probably read from DB (user config) and impl switch view?
	// store per folder? recursively? with a global fallback per user, maybe per user and folder
	// ViewType viewtype.ViewType
}

type ListDir struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewListDir(
	infra *common.Infra,
	actions *Actions,
) *ListDir {
	return &ListDir{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("list-dir"),
			true,
		),
	}
}

func (qq *ListDir) Data(currentDirID, selectedFileID string) *ListDirData {
	return &ListDirData{
		CurrentDirID:   currentDirID,
		SelectedFileID: selectedFileID,
	}
}

func (qq *ListDir) WrapperID() string {
	return "listDirWrapper"
}

func (qq *ListDir) FileListID() string {
	return "fileList"
}

func (qq *ListDir) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ListDirData](rw, req, ctx)
	if err != nil {
		return err
	}

	/*
		// necessary because when filterChip is clicked, the info is send as form
		state, err := autil.FormData[ListDirState](rw, req, ctx)
		if err != nil {
			return err
		}
	*/
	state := autil.StateX[ListDirState](rw, req)

	hxTarget := req.URL.Query().Get("hx-target")
	if hxTarget == "#"+qq.FileListID() {
		// rw.Header().Set("HX-Replace-Url", route.BrowseWithState(state)(data.CurrentDirID))

		// dir := ctx.TenantCtx().TTx.File.GetX(ctx, data.CurrentDirID)
		dir := qq.infra.FileRepo.GetX(ctx, data.CurrentDirID)
		return qq.infra.Renderer().Render(
			rw,
			ctx,
			qq.filesList(
				ctx,
				state,
				dir,
				data,
				0,
			),
		)
	}

	// TODO or HxTrigger? seems to have same value
	if req.Header.Get("Hx-Target") == "filterTagsBtn" {
		return qq.infra.Renderer().Render(
			rw,
			ctx,
			qq.filterTagsBtn(ctx, state, data.CurrentDirID),
		)
	}
	if req.Header.Get("Hx-Target") == "filterPropertiesBtn" {
		return qq.infra.Renderer().Render(
			rw,
			ctx,
			qq.filterPropertiesBtn(ctx, state, data.CurrentDirID),
		)
	}
	if req.Header.Get("Hx-Target") == "filterDocumentTypeBtn" {
		return qq.infra.Renderer().Render(
			rw,
			ctx,
			qq.filterDocumentTypeBtn(ctx, state, data.CurrentDirID),
		)
	}

	if req.Header.Get("Hx-Target") == "listDirLoadMore" {
		// TODO not a good solution, should be more consistent...
		//		both is not good, additional state loading and separate offset
		state = autil.StateX[ListDirState](rw, req)
		offset := 0
		offsetStr := req.URL.Query().Get("offset")
		if offsetStr != "" {
			offset, err = strconv.Atoi(offsetStr)
			if err != nil {
				log.Println(err)
				return err
			}
		}

		dir := qq.infra.FileRepo.GetX(ctx, data.CurrentDirID)
		return qq.infra.Renderer().Render(
			rw,
			ctx,
			&wx.View{
				Children: qq.filesListItems(
					ctx,
					state,
					qq.Data(dir.Data.PublicID.String(), data.SelectedFileID),
					offset,
				),
			},
		)
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(
			ctx,
			state,
			data.CurrentDirID,
			data.SelectedFileID,
		),
	)
}

func (qq *ListDir) WidgetHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	fileID string,
	selectedFileID string,
) *wx.ListDetailLayout {
	state := autil.StateX[ListDirState](rw, req)

	return qq.Widget(
		ctx,
		state,
		fileID,
		selectedFileID,
	)
}

// TODO return error?
// TODO pass in enttenant.File as argument instead of path? how to handle breadcrumbs?
// TODO ListDirData instead of path?
func (qq *ListDir) Widget(
	ctx ctxx.Context,
	state *ListDirState,
	fileID string,
	selectedFileID string,
) *wx.ListDetailLayout {
	// dir := ctx.TenantCtx().TTx.File.GetX(ctx, fileID)
	dirWithParentx := ctx.TenantCtx().TTx.File.Query().WithParent().Where(file.PublicID(entx.NewCIText(fileID))).OnlyX(ctx)
	dirWithParent := qq.infra.FileRepo.GetXX(dirWithParentx)

	if dirWithParent.Data.IsDirectory == false {
		// TODO handle error... return container with error message for user? but should also be 404
		return &wx.ListDetailLayout{}
	}

	var children []wx.IWidget

	children = append(children,
		qq.tagsAndOptions(ctx, state, dirWithParent),
	)

	children = append(children,
		qq.filesList(
			ctx,
			state,
			dirWithParent,
			qq.Data(dirWithParent.Data.PublicID.String(), selectedFileID),
			0,
		),
	)

	if ctx.SpaceCtx().Space.IsFolderMode {
		var breadcrumbs []wx.IWidget
		if dirWithParent.Data.ID > 0 {
			fileInfo := ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.FileID(dirWithParent.Data.ID)).OnlyX(ctx)
			breadcrumbElems := strings.Split(fileInfo.FullPath, string(os.PathSeparator))
			for qi, pathElemID := range fileInfo.PublicPath {
				var breadcrumbLabel wx.IWidget
				if qi == 0 {
					// TODO home Icon
					breadcrumbLabel = &wx.Icon{
						Name: "home",
						Size: wx.IconSizeSmall,
					}
				} else {
					breadcrumbLabel = wx.Tu(breadcrumbElems[qi-1]).SetWrap()
				}
				if qi != len(fileInfo.Path)-1 {
					breadcrumbs = append(breadcrumbs, &wx.Link{
						Href:  route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, pathElemID),
						Child: breadcrumbLabel,
					})
					breadcrumbs = append(breadcrumbs, wx.T("»")) // TODO use icon instead?
				} else {
					// last elem
					breadcrumbs = append(breadcrumbs, breadcrumbLabel)
				}
			}
		}
		if len(breadcrumbs) > 0 {
			children = append(children, &wx.StatusBar{
				Child: breadcrumbs,
			})
		}
	}

	list := &wx.Column{
		GapYSize: wx.Gap2,
		HTMXAttrs: wx.HTMXAttrs{
			HxPost: qq.EndpointWithParams(actionx.ResponseWrapperNone, "#"+qq.FileListID()),
			// HxPost:    qq.EndpointWithState(state, actionx.ResponseWrapperNone, "#"+qq.WrapperID()),
			// state is necessary because tags are not rendered if modal is closed
			HxVals:   util.JSON(qq.Data(dirWithParent.Data.PublicID.String(), selectedFileID)), // overrides form fields, must be added via HxInclude
			HxTarget: "#" + qq.FileListID(),
			HxSwap:   "outerHTML",
			HxTrigger: strings.Join([]string{
				event.FilterTagsChanged.Handler(),
				event.DocumentTypeFilterChanged.Handler(),
				event.PropertyFilterChanged.Handler(),
				event.FolderModeToggled.Handler(),
				event.SearchQueryUpdated.HandlerWithModifier("delay:100ms"),
				event.FileUploaded.Handler(),
				event.ZIPArchiveUnzipped.Handler(),
				event.FileUpdated.Handler(),
				event.FileDeleted.Handler(),
			}, ", "),
		},
		Children: children,
	}

	return &wx.ListDetailLayout{
		Widget: wx.Widget[wx.ListDetailLayout]{
			ID: qq.WrapperID(),
		},
		AppBar: qq.appBar(ctx, state, dirWithParent),
		List:   list,
	}
}

func (qq *ListDir) tagsAndOptions(ctx ctxx.Context, state *ListDirState, dir *model.File) *wx.ChipBar {
	// TODO most used tags within folder, order alphabetically or by use?

	// childDirCount := dir.Data.QueryChildren().Where(file.IsDirectory(true)).CountX(ctx)
	currentDirID := dir.Data.PublicID

	children := qq.filters(ctx, state, currentDirID.String())

	return &wx.ChipBar{
		Widget: wx.Widget[wx.ChipBar]{
			ID: "filters", // TODO was `tags` before
		},
		Children: children,
	}
	// T("Sort by: Name"),
}

func (qq *ListDir) pageSize() int {
	return 50
}

func (qq *ListDir) filesList(
	ctx ctxx.Context,
	state *ListDirState,
	dir *model.File,
	data *ListDirData,
	offset int,
) renderable.Renderable {
	fileListItems := qq.filesListItems(ctx, state, data, offset)

	var content wx.IWidget
	content = &wx.List{
		Children: fileListItems,
	}

	if len(fileListItems) == 0 {
		var widgets []wx.IWidget
		headline := wx.T("No files available yet.")

		if ctx.SpaceCtx().Space.IsFolderMode {
			headline = wx.T("No files or directories available yet.")
			widgets = append(
				widgets,
				qq.actions.MakeDir.ModalLink(
					qq.actions.MakeDir.Data(dir.Data.PublicID.String(), ""),
					[]wx.IWidget{
						&wx.Button{
							Icon:  wx.NewIcon("create_new_folder"),
							Label: wx.T("Create directory"),
						},
					},
					"#"+qq.actions.ListDir.WrapperID(),
				),
			)
		}

		widgets = append(
			widgets,
			&wx.Link{
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:        qq.actions.FileUploadDialog.Endpoint(),
					HxVals:        util.JSON(qq.actions.FileUploadDialog.Data(dir.Data.PublicID.String(), false)),
					LoadInPopover: true,
				},
				Child: &wx.Button{
					Icon:  wx.NewIcon("upload_file"),
					Label: wx.T("Upload file"),
				},
			},
		)

		content = &wx.EmptyState{
			Icon:     wx.NewIcon("description"),
			Headline: headline,
			// Description: NewText("There are no directories or files available yet, you can create"),
			Actions: widgets,
		}
	}

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.FileListID(),
		},
		Children: content,
	}

}

func (qq *ListDir) filesListItems(
	ctx ctxx.Context,
	state *ListDirState,
	data *ListDirData,
	offset int,
) []wx.IWidget {
	// necessary to prevent putting escaped search query into search query field on re-rendering
	state.searchQueryRaw = state.SearchQuery
	// FIXME should be automatically applied on parsing state
	state.SearchQuery = sqlutil.FTSSafeAndQuery(state.SearchQuery, 300)

	// TODO find a better solution; not really robust
	if ctx.SpaceCtx().Space.IsFolderMode && state.DocumentTypeID == 0 && len(state.CheckedTagIDs) == 0 && state.SearchQuery == "" {
		state.IsRecursive = false
		state.HideDirectories = false
	} else {
		state.IsRecursive = true
		// state.HideDirectories = true
	}
	// children := dir.QueryChildren().Order(file.ByName()).WithChildren().AllX(ctx)

	var currentDir *enttenant.File
	if data.CurrentDirID == "" {
		currentDir = ctx.SpaceCtx().SpaceRootDir()
		data.CurrentDirID = currentDir.PublicID.String()
	}
	if currentDir == nil {
		currentDir = ctx.SpaceCtx().Space.QueryFiles().Where(file.PublicID(entx.NewCIText(data.CurrentDirID))).OnlyX(ctx)
	}

	// copied from Search...
	var tagAssignmentPredicates []predicate.TagAssignment
	for _, tagID := range state.ListFilterTagsState.CheckedTagIDs {
		tagAssignmentPredicates = append(tagAssignmentPredicates, tagassignment.TagID(int64(tagID)))
	}

	// TODO sort by relevance
	searchResultQuery := ctx.TenantCtx().TTx.File.Query().
		WithParent().
		WithChildren(). // necessary to count children
		Where(func(qs *sql.Selector) {
			// subquery to select all files in search scope
			if !state.IsRecursive {
				qs.Where(sql.EQ(qs.C(file.FieldParentID), currentDir.ID))
			} else {
				fileInfoView := sql.Table(fileinfo.Table)
				qs.Where(
					sql.In(
						qs.C(file.FieldID),
						sql.Select(fileInfoView.C(fileinfo.FieldFileID)).
							From(fileInfoView).
							Where(sql.And(
								sqljson.ValueContains(fileInfoView.C(fileinfo.FieldPath), currentDir.ID),
								sql.NEQ(fileInfoView.C(fileinfo.FieldFileID), currentDir.ID),
							)),
					),
				)
			}

			if len(state.ListFilterTagsState.CheckedTagIDs) > 0 {
				resolvedTagAssignmentTable := sql.Table(resolvedtagassignment.Table)
				qs.Where(
					sql.Exists(
						sql.Select(resolvedTagAssignmentTable.C(resolvedtagassignment.FieldFileID)).
							From(resolvedTagAssignmentTable).
							Where(
								sql.And(
									// stange behavior if sql.EQ is used instead of sql.ColumnsEQ:
									// executing the query from debugger manually would work, but not via
									// ent because column name (files.id) is passed in as argument for the
									// prepared statement
									sql.ColumnsEQ(resolvedTagAssignmentTable.C(resolvedtagassignment.FieldFileID), qs.C(file.FieldID)),
									sql.InInts(resolvedTagAssignmentTable.C(resolvedtagassignment.FieldTagID), state.ListFilterTagsState.CheckedTagIDs...),
								),
							).
							GroupBy(resolvedTagAssignmentTable.C(resolvedtagassignment.FieldFileID)).
							Having(sql.EQ(sql.Count(resolvedTagAssignmentTable.C(resolvedtagassignment.FieldFileID)), len(state.ListFilterTagsState.CheckedTagIDs))),
					),
				)
			}
		})

	searchResultQuery = searchResultQuery.Where(file.IsInInbox(false))

	// searchResultQuery = searchResultQuery.Where(file.HasSpaceAssignmentWith(spacefileassignment.SpaceID(ctx.SpaceCtx().Space.ID)))
	searchResultQuery = searchResultQuery.Where(file.SpaceID(ctx.SpaceCtx().Space.ID))

	if state.DocumentTypeID != 0 {
		searchResultQuery = searchResultQuery.Where(file.DocumentTypeID(state.DocumentTypeID))
	}

	if state.SearchQuery != "" {
		/*
			searchResultQuery = searchResultQuery.Where(
				file.NameContains(state.SearchQuery),
			) // TODO .Limit(25) // needs hint if enabled
		*/

		// TODO give filename a higher priority?
		searchResultQuery.Where(
			func(qs *sql.Selector) {
				fileSearchTable := sql.Table(filesearch.Table)

				qs.Where(
					sql.In(qs.C(file.FieldID),
						sql.Select(fileSearchTable.C(filesearch.FieldRowid)).From(fileSearchTable).
							Where(
								sql.And(
									sql.EQ(fileSearchTable.C(filesearch.FieldFileSearches), state.SearchQuery),
									sql.LT(fileSearchTable.C(filesearch.FieldRank), 0),
								),
							).
							OrderBy(fileSearchTable.C(filesearch.FieldRank)),
					),
				)
			},
		)
	}

	// TODO use filesearch view instead and order by rank?
	searchResultQuery = searchResultQuery.Order(file.ByIsDirectory(sql.OrderDesc()), file.ByName())

	var fileListItems []wx.IWidget

	if state.HideDirectories && state.HideFiles {
		// do nothing // TODO find a better solution (radio button?)
		searchResultQuery = searchResultQuery.Where(file.And(file.IsDirectory(false), file.IsDirectory(true)))
	} else if state.HideDirectories {
		searchResultQuery = searchResultQuery.Where(file.IsDirectory(false))
	} else if state.HideFiles {
		searchResultQuery = searchResultQuery.Where(file.IsDirectory(true))
	}

	searchResultQuery = qq.applyPropertyFilter(ctx, searchResultQuery, state)

	children := searchResultQuery.Offset(offset).Limit(qq.pageSize() + 1).AllX(ctx)
	hasMore := len(children) > qq.pageSize()
	if hasMore {
		// conditional necessary to prevent out of bounce access
		children = children[:qq.pageSize()]
	}

	// get parent file info full paths for breadcrumbs...
	var childParentFileInfos map[int64]*enttenant.FileInfo
	if state.IsRecursive {
		var childParentIDs []int64
		for _, child := range children {
			childParentIDs = append(childParentIDs, child.ParentID)
		}
		slices.Sort(childParentIDs) // necessary for compact to work?
		childParentIDs = slices.Compact(childParentIDs)
		childParentFileInfosSlice := ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.FileIDIn(childParentIDs...)).AllX(ctx)
		childParentFileInfos = make(map[int64]*enttenant.FileInfo)
		for _, childParentFileInfo := range childParentFileInfosSlice {
			childParentFileInfos[childParentFileInfo.FileID] = childParentFileInfo
		}
	}

	for _, child := range children {
		if !child.IsDirectory {
			continue
		}

		fullPath := ""
		if state.IsRecursive {
			fullPath = childParentFileInfos[child.ParentID].FullPath
		}

		fileListItems = append(fileListItems, qq.actions.FileListItem.DirectoryListItem(
			ctx,
			currentDir.PublicID.String(),
			child,
			fullPath,
			state.IsRecursive,
		))
	}
	for _, child := range children {
		if child.IsDirectory {
			continue
		}

		fullPath := ""
		if state.IsRecursive {
			fullPath = childParentFileInfos[child.ParentID].FullPath
		}

		fileListItems = append(fileListItems, qq.actions.FileListItem.fileListItem(
			ctx,
			data.CurrentDirID,
			child,
			fullPath,
			child.PublicID.String() == data.SelectedFileID,
			// data.SelectedFileID != 0,
			state.IsRecursive && ctx.SpaceCtx().Space.IsFolderMode,
		))
	}

	if hasMore {
		fileListItems = append(fileListItems, &wx.ListItem{
			Widget: wx.Widget[wx.ListItem]{
				ID: "listDirLoadMore",
			},
			Headline: wx.T("Loading more..."),
			HTMXAttrs: wx.HTMXAttrs{
				HxPost:    qq.Endpoint() + "?offset=" + strconv.Itoa(offset+qq.pageSize()), // FIXME
				HxTrigger: "intersect once",
				HxTarget:  "#listDirLoadMore",
				HxSwap:    "outerHTML",
			},
		})
	}

	return fileListItems
}

func (qq *ListDir) appBar(
	ctx ctxx.Context,
	state *ListDirState,
	dir *model.File,
) *wx.AppBar {
	var leadingButton wx.IWidget

	if dir.Data.ParentID != 0 {
		parent, err := dir.Parent(ctx)
		if err != nil {
			log.Println(err)
			panic(err)
		}
		leadingButton = &wx.IconButton{
			Icon: "arrow_back",
			HTMXAttrs: wx.HTMXAttrs{
				HxGet:     route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, parent.Data.PublicID.String()),
				HxHeaders: autil.ResetStateHeader(),
				HxSwap: fmt.Sprintf(
					// duplicate in FileListItem
					// bottom instead of top prevents small jump on nav
					// TODO long query is not ideal because it is error prone, but spaces are not allowed in htmx...
					"innerHTML show:#%s>div>.js-list>.js-list-item:first-child:bottom",
					qq.FileListID(),
				),
			},
		}
	} else if ctx.SpaceCtx().Space.IsFolderMode {
		leadingButton = &wx.Icon{
			Name: "folder_open",
		}
	} else {
		leadingButton = &wx.Icon{
			Name: "folder_open", // TODO folder_open or hub or home?
		}
	}

	supportingText := wx.T("Search")
	supportingTextAltMobile := wx.Tu(dir.Data.Name)
	if ctx.SpaceCtx().Space.IsFolderMode && dir.Data.ID != ctx.SpaceCtx().SpaceRootDir().ID {
		supportingText = wx.Tf("Search in «%s»", dir.Data.Name)
	}

	return &wx.AppBar{
		Leading:          leadingButton,
		LeadingAltMobile: partial.NewMainMenu(ctx),
		Title:            wx.Tu(dir.Data.Name),
		// Actions:          actions,
		Search: &wx.Search{
			Widget: wx.Widget[wx.Search]{
				ID: "search",
			},
			Name:                    "SearchQuery",
			Value:                   state.searchQueryRaw,
			SupportingText:          supportingText,
			SupportingTextAltMobile: supportingTextAltMobile,
			HTMXAttrs: wx.HTMXAttrs{
				HxOn: event.SearchQueryUpdated.HxOnWithQueryParam("input", "q"),
			},
		},
	}
}

func (qq *ListDir) filters(
	ctx ctxx.Context,
	listDirState *ListDirState,
	currentDirID string,
) []wx.IWidget {
	// TODO show only if there are dirs or files in result? would require to check complete query,
	//		not just first 25 results...
	chips := []wx.IWidget{
		qq.filterDocumentTypeBtn(ctx, listDirState, currentDirID),
		qq.filterPropertiesBtn(ctx, listDirState, currentDirID),
		// TODO open on hover
		qq.filterTagsBtn(ctx, listDirState, currentDirID),
	}

	// TODO show only if filter is active
	chips = append(chips,
		&wx.AssistChip{
			// TODO choose another styling, that it is not as prominent as the others
			Label: wx.Tf("Reset"), // just Reset because it also resets search
			HTMXAttrs: wx.HTMXAttrs{
				HxGet:     route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, currentDirID), // TODO or pass in href?
				HxHeaders: autil.ResetStateHeader(),
			},
			LeadingIcon: "restart_alt", // TODO
		},
	)

	return chips
}

func (qq *ListDir) filterTagsBtn(
	ctx ctxx.Context,
	listDirState *ListDirState,
	currentDirID string,
) *wx.Container {
	chipState := wx.AssistChipStateDefault
	if len(listDirState.ListFilterTagsState.CheckedTagIDs) > 0 {
		chipState = wx.AssistChipStateHighlighted
	}

	hxTrigger := ""
	hxPost := qq.actions.TagsFilterDialog.Endpoint()
	var hxOn *wx.HxOn
	if listDirState.ActiveSideSheet == qq.actions.TagsFilterDialog.ID() { // is open
		if !ctx.VisitorCtx().IsHTMXRequest {
			hxTrigger = "load" // leads to strange issues if done on htmx requests
		} else {
			hxPost = ""
		}
		hxOn = event.CloseSideSheet.UnsafeHxOnWithQueryParamAndValue("click", "side_sheet", "")
	} else { // closed
		hxOn = event.SideSheetToggled.UnsafeHxOnWithQueryParamAndValue("click", "side_sheet", qq.actions.TagsFilterDialog.ID())
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: "filterTagsBtn",
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data(currentDirID, "")),
			HxTarget: "#filterTagsBtn",
			HxTrigger: strings.Join([]string{
				event.FilterTagsChanged.Handler(),
				event.SideSheetToggled.Handler(),
			}, ", "),
		},
		Child: &wx.AssistChip{
			IsActive:    listDirState.ActiveSideSheet == qq.actions.TagsFilterDialog.ID(),
			Label:       wx.Tf("Tags"),
			LeadingIcon: "label",
			Badge: &wx.Badge{
				IsInline: true,
				Value:    len(listDirState.ListFilterTagsState.CheckedTagIDs),
			},
			State: chipState,
			HTMXAttrs: wx.HTMXAttrs{
				HxTrigger:     hxTrigger,
				HxPost:        hxPost,
				HxVals:        util.JSON(qq.actions.TagsFilterDialog.Data(currentDirID)),
				LoadInPopover: true,
				HxOn:          hxOn,
			},
		},
	}
}

func (qq *ListDir) filterPropertiesBtn(
	ctx ctxx.Context,
	listDirState *ListDirState,
	currentDirID string,
) *wx.Container {
	// Count the number of active property filters
	activeFilterCount := len(listDirState.PropertyValues)

	chipState := wx.AssistChipStateDefault
	if activeFilterCount > 0 {
		chipState = wx.AssistChipStateHighlighted
	}

	hxTrigger := ""
	hxPost := qq.actions.PropertiesFilterDialog.Endpoint()
	var hxOn *wx.HxOn
	if listDirState.ActiveSideSheet == qq.actions.PropertiesFilterDialog.ID() { // is open
		if !ctx.VisitorCtx().IsHTMXRequest {
			hxTrigger = "load" // leads to strange issues if done on htmx requests
		} else {
			hxPost = ""
		}
		hxOn = event.CloseSideSheet.UnsafeHxOnWithQueryParamAndValue("click", "side_sheet", "")
	} else { // closed
		hxOn = event.SideSheetToggled.UnsafeHxOnWithQueryParamAndValue(
			"click",
			"side_sheet",
			qq.actions.PropertiesFilterDialog.ID(),
		)
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: "filterPropertiesBtn",
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data(currentDirID, "")),
			HxTarget: "#filterPropertiesBtn",
			HxTrigger: strings.Join([]string{
				event.PropertyFilterChanged.Handler(),
				event.SideSheetToggled.Handler(),
			}, ", "),
		},
		Child: &wx.AssistChip{
			IsActive:    listDirState.ActiveSideSheet == qq.actions.PropertiesFilterDialog.ID(),
			Label:       wx.Tf("Fields"),
			LeadingIcon: "tune", // tune or assignment
			Badge: &wx.Badge{
				IsInline: true,
				Value:    activeFilterCount,
			},
			State: chipState,
			HTMXAttrs: wx.HTMXAttrs{
				HxTrigger:     hxTrigger,
				HxPost:        hxPost,
				HxVals:        util.JSON(qq.actions.PropertiesFilterDialog.Data(currentDirID)),
				LoadInPopover: true,
				HxOn:          hxOn,
			},
		},
	}
}

func (qq *ListDir) filterDocumentTypeBtn(
	ctx ctxx.Context,
	listDirState *ListDirState,
	currentDirID string,
) *wx.Container {
	// TODO open on hover

	chipState := wx.AssistChipStateDefault
	if listDirState.DocumentTypeID != 0 {
		chipState = wx.AssistChipStateHighlighted
	}

	hxTrigger := ""
	hxPost := qq.actions.DocumentTypeFilterDialog.Endpoint()
	var hxOn *wx.HxOn
	if listDirState.ActiveSideSheet == qq.actions.DocumentTypeFilterDialog.ID() { // is open
		if !ctx.VisitorCtx().IsHTMXRequest {
			hxTrigger = "load" // leads to strange issues if done on htmx requests
		} else {
			hxPost = ""
		}
		hxOn = event.CloseSideSheet.UnsafeHxOnWithQueryParamAndValue(
			"click",
			"side_sheet",
			"",
		)
	} else { // closed
		hxOn = event.SideSheetToggled.UnsafeHxOnWithQueryParamAndValue(
			"click",
			"side_sheet",
			qq.actions.DocumentTypeFilterDialog.ID(),
		)
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: "filterDocumentTypeBtn",
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data(currentDirID, "")),
			HxTarget: "#filterDocumentTypeBtn",
			HxTrigger: strings.Join([]string{
				// TODO is this necessary (for all buttons) or are handlers on list enough?
				event.FilterTagsChanged.Handler(),
				event.DocumentTypeFilterChanged.Handler(),
				event.PropertyFilterChanged.Handler(),
				event.SideSheetToggled.Handler(),
			}, ", "),
		},
		Child: &wx.AssistChip{
			IsActive:    listDirState.ActiveSideSheet == qq.actions.DocumentTypeFilterDialog.ID(),
			Label:       wx.Tf("Document type"),
			LeadingIcon: "category",
			State:       chipState,
			/*TODO add back
			Badge: &wx.Badge{
				IsInline: true,
				// Value:    len(listDirState.CheckedTagIDs),
			},*/
			HTMXAttrs: wx.HTMXAttrs{
				HxTrigger:     hxTrigger,
				HxPost:        hxPost,
				HxVals:        util.JSON(qq.actions.DocumentTypeFilterDialog.Data(currentDirID)),
				LoadInPopover: true,
				HxOn:          hxOn,
			},
		},
	}
}

func (qq *ListDir) applyPropertyFilter(ctx ctxx.Context, query *enttenant.FileQuery, state *ListDirState) *enttenant.FileQuery {
	var propertyIDs []int64
	for _, propertyFilter := range state.PropertyValues {
		propertyIDs = append(propertyIDs, propertyFilter.PropertyID)
	}
	slices.Sort(propertyIDs)
	propertyIDs = slices.Compact(propertyIDs) // must be sorted

	if len(propertyIDs) == 0 {
		return query
	}

	propertiesx := ctx.SpaceCtx().Space.QueryProperties().Where(property.IDIn(propertyIDs...)).AllX(ctx)

	for _, propertyFilter := range state.PropertyValues {
		propertyx := propertiesx[slices.IndexFunc(propertiesx, func(prop *enttenant.Property) bool {
			return prop.ID == propertyFilter.PropertyID
		})]

		switch propertyx.Type {
		case fieldtype.Text:
			switch propertyFilter.Operator {
			case textOperatorValueContains.String():
				query = query.Where(file.HasPropertyAssignmentWith(
					// TODO space necessary?
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.TextValueContainsFold(propertyFilter.Value), // Fold makes case insensitive
				))
			case operatorValueEquals.String():
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.TextValueEqualFold(propertyFilter.Value), // Fold makes case insensitive
				))
			case textOperatorValueStartsWith.String():
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.TextValueHasPrefix(propertyFilter.Value), // is case insensitive
				))
			}
		case fieldtype.Number:
			value, err := strconv.Atoi(propertyFilter.Value)
			if err != nil {
				log.Println(err)
				continue
			}

			switch propertyFilter.Operator {
			case operatorValueEquals.String():
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.NumberValue(value),
				))
			case operatorValueGreaterThan.String():
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.NumberValueGT(value),
				))
			case operatorValueLessThan.String():
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.NumberValueLT(value),
				))
			}
		case fieldtype.Date:
			value, err := timex.ParseDate(propertyFilter.Value)
			if err != nil {
				log.Println(err)
				continue
			}

			switch propertyFilter.Operator {
			case operatorValueEquals.String():
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.DateValue(value),
				))
			case operatorValueGreaterThan.String():
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.DateValueGT(value),
				))
			case operatorValueLessThan.String():
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.DateValueLT(value),
				))
			}
		case fieldtype.Money:
			valueFloat, err := strconv.ParseFloat(propertyFilter.Value, 64)
			if err != nil {
				log.Println(err)
				continue
			}
			value := int(math.Round(valueFloat * 100)) // convert to minor unit // TODO is this good enough?

			switch propertyFilter.Operator {
			case operatorValueEquals.String():
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.NumberValue(value),
				))
			case operatorValueGreaterThan.String():
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.NumberValueGT(value),
				))
			case operatorValueLessThan.String():
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.NumberValueLT(value),
				))
			}
		case fieldtype.Checkbox:
			value, err := strconv.ParseBool(propertyFilter.Value)
			if err != nil {
				log.Println(err)
				continue
			}
			if value {
				query = query.Where(file.HasPropertyAssignmentWith(
					// TODO space necessary?
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.BoolValue(true),
				))
			} else {
				// file.Or ensures that also listed if no property assignment
				// FIXME doesn't return all results without assignment!
				query = query.Where(
					file.Or(
						file.HasPropertyAssignmentWith(
							filepropertyassignment.PropertyID(propertyFilter.PropertyID),
							filepropertyassignment.Or(filepropertyassignment.BoolValue(false), filepropertyassignment.BoolValueIsNil()),
						),
						file.Not(file.HasPropertyAssignment()),
					),
				)
			}
		}
	}

	return query
}
