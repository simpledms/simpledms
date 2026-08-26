package inbox

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/action/browse"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entquery"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	"github.com/simpledms/simpledms/model/main/filelistpreference"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/sqlutil"
)

type FilesListPartialData struct {
	SelectedFileID string
}

const (
	sortByNewestFirst = "newestFirst"
	sortByOldestFirst = "oldestFirst"
	sortByName        = "name"
	sortByRank        = "rank"
)

type FilesListPartialState struct {
	SearchQuery string `url:"q,omitempty"`
	// used in JS, thus don't change URL and as param name below
	ActiveSideSheet string   `url:"side_sheet,omitempty"`
	SortBy          string   `url:"sort_by,omitempty"` // TODO enum
	SourceValues    []string `url:"source,omitempty"`
}

func (qq *FilesListPartialState) isSortedByDate() bool {
	return qq.SortBy == "" || qq.SortBy == sortByNewestFirst || qq.SortBy == sortByOldestFirst
}

func (qq *FilesListPartialState) hasActiveSearch() bool {
	return sqlutil.FTSSafeAndQuery(qq.SearchQuery, 300) != ""
}

func (qq *FilesListPartialState) normalizeSortBy() {
	if qq.SortBy == sortByRank && !qq.hasActiveSearch() {
		qq.SortBy = ""
	}
}

func (qq *FilesListPartialState) sources() ([]filesource.FileSource, error) {
	if len(qq.SourceValues) == 0 {
		return nil, nil
	}
	sources := make([]filesource.FileSource, 0, len(qq.SourceValues))
	seen := map[filesource.FileSource]bool{}
	for _, value := range qq.SourceValues {
		source, err := filesource.FileSourceString(value)
		if err != nil {
			return nil, err
		}
		if seen[source] {
			continue
		}
		seen[source] = true
		sources = append(sources, source)
	}
	return sources, nil
}

func (qq *FilesListPartialState) hasSource(source filesource.FileSource) bool {
	for _, value := range qq.SourceValues {
		if value == source.String() {
			return true
		}
	}
	return false
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
	// Keep this distinct from Browse IDs because morph navigation can retain HTMX listeners.
	return "inboxListWrapper"
}

func (qq *FilesListPartial) FileListID() string {
	return "inboxFileList"
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

	state := autil.StateX[InboxPageState](rw, req)
	state.FilesListPartialState.normalizeSortBy()
	if _, err := state.FilesListPartialState.sources(); err != nil {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Invalid source filter.")
	}

	hxTarget := req.URL.Query().Get("hx-target")
	if hxTarget == "#"+qq.FileListID() {
		rw.Header().Set("HX-Replace-Url", route.InboxRootWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID))
		fileList := qq.filesList(
			ctx,
			state,
			data,
		)

		return qq.infra.Renderer().Render(
			rw,
			ctx,
			&widget.View{
				Children: []widget.IWidget{
					fileList,
					qq.sortMenuButton(ctx, &state.FilesListPartialState, true),
				},
			},
		)
	}
	if req.Header.Get("Hx-Target") == "inboxLoadMore" {
		offset, err := offsetFromRequest(req)
		if err != nil {
			log.Println(err)
			return err
		}

		children, hasMore := qq.filesPage(ctx, state, offset)
		return qq.infra.Renderer().Render(
			rw,
			ctx,
			&widget.View{
				Children: qq.filesListItemsFromFiles(ctx, state, data, offset, children, hasMore),
			},
		)
	}
	if req.Header.Get("Hx-Target") == "inboxLoadMoreTable" {
		offset, err := offsetFromRequest(req)
		if err != nil {
			log.Println(err)
			return err
		}

		children, hasMore := qq.filesPage(ctx, state, offset)
		preferences := filelistpreference.NewFileListPreferencesFromValue(ctx.MainCtx().Account.FileListPreferences)
		return qq.infra.Renderer().Render(
			rw,
			ctx,
			&widget.View{
				Children: qq.fileTable(ctx, data, offset, children, hasMore, preferences).Rows,
			},
		)
	}

	return qq.infra.Renderer().Render(rw, ctx, qq.Widget(ctx, state, data.SelectedFileID))
}

func offsetFromRequest(req *httpx.Request) (int, error) {
	offsetStr := req.URL.Query().Get("offset")
	if offsetStr == "" {
		return 0, nil
	}
	return strconv.Atoi(offsetStr)
}

