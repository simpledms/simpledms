package managetags

import (
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/ctxx"
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
func (qq *TagContextMenuWidget) Widget(ctx ctxx.Context, tagx *enttenant.Tag) *widget.Menu {
	deleteLink := &widget.MenuItem{
		LeadingIcon: "delete",
		Label:       widget.T("Delete"),
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:    qq.actions.Tagging.DeleteTagCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.Tagging.DeleteTagCmd.Data(tagx.ID)),
			HxConfirm: widget.T("Are you sure? This action will delete the tag and unassign it from all files!").String(ctx),
		},
	}

	// TODO handle Deletion for groups...

	menuItems := []*widget.MenuItem{
		{
			LeadingIcon: "edit",
			Label:       widget.T("Edit"),
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
		assignSubTagsLink := &widget.MenuItem{
			Label:       widget.T("Assign tags"), // TODO or Sub-tags? sounds bad in german
			LeadingIcon: "label",
			HTMXAttrs: widget.HTMXAttrs{
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
				&widget.MenuItem{
					LeadingIcon: "move_item",
					Label:       widget.T("Move to group"),
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
		&widget.MenuItem{
			IsDivider: true,
		},
		deleteLink, // TODO also if group?
	)

	return &widget.Menu{
		Items: menuItems,
	}
}
