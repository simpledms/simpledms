package browse

import (
	"fmt"
	"log"
	"strconv"
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

type ListDirPartialData struct {
	CurrentDirID   string
	SelectedFileID string
}

type ListDirPartialState struct {
	ListFilterTagsPartialState
	DocumentTypeFilterPartialState
	PropertiesFilterState

	// TODO move to dedicated struct?
	// SpaceIDs []int64 `url:"space_ids,omitempty"` // shared with DocumentTypeFilterPartialState

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

type ListDirPartial struct {
	infra            *common.Infra
	actions          *Actions
	fileQueryService *ListDirFileQueryService
	*actionx.Config
}

func NewListDirPartial(
	infra *common.Infra,
	actions *Actions,
) *ListDirPartial {
	return &ListDirPartial{
		infra:            infra,
		actions:          actions,
		fileQueryService: NewListDirFileQueryService(infra),
		Config: actionx.NewConfig(
			actions.Route("list-dir-partial"),
			true,
		),
	}
}

func (qq *ListDirPartial) Data(currentDirID, selectedFileID string) *ListDirPartialData {
	return &ListDirPartialData{
		CurrentDirID:   currentDirID,
		SelectedFileID: selectedFileID,
	}
}

func (qq *ListDirPartial) WrapperID() string {
	return "listDirWrapper"
}

func (qq *ListDirPartial) FileListID() string {
	return "fileList"
}

func (qq *ListDirPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ListDirPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	/*
		// necessary because when filterChip is clicked, the info is send as form
		state, err := autil.FormData[ListDirPartialState](rw, req, ctx)
		if err != nil {
			return err
		}
	*/
	state := autil.StateX[ListDirPartialState](rw, req)

	hxTarget := req.URL.Query().Get("hx-target")
	if hxTarget == "#"+qq.FileListID() {
		// rw.Header().Set("HX-Replace-Url", route.BrowseWithState(state)(data.CurrentDirID))

		repos := qq.infra.SpaceFileRepoFactory().ForSpaceX(ctx)
		dir := repos.Read.FileByPublicIDX(ctx, data.CurrentDirID)
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
		state = autil.StateX[ListDirPartialState](rw, req)
		offset := 0
		offsetStr := req.URL.Query().Get("offset")
		if offsetStr != "" {
			offset, err = strconv.Atoi(offsetStr)
			if err != nil {
				log.Println(err)
				return err
			}
		}

		repos := qq.infra.SpaceFileRepoFactory().ForSpaceX(ctx)
		dir := repos.Read.FileByPublicIDX(ctx, data.CurrentDirID)
		return qq.infra.Renderer().Render(
			rw,
			ctx,
			&wx.View{
				Children: qq.filesListItems(
					ctx,
					state,
					qq.Data(dir.PublicID, data.SelectedFileID),
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

func (qq *ListDirPartial) WidgetHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	fileID string,
	selectedFileID string,
) *wx.ListDetailLayout {
	state := autil.StateX[ListDirPartialState](rw, req)

	return qq.Widget(
		ctx,
		state,
		fileID,
		selectedFileID,
	)
}

// TODO return error?
// TODO pass in enttenant.File as argument instead of path? how to handle breadcrumbs?
// TODO ListDirPartialData instead of path?
func (qq *ListDirPartial) Widget(
	ctx ctxx.Context,
	state *ListDirPartialState,
	fileID string,
	selectedFileID string,
) *wx.ListDetailLayout {
	repos := qq.infra.SpaceFileRepoFactory().ForSpaceX(ctx)
	dirWithParent := repos.Read.FileByPublicIDWithParentX(ctx, fileID)

	if !dirWithParent.IsDirectory {
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
			&dirWithParent.FileDTO,
			qq.Data(dirWithParent.PublicID, selectedFileID),
			0,
		),
	)

	if ctx.SpaceCtx().Space.IsFolderMode {
		var breadcrumbs []wx.IWidget
		if dirWithParent.ID > 0 {
			pathFiles := qq.infra.FileSystem().FileTree().PathFilesByFileIDX(ctx, dirWithParent.ID)
			for qi, pathFile := range pathFiles {
				var breadcrumbLabel wx.IWidget
				if qi == 0 {
					// TODO home Icon
					breadcrumbLabel = &wx.Icon{
						Name: "home",
						Size: wx.IconSizeSmall,
					}
				} else {
					breadcrumbLabel = wx.Tu(pathFile.Name).SetWrap()
				}
				if qi != len(pathFiles)-1 {
					breadcrumbs = append(breadcrumbs, &wx.Link{
						Href:  route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, pathFile.PublicID.String()),
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
			HxVals:   util.JSON(qq.Data(dirWithParent.PublicID, selectedFileID)), // overrides form fields, must be added via HxInclude
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

func (qq *ListDirPartial) tagsAndOptions(
	ctx ctxx.Context,
	state *ListDirPartialState,
	dir *filemodel.FileWithParentDTO,
) *wx.ChipBar {
	// TODO most used tags within folder, order alphabetically or by use?

	// childDirCount := len(children)
	currentDirID := dir.PublicID

	children := qq.filters(ctx, state, currentDirID)

	return &wx.ChipBar{
		Widget: wx.Widget[wx.ChipBar]{
			ID: "filters", // TODO was `tags` before
		},
		Children: children,
	}
	// T("Sort by: Name"),
}

func (qq *ListDirPartial) pageSize() int {
	return 50
}

func (qq *ListDirPartial) filesList(
	ctx ctxx.Context,
	state *ListDirPartialState,
	dir *filemodel.FileDTO,
	data *ListDirPartialData,
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
				qq.actions.MakeDirCmd.ModalLink(
					qq.actions.MakeDirCmd.Data(dir.PublicID, ""),
					[]wx.IWidget{
						&wx.Button{
							Icon:  wx.NewIcon("create_new_folder"),
							Label: wx.T("Create directory"),
						},
					},
					"#"+qq.actions.ListDirPartial.WrapperID(),
				),
			)
		}

		widgets = append(
			widgets,
			&wx.Link{
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:        qq.actions.FileUploadDialogPartial.Endpoint(),
					HxVals:        util.JSON(qq.actions.FileUploadDialogPartial.Data(dir.PublicID, false)),
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

func (qq *ListDirPartial) filesListItems(
	ctx ctxx.Context,
	state *ListDirPartialState,
	data *ListDirPartialData,
	offset int,
) []wx.IWidget {
	var fileListItems []wx.IWidget

	queryResult := qq.fileQueryService.Query(
		ctx,
		state,
		data,
		offset,
		qq.pageSize(),
	)
	currentDir := queryResult.CurrentDir
	children := queryResult.Children
	hasMore := queryResult.HasMore
	childParentFullPaths := queryResult.ChildParentFullPaths

	for _, child := range children {
		if !child.IsDirectory {
			continue
		}

		fullPath := ""
		if state.IsRecursive {
			fullPath = childParentFullPaths[child.ParentID]
		}

		fileListItems = append(fileListItems, qq.actions.FileListItemPartial.DirectoryListItem(
			ctx,
			currentDir.PublicID,
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
			fullPath = childParentFullPaths[child.ParentID]
		}

		fileListItems = append(fileListItems, qq.actions.FileListItemPartial.fileListItem(
			ctx,
			data.CurrentDirID,
			child,
			fullPath,
			child.PublicID == data.SelectedFileID,
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

func (qq *ListDirPartial) appBar(
	ctx ctxx.Context,
	state *ListDirPartialState,
	dir *filemodel.FileWithParentDTO,
) *wx.AppBar {
	var leadingButton wx.IWidget

	if dir.ParentID != 0 {
		parentPublicID := ""
		if dir.Parent != nil {
			parentPublicID = dir.Parent.PublicID
		} else {
			repos := qq.infra.SpaceFileRepoFactory().ForSpaceX(ctx)
			parent := repos.Read.FileByIDX(ctx, dir.ParentID)
			parentPublicID = parent.PublicID
		}
		leadingButton = &wx.IconButton{
			Icon:    "arrow_back",
			Tooltip: wx.T("Back to parent folder"),
			HTMXAttrs: wx.HTMXAttrs{
				HxGet:     route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, parentPublicID),
				HxHeaders: autil.ResetStateHeader(),
				HxSwap: fmt.Sprintf(
					// duplicate in FileListItemPartial
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
	supportingTextAltMobile := wx.Tu(dir.Name)
	if ctx.SpaceCtx().Space.IsFolderMode && dir.ID != ctx.SpaceCtx().SpaceRootDir().ID {
		supportingText = wx.Tf("Search in «%s»", dir.Name)
	}

	return &wx.AppBar{
		Leading:          leadingButton,
		LeadingAltMobile: partial.NewMainMenu(ctx, qq.infra),
		Title:            wx.Tu(dir.Name),
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

func (qq *ListDirPartial) filters(
	ctx ctxx.Context,
	listDirState *ListDirPartialState,
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
				HxOn:      event.CloseSideSheet.HxOn("click"),
			},
			LeadingIcon: "restart_alt", // TODO
		},
	)

	return chips
}

func (qq *ListDirPartial) filterTagsBtn(
	ctx ctxx.Context,
	listDirState *ListDirPartialState,
	currentDirID string,
) *wx.Container {
	chipState := wx.AssistChipStateDefault
	if len(listDirState.ListFilterTagsPartialState.CheckedTagIDs) > 0 {
		chipState = wx.AssistChipStateHighlighted
	}

	hxTrigger := ""
	hxPost := qq.actions.TagsFilterDialogPartial.Endpoint()
	var hxOn *wx.HxOn
	if listDirState.ActiveSideSheet == qq.actions.TagsFilterDialogPartial.ID() { // is open
		if !ctx.VisitorCtx().IsHTMXRequest {
			hxTrigger = "load" // leads to strange issues if done on htmx requests
		} else {
			hxPost = ""
		}
		/*hxOn = &wx.HxOn{
			Event:   "click",
			Handler: "document.querySelectorAll('.js-side-sheet-dialog').forEach(elem => elem.closeSideSheet())",
		}*/
		// hxOn = event.CloseSideSheet.UnsafeHxOnWithQueryParamAndValue("click", "side_sheet", "")
		hxOn = event.CloseSideSheet.HxOn("click")
	} else { // closed
		/*hxOn = &wx.HxOn{
			Event:   "click",
			Handler: "document.querySelectorAll('.js-side-sheet-dialog').forEach(elem => elem.toggleCustom())",
		}*/
		// hxOn = event.SideSheetToggled.UnsafeHxOnWithQueryParamAndValue("click", "side_sheet", qq.actions.TagsFilterDialogPartial.ID())
		hxOn = event.SideSheetToggled.HxOn("click")
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: "filterTagsBtn",
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data(currentDirID, "")),
			HxTarget: "#filterTagsBtn",
			HxSwap:   "outerHTML",
			HxTrigger: strings.Join([]string{
				event.FilterTagsChanged.Handler(),
				event.SideSheetToggled.Handler(),
			}, ", "),
		},
		Child: &wx.AssistChip{
			IsActive:    listDirState.ActiveSideSheet == qq.actions.TagsFilterDialogPartial.ID(),
			Label:       wx.Tf("Tags"),
			LeadingIcon: "label",
			Badge: &wx.Badge{
				IsInline: true,
				Value:    len(listDirState.ListFilterTagsPartialState.CheckedTagIDs),
			},
			State: chipState,
			HTMXAttrs: wx.HTMXAttrs{
				HxTrigger:     hxTrigger,
				HxPost:        hxPost,
				HxVals:        util.JSON(qq.actions.TagsFilterDialogPartial.Data(currentDirID)),
				LoadInPopover: true,
				HxOn:          hxOn,
			},
		},
	}
}

func (qq *ListDirPartial) filterPropertiesBtn(
	ctx ctxx.Context,
	listDirState *ListDirPartialState,
	currentDirID string,
) *wx.Container {
	// Count the number of active property filters
	activeFilterCount := len(listDirState.PropertyValues)

	chipState := wx.AssistChipStateDefault
	if activeFilterCount > 0 {
		chipState = wx.AssistChipStateHighlighted
	}

	hxTrigger := ""
	hxPost := qq.actions.PropertiesFilterDialogPartial.Endpoint()
	var hxOn *wx.HxOn
	if listDirState.ActiveSideSheet == qq.actions.PropertiesFilterDialogPartial.ID() { // is open
		if !ctx.VisitorCtx().IsHTMXRequest {
			hxTrigger = "load" // leads to strange issues if done on htmx requests
		} else {
			hxPost = ""
		}
		hxOn = event.CloseSideSheet.HxOn("click")
	} else { // closed
		hxOn = event.SideSheetToggled.HxOn("click")
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: "filterPropertiesBtn",
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data(currentDirID, "")),
			HxTarget: "#filterPropertiesBtn",
			HxSwap:   "outerHTML",
			HxTrigger: strings.Join([]string{
				event.PropertyFilterChanged.Handler(),
				event.SideSheetToggled.Handler(),
			}, ", "),
		},
		Child: &wx.AssistChip{
			IsActive:    listDirState.ActiveSideSheet == qq.actions.PropertiesFilterDialogPartial.ID(),
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
				HxVals:        util.JSON(qq.actions.PropertiesFilterDialogPartial.Data(currentDirID)),
				LoadInPopover: true,
				HxOn:          hxOn,
			},
		},
	}
}

func (qq *ListDirPartial) filterDocumentTypeBtn(
	ctx ctxx.Context,
	listDirState *ListDirPartialState,
	currentDirID string,
) *wx.Container {
	// TODO open on hover

	chipState := wx.AssistChipStateDefault
	if listDirState.DocumentTypeID != 0 {
		chipState = wx.AssistChipStateHighlighted
	}

	hxTrigger := ""
	hxPost := qq.actions.DocumentTypeFilterDialogPartial.Endpoint()
	var hxOn *wx.HxOn
	if listDirState.ActiveSideSheet == qq.actions.DocumentTypeFilterDialogPartial.ID() { // is open
		if !ctx.VisitorCtx().IsHTMXRequest {
			hxTrigger = "load" // leads to strange issues if done on htmx requests
		} else {
			hxPost = ""
		}
		hxOn = event.CloseSideSheet.HxOn("click")
	} else { // closed
		hxOn = event.SideSheetToggled.HxOn("click")
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: "filterDocumentTypeBtn",
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data(currentDirID, "")),
			HxTarget: "#filterDocumentTypeBtn",
			HxSwap:   "outerHTML",
			HxTrigger: strings.Join([]string{
				// TODO is this necessary (for all buttons) or are handlers on list enough?
				event.FilterTagsChanged.Handler(),
				event.DocumentTypeFilterChanged.Handler(),
				event.PropertyFilterChanged.Handler(),
				event.SideSheetToggled.Handler(),
			}, ", "),
		},
		Child: &wx.AssistChip{
			IsActive:    listDirState.ActiveSideSheet == qq.actions.DocumentTypeFilterDialogPartial.ID(),
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
				HxVals:        util.JSON(qq.actions.DocumentTypeFilterDialogPartial.Data(currentDirID)),
				LoadInPopover: true,
				HxOn:          hxOn,
			},
		},
	}
}