func (qq *FilesListPartial) WidgetHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	selectedFileID string,
) *widget.ListDetailLayout {
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
) *widget.ListDetailLayout {
	state.FilesListPartialState.normalizeSortBy()

	var children []widget.IWidget
	var appBar *widget.AppBar

	if selectedFileID == "" {
		appBar = qq.appBar(ctx, state)
	} else {
		appBar = &widget.AppBar{
			Title:   widget.T("Inbox"),
			Leading: widget.NewIcon("inbox"),
		}
	}

	children = append(children,
		qq.filesList(
			ctx,
			state,
			qq.Data(selectedFileID),
		),
	)

	list := &widget.Column{
		Widget: widget.Widget[widget.Column]{
			ID: qq.WrapperID(),
		},
		GapYSize: widget.Gap2,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.EndpointWithParams(actionx.ResponseWrapperNone, ""),
			HxVals:   util.JSON(qq.Data(selectedFileID)), // overrides form fields, must be added via HxInclude
			HxTarget: "#innerContent",                    // not just fileList because of sortBy selection
			HxSwap:   "innerHTML",
			HxTrigger: strings.Join([]string{
				// see comment on HTMXAttrs on ScrollableContent (FileList)
				event.SortByUpdated.HandlerWithModifier("delay:100ms"), // TODO delay necessary?
				event.SourceFilterChanged.HandlerWithModifier("delay:100ms"),
				event.FileMoved.Handler(),   // because it also has to close details
				event.FileDeleted.Handler(), // because it also has to close details
			}, ", "),
			HxInclude: "#search,#sortBy,#inboxSourceFilter",
		},
		Children: children,
	}
	return &widget.ListDetailLayout{
		AppBar: appBar,
		List:   list,
	}
}

func (qq *FilesListPartial) filesList(
	ctx ctxx.Context,
	state *InboxPageState,
	data *FilesListPartialData,
) renderable.Renderable {
	children, hasMore := qq.filesPage(ctx, state, 0)
	preferences := filelistpreference.NewFileListPreferencesFromValue(ctx.MainCtx().Account.FileListPreferences)
	fileListItems := qq.filesListItemsFromFiles(ctx, state, data, 0, children, hasMore)

	var content widget.IWidget
	content = &widget.List{
		Children: fileListItems,
	}
	if preferences.IsTable() && len(children) > 0 {
		content = qq.fileTable(ctx, data, 0, children, hasMore, preferences)
	}

	if len(children) == 0 {
		content = &widget.EmptyState{
			Icon:     widget.NewIcon("description"),
			Headline: widget.T("No files available yet."),
			// Description: NewText("There are no directories or files available yet, you can create"),
			Actions: []widget.IWidget{
				&widget.Link{
					HTMXAttrs: widget.HTMXAttrs{
						HxPost: qq.actions.Browse.FileUploadDialogPartial.Endpoint(),
						HxVals: util.JSON(qq.actions.Browse.FileUploadDialogPartial.Data(
							ctx.SpaceCtx().SpaceRootDir().PublicID.String(),
							true,
						)),
						LoadInPopover: true,
					},
					Child: &widget.Button{
						Icon:  widget.NewIcon("upload_file"),
						Label: widget.T("Upload file"),
					},
				},
			},
		}
	}

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: qq.FileListID(),
		},
		Children: content,
		// must be on ScrollableContent and not directly on wx.List because otherwise page breaks
		// if a search has no results and empty state is rendered without HTMXAttrs
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.EndpointWithParams(actionx.ResponseWrapperNone, "#"+qq.FileListID()),
			HxVals:   util.JSON(data), // overrides form fields, must be added via HxInclude
			HxTarget: "#" + qq.FileListID(),
			HxSwap:   "outerHTML",
			HxSync:   "this:replace",
			HxTrigger: strings.Join([]string{
				// SortByUpdated is handled separately because it has to update sortby
				// context menu and the files list; sortby context menu is part of appbar and thus
				// updating the app bar while using the search input leads to flickering and
				// loss of input while typing
				event.SearchQueryUpdated.HandlerWithModifier("delay:100ms"),
				event.SourceFilterChanged.HandlerWithModifier("delay:100ms"),
				event.FileUploaded.Handler(),
				event.ZIPArchiveUnzipped.Handler(), // TODO necessary?
				event.FileDeleted.Handler(),
				event.FileUpdated.Handler(),
			}, ", "),
			HxInclude: "#search,#sortBy,#inboxSourceFilter",
		},
	}
}

func (qq *FilesListPartial) pageSize() int {
	return 50
}

