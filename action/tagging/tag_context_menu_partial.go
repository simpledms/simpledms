package tagging

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
)

// TODO move to partial package?
type TagContextMenuPartial struct {
	// TODO add infra? not sure, partials probably shouldn't
	//		do db queries

	actions *Actions
}

func NewTagContextMenuPartial(actions *Actions) *TagContextMenuPartial {
	return &TagContextMenuPartial{
		actions: actions,
	}
}

// TODO should also work without file
func (qq *TagContextMenuPartial) Widget(ctx ctxx.Context, fileID string, tagx *enttenant.Tag) *wx.Menu {
	deleteLink := &wx.MenuItem{
		LeadingIcon: "delete",
		Label:       wx.T("Delete"),
		HTMXAttrs: wx.HTMXAttrs{
			HxPost: qq.actions.DeleteTagCmd.Endpoint(),
			HxVals: util.JSON(qq.actions.DeleteTagCmd.Data(tagx.ID)),
			// HxTarget:  "#" + qq.actions.AssignedTags.EditListItem.listItemID(fileID, tagx.ID),
			HxConfirm: wx.T("Are you sure? This action will delete the tag entirely and not just unassign it from the current file!").String(ctx),
		},
	}

	// TODO handle Deletion for groups...

	menuItems := []*wx.MenuItem{
		{
			LeadingIcon: "edit",
			Label:       wx.T("Edit"),
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
		assignSubTagsLink := &wx.MenuItem{
			Label:       wx.T("Assign tags"), // TODO or Sub-tags? sounds bad in german
			LeadingIcon: "label",
			HTMXAttrs: wx.HTMXAttrs{
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
				&wx.MenuItem{
					LeadingIcon: "move_item",
					Label:       wx.T("Move to group"),
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
		&wx.MenuItem{
			IsDivider: true,
		},
		deleteLink,
	)
	return &wx.Menu{
		Items: menuItems,
	}
}
