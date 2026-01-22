package spaces

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type SpaceContextMenuPartial struct {
	actions *Actions
}

func NewSpaceContextMenuPartial(actions *Actions) *SpaceContextMenuPartial {
	return &SpaceContextMenuPartial{
		actions: actions,
	}
}

func (qq *SpaceContextMenuPartial) Widget(ctx ctxx.Context, spacem *model.Space) *wx.Menu {
	renameItem := &wx.MenuItem{
		LeadingIcon: "edit",
		Label:       wx.T("Edit"),
		HTMXAttrs: qq.actions.EditSpaceCmd.ModalLinkAttrs(
			qq.actions.EditSpaceCmd.Data(
				spacem.Data.PublicID.String(),
				spacem.Data.Name,
				spacem.Data.Description,
			),
			"",
		),
	}

	deleteItem := &wx.MenuItem{
		LeadingIcon: "delete",
		Label:       wx.T("Delete"),
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:    qq.actions.DeleteSpaceCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.DeleteSpaceCmd.Data(spacem.Data.PublicID.String())),
			HxConfirm: wx.T("Are you sure?").String(ctx),
		},
	}

	menuItems := []*wx.MenuItem{
		renameItem,
		{
			IsDivider: true,
		},
		deleteItem,
		{
			IsDivider: true,
		},
		// similar code in SpaceContextMenuPartial and MainMenu
		{
			LeadingIcon: "category", // TODO category or description?
			Label:       wx.T("Document types"),
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route2.ManageDocumentTypes(ctx.TenantCtx().TenantID, spacem.Data.PublicID.String()),
			},
		},
		{
			LeadingIcon: "label",
			Label:       wx.T("Tags"),
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route2.ManageTags(ctx.TenantCtx().TenantID, spacem.Data.PublicID.String()),
			},
		},
		{
			LeadingIcon: "tune", // tune or assignment
			Label:       wx.T("Fields"),
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route2.ManageProperties(ctx.TenantCtx().TenantID, spacem.Data.PublicID.String()),
			},
		},
		{
			LeadingIcon: "person",
			Label:       wx.T("Users"),
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route2.ManageUsersOfSpace(ctx.TenantCtx().TenantID, spacem.Data.PublicID.String()),
			},
		},
	}

	return &wx.Menu{
		Items: menuItems,
	}
}