func (qq *FilesListPartial) filesPage(
	ctx ctxx.Context,
	state *InboxPageState,
	offset int,
) ([]*enttenant.File, bool) {
	children := qq.filesQuery(ctx, state).
		Offset(offset).
		Limit(qq.pageSize() + 1).
		AllX(ctx)
	hasMore := len(children) > qq.pageSize()
	if hasMore {
		children = children[:qq.pageSize()]
	}
	return children, hasMore
}

func (qq *FilesListPartial) filesListItemsFromFiles(
	ctx ctxx.Context,
	state *InboxPageState,
	data *FilesListPartialData,
	offset int,
	children []*enttenant.File,
	hasMore bool,
) []widget.IWidget {
	fileListItems := make([]widget.IWidget, 0, len(children)+1)
	for _, child := range children {
		fileListItems = append(fileListItems, qq.actions.FileListItemPartial.Widget(
			ctx,
			// route.InboxWithState(state),
			route.Inbox,
			child,
			child.PublicID.String() == data.SelectedFileID,
			state.FilesListPartialState.isSortedByDate(),
		))
	}

	if hasMore {
		fileListItems = append(fileListItems, &widget.ListItem{
			Widget: widget.Widget[widget.ListItem]{
				ID: "inboxLoadMore",
			},
			Headline: widget.T("Loading more..."),
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:    qq.Endpoint() + "?offset=" + strconv.Itoa(offset+qq.pageSize()),
				HxVals:    util.JSON(data),
				HxTrigger: "intersect once",
				HxTarget:  "#inboxLoadMore",
				HxSwap:    "outerHTML",
				HxInclude: "#search,#sortBy,#inboxSourceFilter",
			},
		})
	}

	return fileListItems
}

// LIMIT must be applied by caller
func (qq *FilesListPartial) filesQuery(ctx ctxx.Context, state *InboxPageState) *enttenant.FileQuery {
	state.FilesListPartialState.normalizeSortBy()

	searchResultQuery := ctx.TenantCtx().TTx.File.Query()
	searchQuery := sqlutil.FTSSafeAndQuery(state.SearchQuery, 300)
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

	if searchQuery != "" {
		searchResultQuery = searchResultQuery.Where(
			file.SpaceID(ctx.SpaceCtx().Space.ID),
			file.IsInInbox(true),
			file.IsDirectory(false),
			/*file.HasSpaceAssignmentWith(
				spacefileassignment.SpaceID(ctx.SpaceCtx().Space.ID),
				spacefileassignment.IsInInbox(true),
			),*/
		)
	} else {
		searchResultQuery = searchResultQuery.Where(
			file.SpaceID(ctx.SpaceCtx().Space.ID),
			entquery.FileIsInInbox(true),
			entquery.FileIsDirectory(false),
			/*file.HasSpaceAssignmentWith(
				spacefileassignment.SpaceID(ctx.SpaceCtx().Space.ID),
				spacefileassignment.IsInInbox(true),
			),*/
		)
	}

	sources, err := state.FilesListPartialState.sources()
	if err != nil {
		log.Println(err)
	} else if len(sources) > 0 {
		searchResultQuery = searchResultQuery.Where(file.SourceIn(sources...))
	}

	if searchQuery != "" {
		searchResultQuery = searchResultQuery.Where(
			func(qs *sql.Selector) {
				entquery.ApplyFileSearchCandidateFilterWithDirectory(
					qs,
					searchQuery,
					ctx.SpaceCtx().Space.ID,
					true,
					false,
				)
			},
		)
	}

	switch state.SortBy {
	case sortByRank:
		searchResultQuery = searchResultQuery.Order(
			entquery.OrderFileSearchRankWithDirectory(
				searchQuery,
				ctx.SpaceCtx().Space.ID,
				true,
				false,
			),
			file.ByCreatedAt(sql.OrderDesc()),
		)
	case sortByName:
		searchResultQuery = searchResultQuery.Order(file.ByName())
	case sortByOldestFirst:
		searchResultQuery = searchResultQuery.Order(file.ByCreatedAt())
	case sortByNewestFirst:
		fallthrough
	default:
		searchResultQuery = searchResultQuery.Order(file.ByCreatedAt(sql.OrderDesc()))
	}
	// searchResultQuery = searchResultQuery.Order(file.ByName())

	return searchResultQuery
}

