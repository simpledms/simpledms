package tagging

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	actionx2 "github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
)

type EditAssignedTagsPartialData struct {
	FileID      string
	ParentTagID int64 // to load group children
}

type EditAssignedTagsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx2.Config
}

func NewEditAssignedTagsPartial(infra *common.Infra, actions *Actions) *EditAssignedTagsPartial {
	config := actionx2.NewConfig(
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

func (qq *EditAssignedTagsPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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
		qq.Form(ctx, data, actionx2.ResponseWrapper(wrapper)),
	)
}

func (qq *EditAssignedTagsPartial) Form(
	ctx ctxx.Context,
	data *EditAssignedTagsPartialData,
	wrapper actionx2.ResponseWrapper,
) renderable.Renderable {
	// TODO readd header

	return qq.ListView(ctx, data)
}

func (qq *EditAssignedTagsPartial) bottomAppBar(
	data *EditAssignedTagsPartialData,
) *widget.BottomAppBar {
	hxTarget := "#" + qq.hxTargetID()
	return &widget.BottomAppBar{
		Actions: []widget.IWidget{
			&widget.IconButton{
				Icon:    "list_alt",
				Tooltip: widget.T("Show assigned tags"),
				HTMXAttrs: widget.HTMXAttrs{
					HxPost: qq.actions.AssignedTags.List.EndpointWithParams(actionx2.ResponseWrapperNone, hxTarget),
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
) *widget.ScrollableContent {
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
	var allListItems []widget.IWidget
	var tagListItems []widget.IWidget
	var groupListItems []widget.IWidget

	if !isLoadingPartial {
		// TODO or as FAB? would be more consistent with rest of application
		//		but less consistent with adding tags to group
		allListItems = append(allListItems,
			// TODO segment into two list items?
			&widget.ListItem{
				HTMXAttrs: qq.actions.AssignedTags.CreateAndAssignTagCmd.ModalLinkAttrs(
					qq.actions.AssignedTags.CreateAndAssignTagCmd.Data(data.FileID, 0),
					"#"+qq.hxTargetID(),
				),
				Leading:  widget.NewIcon("new_label"),
				Headline: widget.T("Create new tag or group"),
				Type:     widget.ListItemTypeHelper,
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
		return &widget.ScrollableContent{
			Children: allListItems,
		}
	}

	if len(allListItems) == 1 { // check if only create list item is added
		return &widget.ScrollableContent{
			Widget: widget.Widget[widget.ScrollableContent]{
				ID: qq.hxTargetID(), // important because used as target
			},
			MarginY: true,
			Children: &widget.EmptyState{
				// Icon:     wx.NewIcon("label"),
				Headline: widget.T("No tags available yet."),
				// Description: NewText("There are no directories or files available yet, you can create"),
				Actions: []widget.IWidget{
					qq.actions.AssignedTags.CreateAndAssignTagCmd.ModalLink(
						qq.actions.AssignedTags.CreateAndAssignTagCmd.Data(data.FileID, 0),
						[]widget.IWidget{
							&widget.Button{
								Icon:  widget.NewIcon("folder_special"),
								Label: widget.T("Create new group"),
							},
						},
						"#"+qq.hxTargetID(),
					),
					qq.actions.AssignedTags.CreateAndAssignTagCmd.ModalLink(
						qq.actions.AssignedTags.CreateAndAssignTagCmd.Data(data.FileID, 0),
						[]widget.IWidget{
							&widget.Button{
								Icon:  widget.NewIcon("new_label"),
								Label: widget.T("Create new tag"),
							},
						},
						"#"+qq.hxTargetID(),
					),
				},
			},
		}
	}

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: qq.hxTargetID(),
		},
		Children: &widget.List{
			Children: allListItems,
		},
		BottomAppBar: qq.bottomAppBar(data),
	}
}

func (qq *EditAssignedTagsPartial) hxTargetID() string {
	return "tagAssignmentList"
}
