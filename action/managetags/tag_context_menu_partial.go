package managetags

import (
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
func (qq *TagContextMenuPartial) Widget(ctx ctxx.Context, tagx *enttenant.Tag) *wx.Menu {
	deleteLink := &wx.MenuItem{
		LeadingIcon: "delete",
		Label:       wx.T("Delete"),
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:    qq.actions.Tagging.DeleteTagCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.Tagging.DeleteTagCmd.Data(tagx.ID)),
			HxConfirm: wx.T("Are you sure? This action will delete the tag and unassign it from all files!").String(ctx),
		},
	}

	// TODO handle Deletion for groups...

	menuItems := []*wx.MenuItem{
		{
			LeadingIcon: "edit",
			Label:       wx.T("Edit"),
			HTMXAttrs: qq.actions.Tagging.EditTagCmd.ModalLinkAttrs(
				qq.actions.Tagging.EditTagCmd.Data(tagx.ID, tagx.Name),
				"",
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
				HxPost:        qq.actions.Tagging.SubTags.Edit.EndpointWithParams(actionx.ResponseWrapperDialog, ""),
				HxVals:        util.JSON(qq.actions.Tagging.SubTags.Edit.Data(tagx.ID, false)),
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
					HTMXAttrs: qq.actions.Tagging.MoveTagToGroupCmd.ModalLinkAttrs(
						qq.actions.Tagging.MoveTagToGroupCmd.Data(tagx.ID, 0),
						"",
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
		deleteLink, // TODO also if group?
	)

	return &wx.Menu{
		Items: menuItems,
	}
}