func (qq *FilesListPartial) appBar(ctx ctxx.Context, state *InboxPageState) *widget.AppBar {
	return &widget.AppBar{
		Leading:          widget.NewIcon("inbox"),
		LeadingAltMobile: partial.NewNavigationRailToggle(),
		Title:            widget.T("Inbox"),
		Actions: []widget.IWidget{
			qq.fileListViewButton(ctx),
			qq.sortMenuButton(ctx, &state.FilesListPartialState, false),
			qq.sourceFilterButton(),
		},
		Search: &widget.Search{
			Widget: widget.Widget[widget.Search]{
				ID: "search",
			},
			Name:           "SearchQuery",
			Value:          state.SearchQuery,
			SupportingText: widget.Tf("Search in «Inbox»"),
			HTMXAttrs: widget.HTMXAttrs{
				HxOn: event.SearchQueryUpdated.HxOnWithSearchQueryParamAndRankSortReset("input", "q"),
			},
		},
	}
}

func (qq *FilesListPartial) sortMenuButton(
	ctx ctxx.Context,
	state *FilesListPartialState,
	isOOB bool,
) *widget.Container {
	swapOOB := ""
	if isOOB {
		swapOOB = "outerHTML"
	}

	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: "inboxSortFilesButton",
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxSwapOOB: swapOOB,
		},
		Child: &widget.IconButton{
			Icon:     "sort",
			Tooltip:  widget.T("Sort files"),
			Children: NewSortListContextMenuWidget(qq.actions).Widget(ctx, state),
		},
	}
}

func (qq *FilesListPartial) sourceFilterButton() *widget.IconButton {
	return &widget.IconButton{
		Icon:    "filter_alt",
		Tooltip: widget.T("Filter by source"),
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:        qq.actions.SourceFilterDialog.Endpoint(),
			LoadInPopover: true,
		},
	}
}

func (qq *FilesListPartial) fileListViewButton(ctx ctxx.Context) *widget.IconButton {
	preferences := filelistpreference.NewFileListPreferencesFromValue(ctx.MainCtx().Account.FileListPreferences)
	return &widget.IconButton{
		Icon:    "view_agenda",
		Tooltip: widget.T("Change file list view"),
		Children: qq.fileListViewMenu(
			ctx,
			preferences,
			autil.QueryHeader(qq.Endpoint(), qq.Data("")),
		),
	}
}

func (qq *FilesListPartial) fileListViewMenu(
	ctx ctxx.Context,
	preferences *filelistpreference.FileListPreferences,
	hxHeaders template.JS,
) *widget.Menu {
	items := []*widget.MenuItem{
		qq.fileListViewMenuItem(widget.T("List"), "list", preferences.ViewMode == filelistpreference.FileListViewModeList, hxHeaders),
		qq.fileListViewMenuItem(widget.T("Table"), "table", preferences.ViewMode == filelistpreference.FileListViewModeTable, hxHeaders),
	}
	if !preferences.IsTable() {
		return &widget.Menu{
			Widget: widget.Widget[widget.Menu]{
				ID: "inboxFileListViewMenu",
			},
			Position: widget.PositionLeft,
			Items:    items,
		}
	}

	items = append(items, &widget.MenuItem{IsDivider: true})

	for _, column := range []struct {
		column filelistpreference.FileListColumn
		label  *widget.Text
	}{
		{filelistpreference.FileListColumnName, widget.T("Name")},
		{filelistpreference.FileListColumnSource, widget.T("Source")},
		{filelistpreference.FileListColumnOriginalFilename, widget.T("Original filename")},
		{filelistpreference.FileListColumnDocumentType, widget.T("Type")},
		{filelistpreference.FileListColumnMetadata, widget.T("Metadata")},
		{filelistpreference.FileListColumnDate, widget.T("Date")},
		{filelistpreference.FileListColumnSize, widget.T("Size")},
	} {
		items = append(items, qq.fileListColumnMenuItem(
			column.label,
			column.column.String(),
			preferences.HasBuiltInColumn(column.column),
			hxHeaders,
		))
	}

	spaceColumns := preferences.SpaceColumnsFor(ctx.SpaceCtx().SpaceID)
	showTags := !spaceColumns.ShowTags
	items = append(items, qq.fileListTagsMenuItem(widget.T("Tags"), showTags, spaceColumns.ShowTags, hxHeaders))

	tagGroups := ctx.SpaceCtx().Space.QueryTags().
		Where(tag.TypeEQ(tagtype.Group)).
		Order(tag.ByName()).
		AllX(ctx)
	if len(tagGroups) > 0 {
		items = append(items, &widget.MenuItem{IsDivider: true})
	}
	for _, tagGroup := range tagGroups {
		items = append(items, qq.fileListTagGroupMenuItem(
			widget.Tu(tagGroup.Name),
			tagGroup.ID,
			spaceColumns.HasTagGroupID(tagGroup.ID),
			hxHeaders,
		))
	}

	properties := ctx.SpaceCtx().TTx.Property.Query().Order(property.ByName()).AllX(ctx)
	if len(properties) > 0 {
		items = append(items, &widget.MenuItem{IsDivider: true})
	}
	for _, propertyx := range properties {
		items = append(items, qq.fileListPropertyMenuItem(
			widget.Tu(propertyx.Name),
			propertyx.ID,
			spaceColumns.HasPropertyID(propertyx.ID),
			hxHeaders,
		))
	}

	return &widget.Menu{
		Widget: widget.Widget[widget.Menu]{
			ID: "inboxFileListViewMenu",
		},
		Position: widget.PositionLeft,
		Items:    items,
	}
}

