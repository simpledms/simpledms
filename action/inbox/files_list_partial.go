package inbox

import (
	"html/template"
	"log"
	"strconv"
	"strings"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/action/browse"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/main/filelistpreference"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
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

func (qq *FilesListPartialState) isSortedByDate() bool {
	return qq.SortBy == "" || qq.SortBy == "newestFirst" || qq.SortBy == "oldestFirst"
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
				event.FileDeleted.Handler(),                            // because it also has to close details
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

	searchResultQuery := qq.filesQuery(ctx, state)
	// TODO .Limit(25) // needs hint if enabled
	children := searchResultQuery.AllX(ctx)
	preferences := filelistpreference.NewFileListPreferencesFromValue(ctx.MainCtx().Account.FileListPreferences)

	for _, child := range children {
		if child.IsDirectory {
			continue
		}

		fileListItems = append(fileListItems, qq.actions.FileListItemPartial.Widget(
			ctx,
			// route.InboxWithState(state),
			route.Inbox,
			child,
			child.PublicID.String() == data.SelectedFileID,
			state.FilesListPartialState.isSortedByDate(),
		))
	}

	var content wx.IWidget
	content = &wx.List{
		Children: fileListItems,
	}
	if preferences.IsTable() && len(fileListItems) > 0 {
		content = &wx.View{
			Children: []wx.IWidget{
				&wx.List{
					Children:      fileListItems,
					HideOnDesktop: true,
				},
				qq.fileTable(ctx, data, children, preferences),
			},
		}
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

// LIMIT must be applied by caller
func (qq *FilesListPartial) filesQuery(ctx ctxx.Context, state *InboxPageState) *enttenant.FileQuery {
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

func (qq *FilesListPartial) appBar(ctx ctxx.Context, state *InboxPageState) *wx.AppBar {
	return &wx.AppBar{
		Leading:          wx.NewIcon("inbox"),
		LeadingAltMobile: partial.NewMainMenu(ctx, qq.infra),
		Title:            wx.T("Inbox"),
		Actions: []wx.IWidget{
			qq.fileListViewButton(ctx),
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

func (qq *FilesListPartial) fileListViewButton(ctx ctxx.Context) *wx.IconButton {
	preferences := filelistpreference.NewFileListPreferencesFromValue(ctx.MainCtx().Account.FileListPreferences)
	return &wx.IconButton{
		Icon:    "view_agenda",
		Tooltip: wx.T("Change file list view"),
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
) *wx.Menu {
	items := []*wx.MenuItem{
		qq.fileListViewMenuItem(wx.T("List"), "list", preferences.ViewMode == filelistpreference.FileListViewModeList, hxHeaders),
		qq.fileListViewMenuItem(wx.T("Table"), "table", preferences.ViewMode == filelistpreference.FileListViewModeTable, hxHeaders),
		&wx.MenuItem{IsDivider: true},
	}

	for _, column := range []struct {
		column filelistpreference.FileListColumn
		label  *wx.Text
	}{
		{filelistpreference.FileListColumnName, wx.T("Name")},
		{filelistpreference.FileListColumnDocumentType, wx.T("Type")},
		{filelistpreference.FileListColumnMetadata, wx.T("Metadata")},
		{filelistpreference.FileListColumnDate, wx.T("Date")},
		{filelistpreference.FileListColumnSize, wx.T("Size")},
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
	items = append(items, qq.fileListTagsMenuItem(wx.T("Tags"), showTags, spaceColumns.ShowTags, hxHeaders))

	tagGroups := ctx.SpaceCtx().Space.QueryTags().
		Where(tag.TypeEQ(tagtype.Group)).
		Order(tag.ByName()).
		AllX(ctx)
	if len(tagGroups) > 0 {
		items = append(items, &wx.MenuItem{IsDivider: true})
	}
	for _, tagGroup := range tagGroups {
		items = append(items, qq.fileListTagGroupMenuItem(
			wx.Tu(tagGroup.Name),
			tagGroup.ID,
			spaceColumns.HasTagGroupID(tagGroup.ID),
			hxHeaders,
		))
	}

	properties := ctx.SpaceCtx().TTx.Property.Query().Order(property.ByName()).AllX(ctx)
	if len(properties) > 0 {
		items = append(items, &wx.MenuItem{IsDivider: true})
	}
	for _, propertyx := range properties {
		items = append(items, qq.fileListPropertyMenuItem(
			wx.Tu(propertyx.Name),
			propertyx.ID,
			spaceColumns.HasPropertyID(propertyx.ID),
			hxHeaders,
		))
	}

	return &wx.Menu{
		Widget: wx.Widget[wx.Menu]{
			ID: "inboxFileListViewMenu",
		},
		Position: wx.PositionLeft,
		Items:    items,
	}
}

func (qq *FilesListPartial) fileListViewMenuItem(
	label *wx.Text,
	viewMode string,
	isSelected bool,
	hxHeaders template.JS,
) *wx.MenuItem {
	data := qq.actions.Browse.UpdateFileListPreferencesCmd.Data()
	data.ViewMode = viewMode
	return &wx.MenuItem{
		Label:          label,
		RadioGroupName: "FileListViewMode",
		RadioValue:     viewMode,
		IsSelected:     isSelected,
		HTMXAttrs:      qq.fileListPreferencesMenuItemAttrs(data, hxHeaders),
	}
}

func (qq *FilesListPartial) fileListColumnMenuItem(
	label *wx.Text,
	column string,
	isChecked bool,
	hxHeaders template.JS,
) *wx.MenuItem {
	data := qq.actions.Browse.UpdateFileListPreferencesCmd.Data()
	data.BuiltInColumn = column
	return &wx.MenuItem{
		Label:         label,
		CheckboxName:  "FileListColumn",
		CheckboxValue: column,
		IsChecked:     isChecked,
		HTMXAttrs:     qq.fileListPreferencesMenuItemAttrs(data, hxHeaders),
	}
}

func (qq *FilesListPartial) fileListTagsMenuItem(
	label *wx.Text,
	showTags bool,
	isChecked bool,
	hxHeaders template.JS,
) *wx.MenuItem {
	data := qq.actions.Browse.UpdateFileListPreferencesCmd.Data()
	data.ShowTags = &showTags
	return &wx.MenuItem{
		Label:         label,
		CheckboxName:  "FileListTags",
		CheckboxValue: "tags",
		IsChecked:     isChecked,
		HTMXAttrs:     qq.fileListPreferencesMenuItemAttrs(data, hxHeaders),
	}
}

func (qq *FilesListPartial) fileListPropertyMenuItem(
	label *wx.Text,
	propertyID int64,
	isChecked bool,
	hxHeaders template.JS,
) *wx.MenuItem {
	data := qq.actions.Browse.UpdateFileListPreferencesCmd.Data()
	data.PropertyID = propertyID
	return &wx.MenuItem{
		Label:         label,
		CheckboxName:  "FileListProperty",
		CheckboxValue: strconv.FormatInt(propertyID, 10),
		IsChecked:     isChecked,
		HTMXAttrs:     qq.fileListPreferencesMenuItemAttrs(data, hxHeaders),
	}
}

func (qq *FilesListPartial) fileListTagGroupMenuItem(
	label *wx.Text,
	tagGroupID int64,
	isChecked bool,
	hxHeaders template.JS,
) *wx.MenuItem {
	data := qq.actions.Browse.UpdateFileListPreferencesCmd.Data()
	data.TagGroupID = tagGroupID
	return &wx.MenuItem{
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
) wx.HTMXAttrs {
	return wx.HTMXAttrs{
		HxPost:    qq.actions.Browse.UpdateFileListPreferencesCmd.Endpoint(),
		HxVals:    util.JSON(data),
		HxHeaders: hxHeaders,
		HxTarget:  "#innerContent",
		HxSwap:    "innerHTML",
	}
}
