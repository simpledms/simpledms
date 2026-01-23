package tagging

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type EditAssignedTagsPartialData struct {
	FileID      string
	ParentTagID int64 // to load group children
}

type EditAssignedTagsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewEditAssignedTagsPartial(infra *common.Infra, actions *Actions) *EditAssignedTagsPartial {
	config := actionx.NewConfig(
		actions.Route("edit-assigned-tags-partial"),
		true, // TODO true or false?
	)
	return &EditAssignedTagsPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *EditAssignedTagsPartial) Data(fileID string, parentTagID int64) *EditAssignedTagsPartialData {
	return &EditAssignedTagsPartialData{
		FileID:      fileID,
		ParentTagID: parentTagID,
	}
}

func (qq *EditAssignedTagsPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	// TODO do prep if necessary... (filter selects, etc.)
	// TODO set default value, for example for current

	data, err := autil.FormData[EditAssignedTagsPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	wrapper := req.URL.Query().Get("wrapper")
	// hxTarget := req.URL.Query().Get("hx-target")

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Form(ctx, data, actionx.ResponseWrapper(wrapper)),
	)
}

func (qq *EditAssignedTagsPartial) Form(
	ctx ctxx.Context,
	data *EditAssignedTagsPartialData,
	wrapper actionx.ResponseWrapper,
) renderable.Renderable {
	// TODO readd header

	return qq.ListView(ctx, data)
}

func (qq *EditAssignedTagsPartial) bottomAppBar(
	data *EditAssignedTagsPartialData,
) *wx.BottomAppBar {
	hxTarget := "#" + qq.hxTargetID()
	return &wx.BottomAppBar{
		Actions: []wx.IWidget{
			&wx.IconButton{
				Icon: "list_alt",
				HTMXAttrs: wx.HTMXAttrs{
					HxPost: qq.actions.AssignedTags.List.EndpointWithParams(actionx.ResponseWrapperNone, hxTarget),
					HxVals: util.JSON(qq.actions.AssignedTags.List.Data(data.FileID)),
					// TODO is this a good idea? or try to select closest tab?
					HxTarget: hxTarget,
					HxSwap:   "outerHTML",
				},
			},
		},
	}
}

// TODO rename to Widget?
func (qq *EditAssignedTagsPartial) ListView(
	ctx ctxx.Context,
	data *EditAssignedTagsPartialData,
) *wx.ScrollableContent {
	// duplicate in qq.Form
	isLoadingPartial := data.ParentTagID > 0

	allTagsQuery := ctx.SpaceCtx().Space.QueryTags().
		Order(tag.ByName()).
		WithChildren(func(query *enttenant.TagQuery) {
			query.Order(tag.ByName())
		}).
		WithSubTags(func(query *enttenant.TagQuery) {
			query.Order(tag.ByName())
		})
	if isLoadingPartial {
		allTagsQuery = allTagsQuery.Where(tag.HasGroupWith(tag.ID(data.ParentTagID)))
	} else {
		// load just top layer into collection, but eagerly load children
		// for more efficient processing
		allTagsQuery = allTagsQuery.Where(tag.Not(tag.HasGroup()))
	}
	allTags := allTagsQuery.AllX(ctx)

	// TODO is there a better way than working with 3 groups?
	var allListItems []wx.IWidget
	var tagListItems []wx.IWidget
	var groupListItems []wx.IWidget

	if !isLoadingPartial {
		// TODO or as FAB? would be more consistent with rest of application
		//		but less consistent with adding tags to group
		allListItems = append(allListItems,
			// TODO segment into two list items?
			&wx.ListItem{
				HTMXAttrs: qq.actions.AssignedTags.CreateAndAssignTagCmd.ModalLinkAttrs(
					qq.actions.AssignedTags.CreateAndAssignTagCmd.Data(data.FileID, 0),
					"#"+qq.hxTargetID(),
				),
				Leading:  wx.NewIcon("new_label"),
				Headline: wx.T("Create new tag or group"),
				Type:     wx.ListItemTypeHelper,
			},
		)
	}

	// TODO factor into function?
	isCheckedFn := qq.actions.AssignedTags.EditListItem.IsCheckedFn(ctx, data.FileID)
	for _, tagx := range allTags {
		listItem := qq.actions.AssignedTags.EditListItem.listItem(ctx, data.FileID, tagx, isCheckedFn)
		if tagx.Type == tagtype.Group {
			groupListItems = append(groupListItems, listItem)
		} else {
			tagListItems = append(tagListItems, listItem)
		}
	}

	allListItems = append(allListItems, groupListItems...)
	allListItems = append(allListItems, tagListItems...)

	if isLoadingPartial {
		return &wx.ScrollableContent{
			Children: allListItems,
		}
	}

	if len(allListItems) == 1 { // check if only create list item is added
		return &wx.ScrollableContent{
			Widget: wx.Widget[wx.ScrollableContent]{
				ID: qq.hxTargetID(), // important because used as target
			},
			MarginY: true,
			Children: &wx.EmptyState{
				// Icon:     wx.NewIcon("label"),
				Headline: wx.T("No tags available yet."),
				// Description: NewText("There are no directories or files available yet, you can create"),
				Actions: []wx.IWidget{
					qq.actions.AssignedTags.CreateAndAssignTagCmd.ModalLink(
						qq.actions.AssignedTags.CreateAndAssignTagCmd.Data(data.FileID, 0),
						[]wx.IWidget{
							&wx.Button{
								Icon:  wx.NewIcon("folder_special"),
								Label: wx.T("Create new group"),
							},
						},
						"#"+qq.hxTargetID(),
					),
					qq.actions.AssignedTags.CreateAndAssignTagCmd.ModalLink(
						qq.actions.AssignedTags.CreateAndAssignTagCmd.Data(data.FileID, 0),
						[]wx.IWidget{
							&wx.Button{
								Icon:  wx.NewIcon("new_label"),
								Label: wx.T("Create new tag"),
							},
						},
						"#"+qq.hxTargetID(),
					),
				},
			},
		}
	}

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.hxTargetID(),
		},
		Children: &wx.List{
			Children: allListItems,
		},
		BottomAppBar: qq.bottomAppBar(data),
	}
}

func (qq *EditAssignedTagsPartial) hxTargetID() string {
	return "tagAssignmentList"
}