func (qq *FilesListPartial) fileListViewMenuItem(
	label *widget.Text,
	viewMode string,
	isSelected bool,
	hxHeaders template.JS,
) *widget.MenuItem {
	data := qq.actions.Browse.UpdateFileListPreferencesCmd.Data()
	data.ViewMode = viewMode
	return &widget.MenuItem{
		Label:          label,
		RadioGroupName: "FileListViewMode",
		RadioValue:     viewMode,
		IsSelected:     isSelected,
		HTMXAttrs:      qq.fileListPreferencesMenuItemAttrs(data, hxHeaders),
	}
}

func (qq *FilesListPartial) fileListColumnMenuItem(
	label *widget.Text,
	column string,
	isChecked bool,
	hxHeaders template.JS,
) *widget.MenuItem {
	data := qq.actions.Browse.UpdateFileListPreferencesCmd.Data()
	data.BuiltInColumn = column
	return &widget.MenuItem{
		Label:         label,
		CheckboxName:  "FileListColumn",
		CheckboxValue: column,
		IsChecked:     isChecked,
		HTMXAttrs:     qq.fileListPreferencesMenuItemAttrs(data, hxHeaders),
	}
}

func (qq *FilesListPartial) fileListTagsMenuItem(
	label *widget.Text,
	showTags bool,
	isChecked bool,
	hxHeaders template.JS,
) *widget.MenuItem {
	data := qq.actions.Browse.UpdateFileListPreferencesCmd.Data()
	data.ShowTags = &showTags
	return &widget.MenuItem{
		Label:         label,
		CheckboxName:  "FileListTags",
		CheckboxValue: "tags",
		IsChecked:     isChecked,
		HTMXAttrs:     qq.fileListPreferencesMenuItemAttrs(data, hxHeaders),
	}
}

func (qq *FilesListPartial) fileListPropertyMenuItem(
	label *widget.Text,
	propertyID int64,
	isChecked bool,
	hxHeaders template.JS,
) *widget.MenuItem {
	data := qq.actions.Browse.UpdateFileListPreferencesCmd.Data()
	data.PropertyID = propertyID
	return &widget.MenuItem{
		Label:         label,
		CheckboxName:  "FileListProperty",
		CheckboxValue: strconv.FormatInt(propertyID, 10),
		IsChecked:     isChecked,
		HTMXAttrs:     qq.fileListPreferencesMenuItemAttrs(data, hxHeaders),
	}
}

func (qq *FilesListPartial) fileListTagGroupMenuItem(
	label *widget.Text,
	tagGroupID int64,
	isChecked bool,
	hxHeaders template.JS,
) *widget.MenuItem {
	data := qq.actions.Browse.UpdateFileListPreferencesCmd.Data()
	data.TagGroupID = tagGroupID
	return &widget.MenuItem{
		Label:         label,
		CheckboxName:  "FileListTagGroup",
		CheckboxValue: strconv.FormatInt(tagGroupID, 10),
		IsChecked:     isChecked,
		HTMXAttrs:     qq.fileListPreferencesMenuItemAttrs(data, hxHeaders),
	}
}

func (qq *FilesListPartial) fileListPreferencesMenuItemAttrs(
	data *browse.UpdateFileListPreferencesCmdData,
	hxHeaders template.JS,
) widget.HTMXAttrs {
	return widget.HTMXAttrs{
		HxPost:    qq.actions.Browse.UpdateFileListPreferencesCmd.Endpoint(),
		HxVals:    util.JSON(data),
		HxHeaders: hxHeaders,
		HxTarget:  "#innerContent",
		HxSwap:    "innerHTML",
	}
}
