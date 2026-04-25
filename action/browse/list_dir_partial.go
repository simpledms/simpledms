package browse

import (
	"fmt"
	"log"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/marcobeierer/go-core/db/entx"
	"github.com/marcobeierer/go-core/ui/uix/partial"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/model/common/fieldtype"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/uix/events"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	actionx2 "github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/marcobeierer/go-core/util/timex"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/property"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
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
	*actionx2.Config
}

func NewListDirPartial(
	infra *common.Infra,
	actions *Actions,
) *ListDirPartial {
	return &ListDirPartial{
		infra:            infra,
		actions:          actions,
		fileQueryService: NewListDirFileQueryService(infra),
		Config: actionx2.NewConfig(
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

func (qq *ListDirPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

		dir := qq.infra.FileRepo.GetX(ctx, data.CurrentDirID)
		return qq.infra.Renderer().Render(
			rw,
			ctx,
			&widget.View{
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

func (qq *ListDirPartial) WidgetHandler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
	fileID string,
	selectedFileID string,
) *widget.ListDetailLayout {
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
) *widget.ListDetailLayout {
	// dir := ctx.TenantCtx().TTx.File.GetX(ctx, fileID)
	dirWithParentx := ctx.TenantCtx().TTx.File.Query().WithParent().Where(file.PublicID(entx.NewCIText(fileID))).OnlyX(ctx)
	dirWithParent := qq.infra.FileRepo.GetXX(dirWithParentx)

	if dirWithParent.Data.IsDirectory == false {
		// TODO handle error... return container with error message for user? but should also be 404
		return &widget.ListDetailLayout{}
	}

	var children []widget.IWidget

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
		var breadcrumbs []widget.IWidget
		if dirWithParent.Data.ID > 0 {
			pathFiles := qq.infra.FileSystem().FileTree().PathFilesByFileIDX(ctx, dirWithParent.Data.ID)
			for qi, pathFile := range pathFiles {
				var breadcrumbLabel widget.IWidget
				if qi == 0 {
					// TODO home Icon
					breadcrumbLabel = &widget.Icon{
						Name: "home",
						Size: widget.IconSizeSmall,
					}
				} else {
					breadcrumbLabel = widget.Tu(pathFile.Name).SetWrap()
				}
				if qi != len(pathFiles)-1 {
					breadcrumbs = append(breadcrumbs, &widget.Link{
						Href:  route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, pathFile.PublicID.String()),
						Child: breadcrumbLabel,
					})
					breadcrumbs = append(breadcrumbs, widget.T("»")) // TODO use icon instead?
				} else {
					// last elem
					breadcrumbs = append(breadcrumbs, breadcrumbLabel)
				}
			}
		}
		if len(breadcrumbs) > 0 {
			children = append(children, &widget.StatusBar{
				Child: breadcrumbs,
			})
		}
	}

	list := &widget.Column{
		GapYSize: widget.Gap2,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost: qq.EndpointWithParams(actionx2.ResponseWrapperNone, "#"+qq.FileListID()),
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

	return &widget.ListDetailLayout{
		Widget: widget.Widget[widget.ListDetailLayout]{
			ID: qq.WrapperID(),
		},
		AppBar: qq.appBar(ctx, state, dirWithParent),
		List:   list,
	}
}

func (qq *ListDirPartial) tagsAndOptions(ctx ctxx.Context, state *ListDirPartialState, dir *filemodel.File) *widget.ChipBar {
	// TODO most used tags within folder, order alphabetically or by use?

	// childDirCount := dir.Data.QueryChildren().Where(file.IsDirectory(true)).CountX(ctx)
	currentDirID := dir.Data.PublicID

	children := qq.filters(ctx, state, currentDirID.String())

	return &widget.ChipBar{
		Widget: widget.Widget[widget.ChipBar]{
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
	dir *filemodel.File,
	data *ListDirPartialData,
	offset int,
) renderable.Renderable {
	fileListItems := qq.filesListItems(ctx, state, data, offset)

	var content widget.IWidget
	content = &widget.List{
		Children: fileListItems,
	}

	if len(fileListItems) == 0 {
		var widgets []widget.IWidget
		headline := widget.T("No files available yet.")

		if ctx.SpaceCtx().Space.IsFolderMode {
			headline = widget.T("No files or directories available yet.")
			widgets = append(
				widgets,
				qq.actions.MakeDirCmd.ModalLink(
					qq.actions.MakeDirCmd.Data(dir.Data.PublicID.String(), ""),
					[]widget.IWidget{
						&widget.Button{
							Icon:  widget.NewIcon("create_new_folder"),
							Label: widget.T("Create directory"),
						},
					},
					"#"+qq.actions.ListDirPartial.WrapperID(),
				),
			)
		}

		widgets = append(
			widgets,
			&widget.Link{
				HTMXAttrs: widget.HTMXAttrs{
					HxPost:        qq.actions.FileUploadDialogPartial.Endpoint(),
					HxVals:        util.JSON(qq.actions.FileUploadDialogPartial.Data(dir.Data.PublicID.String(), false)),
					LoadInPopover: true,
				},
				Child: &widget.Button{
					Icon:  widget.NewIcon("upload_file"),
					Label: widget.T("Upload file"),
				},
			},
		)

		content = &widget.EmptyState{
			Icon:     widget.NewIcon("description"),
			Headline: headline,
			// Description: NewText("There are no directories or files available yet, you can create"),
			Actions: widgets,
		}
	}

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
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
) []widget.IWidget {
	var fileListItems []widget.IWidget

	queryResult := qq.fileQueryService.Query(
		ctx,
		state,
		data,
		offset,
		qq.pageSize(),
		qq.applyPropertyFilter,
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
			fullPath = childParentFullPaths[child.ParentID]
		}

		fileListItems = append(fileListItems, qq.actions.FileListItemPartial.fileListItem(
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
		fileListItems = append(fileListItems, &widget.ListItem{
			Widget: widget.Widget[widget.ListItem]{
				ID: "listDirLoadMore",
			},
			Headline: widget.T("Loading more..."),
			HTMXAttrs: widget.HTMXAttrs{
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
	dir *filemodel.File,
) *widget.AppBar {
	var leadingButton widget.IWidget

	if dir.Data.ParentID != 0 {
		parent, err := dir.Parent(ctx)
		if err != nil {
			log.Println(err)
			panic(err)
		}
		leadingButton = &widget.IconButton{
			Icon:    "arrow_back",
			Tooltip: widget.T("Back to parent folder"),
			HTMXAttrs: widget.HTMXAttrs{
				HxGet:     route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, parent.Data.PublicID.String()),
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
		leadingButton = &widget.Icon{
			Name: "folder_open",
		}
	} else {
		leadingButton = &widget.Icon{
			Name: "folder_open", // TODO folder_open or hub or home?
		}
	}

	supportingText := widget.T("Search")
	supportingTextAltMobile := widget.Tu(dir.Data.Name)
	if ctx.SpaceCtx().Space.IsFolderMode && dir.Data.ID != ctx.SpaceCtx().SpaceRootDir().ID {
		supportingText = widget.Tf("Search in «%s»", dir.Data.Name)
	}

	return &widget.AppBar{
		Leading:          leadingButton,
		LeadingAltMobile: partial.NewMainMenu(ctx, qq.infra),
		Title:            widget.Tu(dir.Data.Name),
		// Actions:          actions,
		Search: &widget.Search{
			Widget: widget.Widget[widget.Search]{
				ID: "search",
			},
			Name:                    "SearchQuery",
			Value:                   state.searchQueryRaw,
			SupportingText:          supportingText,
			SupportingTextAltMobile: supportingTextAltMobile,
			HTMXAttrs: widget.HTMXAttrs{
				HxOn: event.SearchQueryUpdated.HxOnWithQueryParam("input", "q"),
			},
		},
	}
}

func (qq *ListDirPartial) filters(
	ctx ctxx.Context,
	listDirState *ListDirPartialState,
	currentDirID string,
) []widget.IWidget {
	// TODO show only if there are dirs or files in result? would require to check complete query,
	//		not just first 25 results...
	chips := []widget.IWidget{
		qq.filterDocumentTypeBtn(ctx, listDirState, currentDirID),
		qq.filterPropertiesBtn(ctx, listDirState, currentDirID),
		// TODO open on hover
		qq.filterTagsBtn(ctx, listDirState, currentDirID),
	}

	// TODO show only if filter is active
	chips = append(chips,
		&widget.AssistChip{
			// TODO choose another styling, that it is not as prominent as the others
			Label: widget.Tf("Reset"), // just Reset because it also resets search
			HTMXAttrs: widget.HTMXAttrs{
				HxGet:     route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, currentDirID), // TODO or pass in href?
				HxHeaders: autil.ResetStateHeader(),
				HxOn:      events.CloseSideSheet.HxOn("click"),
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
) *widget.Container {
	chipState := widget.AssistChipStateDefault
	if len(listDirState.ListFilterTagsPartialState.CheckedTagIDs) > 0 {
		chipState = widget.AssistChipStateHighlighted
	}

	hxTrigger := ""
	hxPost := qq.actions.TagsFilterDialogPartial.Endpoint()
	var hxOn *widget.HxOn
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
		hxOn = events.CloseSideSheet.HxOn("click")
	} else { // closed
		/*hxOn = &wx.HxOn{
			Event:   "click",
			Handler: "document.querySelectorAll('.js-side-sheet-dialog').forEach(elem => elem.toggleCustom())",
		}*/
		// hxOn = event.SideSheetToggled.UnsafeHxOnWithQueryParamAndValue("click", "side_sheet", qq.actions.TagsFilterDialogPartial.ID())
		hxOn = events.SideSheetToggled.HxOn("click")
	}

	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: "filterTagsBtn",
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data(currentDirID, "")),
			HxTarget: "#filterTagsBtn",
			HxSwap:   "outerHTML",
			HxTrigger: strings.Join([]string{
				event.FilterTagsChanged.Handler(),
				events.SideSheetToggled.Handler(),
			}, ", "),
		},
		Child: &widget.AssistChip{
			IsActive:    listDirState.ActiveSideSheet == qq.actions.TagsFilterDialogPartial.ID(),
			Label:       widget.Tf("Tags"),
			LeadingIcon: "label",
			Badge: &widget.Badge{
				IsInline: true,
				Value:    len(listDirState.ListFilterTagsPartialState.CheckedTagIDs),
			},
			State: chipState,
			HTMXAttrs: widget.HTMXAttrs{
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
) *widget.Container {
	// Count the number of active property filters
	activeFilterCount := len(listDirState.PropertyValues)

	chipState := widget.AssistChipStateDefault
	if activeFilterCount > 0 {
		chipState = widget.AssistChipStateHighlighted
	}

	hxTrigger := ""
	hxPost := qq.actions.PropertiesFilterDialogPartial.Endpoint()
	var hxOn *widget.HxOn
	if listDirState.ActiveSideSheet == qq.actions.PropertiesFilterDialogPartial.ID() { // is open
		if !ctx.VisitorCtx().IsHTMXRequest {
			hxTrigger = "load" // leads to strange issues if done on htmx requests
		} else {
			hxPost = ""
		}
		hxOn = events.CloseSideSheet.HxOn("click")
	} else { // closed
		hxOn = events.SideSheetToggled.HxOn("click")
	}

	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: "filterPropertiesBtn",
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data(currentDirID, "")),
			HxTarget: "#filterPropertiesBtn",
			HxSwap:   "outerHTML",
			HxTrigger: strings.Join([]string{
				event.PropertyFilterChanged.Handler(),
				events.SideSheetToggled.Handler(),
			}, ", "),
		},
		Child: &widget.AssistChip{
			IsActive:    listDirState.ActiveSideSheet == qq.actions.PropertiesFilterDialogPartial.ID(),
			Label:       widget.Tf("Fields"),
			LeadingIcon: "tune", // tune or assignment
			Badge: &widget.Badge{
				IsInline: true,
				Value:    activeFilterCount,
			},
			State: chipState,
			HTMXAttrs: widget.HTMXAttrs{
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
) *widget.Container {
	// TODO open on hover

	chipState := widget.AssistChipStateDefault
	if listDirState.DocumentTypeID != 0 {
		chipState = widget.AssistChipStateHighlighted
	}

	hxTrigger := ""
	hxPost := qq.actions.DocumentTypeFilterDialogPartial.Endpoint()
	var hxOn *widget.HxOn
	if listDirState.ActiveSideSheet == qq.actions.DocumentTypeFilterDialogPartial.ID() { // is open
		if !ctx.VisitorCtx().IsHTMXRequest {
			hxTrigger = "load" // leads to strange issues if done on htmx requests
		} else {
			hxPost = ""
		}
		hxOn = events.CloseSideSheet.HxOn("click")
	} else { // closed
		hxOn = events.SideSheetToggled.HxOn("click")
	}

	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: "filterDocumentTypeBtn",
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data(currentDirID, "")),
			HxTarget: "#filterDocumentTypeBtn",
			HxSwap:   "outerHTML",
			HxTrigger: strings.Join([]string{
				// TODO is this necessary (for all buttons) or are handlers on list enough?
				event.FilterTagsChanged.Handler(),
				event.DocumentTypeFilterChanged.Handler(),
				event.PropertyFilterChanged.Handler(),
				events.SideSheetToggled.Handler(),
			}, ", "),
		},
		Child: &widget.AssistChip{
			IsActive:    listDirState.ActiveSideSheet == qq.actions.DocumentTypeFilterDialogPartial.ID(),
			Label:       widget.Tf("Document type"),
			LeadingIcon: "category",
			State:       chipState,
			/*TODO add back
			Badge: &wx.Badge{
				IsInline: true,
				// Value:    len(listDirState.CheckedTagIDs),
			},*/
			HTMXAttrs: widget.HTMXAttrs{
				HxTrigger:     hxTrigger,
				HxPost:        hxPost,
				HxVals:        util.JSON(qq.actions.DocumentTypeFilterDialogPartial.Data(currentDirID)),
				LoadInPopover: true,
				HxOn:          hxOn,
			},
		},
	}
}

func (qq *ListDirPartial) applyPropertyFilter(ctx ctxx.Context, query *enttenant.FileQuery, state *ListDirPartialState) *enttenant.FileQuery {
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
			switch propertyFilter.Operator {
			case operatorValueEquals.String():
				value, err := timex.ParseDate(propertyFilter.Value)
				if err != nil {
					log.Println(err)
					continue
				}
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.DateValue(value),
				))
			case operatorValueGreaterThan.String():
				value, err := timex.ParseDate(propertyFilter.Value)
				if err != nil {
					log.Println(err)
					continue
				}
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.DateValueGT(value),
				))
			case operatorValueLessThan.String():
				value, err := timex.ParseDate(propertyFilter.Value)
				if err != nil {
					log.Println(err)
					continue
				}
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.DateValueLT(value),
				))
			case operatorValueBetween.String():
				startDate := ""
				endDate := ""
				if propertyFilter.Value != "" {
					parts := strings.SplitN(propertyFilter.Value, ",", 2)
					startDate = parts[0]
					if len(parts) > 1 {
						endDate = parts[1]
					}
				}

				if startDate != "" {
					value, err := timex.ParseDate(startDate)
					if err != nil {
						log.Println(err)
						continue
					}
					query = query.Where(file.HasPropertyAssignmentWith(
						filepropertyassignment.PropertyID(propertyFilter.PropertyID),
						filepropertyassignment.DateValueGTE(value),
					))
				}

				if endDate != "" {
					value, err := timex.ParseDate(endDate)
					if err != nil {
						log.Println(err)
						continue
					}
					query = query.Where(file.HasPropertyAssignmentWith(
						filepropertyassignment.PropertyID(propertyFilter.PropertyID),
						filepropertyassignment.DateValueLTE(value),
					))
				}
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
