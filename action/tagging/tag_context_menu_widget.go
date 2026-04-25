package tagging

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
)

// TODO move to partial package?
type TagContextMenuWidget struct {
	// TODO add infra? not sure, partials probably shouldn't
	//		do db queries

	actions *Actions
}

func NewTagContextMenuWidget(actions *Actions) *TagContextMenuWidget {
	return &TagContextMenuWidget{
		actions: actions,
	}
}

// TODO should also work without file
func (qq *TagContextMenuWidget) Widget(ctx ctxx.Context, fileID string, tagx *enttenant.Tag) *widget.Menu {
	deleteLink := &widget.MenuItem{
		LeadingIcon: "delete",
		Label:       widget.T("Delete"),
		HTMXAttrs: widget.HTMXAttrs{
			HxPost: qq.actions.DeleteTagCmd.Endpoint(),
			HxVals: util.JSON(qq.actions.DeleteTagCmd.Data(tagx.ID)),
			// HxTarget:  "#" + qq.actions.AssignedTags.EditListItem.listItemID(fileID, tagx.ID),
			HxConfirm: widget.T("Are you sure? This action will delete the tag entirely and not just unassign it from the current file!").String(ctx),
		},
	}

	// TODO handle Deletion for groups...

	menuItems := []*widget.MenuItem{
		{
			LeadingIcon: "edit",
			Label:       widget.T("Edit"),
			HTMXAttrs: qq.actions.EditTagCmd.ModalLinkAttrs(
				qq.actions.EditTagCmd.Data(tagx.ID, tagx.Name),
				"#"+qq.actions.AssignedTags.EditListItem.listItemID(fileID, tagx.ID),
			).SetHxHeaders(
				autil.QueryHeader(
					qq.actions.AssignedTags.EditListItem.Endpoint(),
					qq.actions.AssignedTags.EditListItem.Data(fileID, tagx.ID),
				),
			),
		},
	}

	if tagx.Type == tagtype.Simple {
		/* TODO
		menuItems = append(
			menuItems,
			&wx.MenuItem{
				Label: wx.T("Convert to composed tag"),
			},
		)

		*/
	}
	if tagx.Type == tagtype.Super {
		assignSubTagsLink := &widget.MenuItem{
			Label:       widget.T("Assign tags"), // TODO or Sub-tags? sounds bad in german
			LeadingIcon: "label",
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:        qq.actions.SubTags.Edit.EndpointWithParams(actionx.ResponseWrapperDialog, ""),
				HxVals:        util.JSON(qq.actions.SubTags.Edit.Data(tagx.ID, false)),
				LoadInPopover: true,
			},
		}
		menuItems = append(
			menuItems,
			assignSubTagsLink,
		)
		/* TODO
		isDisabled := len(tagx.Edges.SubTags) > 0
		/* FIXME as tooltip
		supportingText := ""
		if isDisabled {
			supportingText = "Only available without sub-tags"
		}
		/
		menuItems = append(
			menuItems,
			&wx.MenuItem{
				Label: wx.T("Convert to base tag"),
				// SupportingText: supportingText, // TODO as tooltip
				IsDisabled: isDisabled,
			},
		)
		*/
	}

	if tagx.Type != tagtype.Group {
		groupCount := ctx.SpaceCtx().Space.QueryTags().Where(tag.TypeEQ(tagtype.Group)).CountX(ctx)
		if groupCount > 0 {
			menuItems = append(
				menuItems,
				&widget.MenuItem{
					LeadingIcon: "move_item",
					Label:       widget.T("Move to group"),
					HTMXAttrs: qq.actions.MoveTagToGroupCmd.ModalLinkAttrs(
						qq.actions.MoveTagToGroupCmd.Data(tagx.ID, 0),
						"#"+qq.actions.AssignedTags.Edit.hxTargetID(),
					).SetHxHeaders(
						autil.QueryHeader(
							// always ListView (edit mode) because context menu is just available from there...
							qq.actions.AssignedTags.Edit.Endpoint(),
							qq.actions.AssignedTags.Edit.Data(fileID, tagx.ID),
						),
					),
				},
			)
		}
	}

	menuItems = append(
		menuItems,
		&widget.MenuItem{
			IsDivider: true,
		},
		deleteLink,
	)
	return &widget.Menu{
		Items: menuItems,
	}
}
