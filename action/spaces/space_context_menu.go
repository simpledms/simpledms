package spaces

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/route"
)

type SpaceContextMenu struct {
	actions *Actions
}

func NewSpaceContextMenu(actions *Actions) *SpaceContextMenu {
	return &SpaceContextMenu{
		actions: actions,
	}
}

func (qq *SpaceContextMenu) Widget(ctx ctxx.Context, spacem *model.Space) *wx.Menu {
	renameItem := &wx.MenuItem{
		LeadingIcon: "edit",
		Label:       wx.T("Edit"),
		HTMXAttrs: qq.actions.EditSpace.ModalLinkAttrs(
			qq.actions.EditSpace.Data(
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
			HxPost:    qq.actions.DeleteSpace.Endpoint(),
			HxVals:    util.JSON(qq.actions.DeleteSpace.Data(spacem.Data.PublicID.String())),
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
		// similar code in SpaceContextMenu and MainMenu
		{
			LeadingIcon: "category", // TODO category or description?
			Label:       wx.T("Document types"),
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route.ManageDocumentTypes(ctx.TenantCtx().TenantID, spacem.Data.PublicID.String()),
			},
		},
		{
			LeadingIcon: "label",
			Label:       wx.T("Tags"),
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route.ManageTags(ctx.TenantCtx().TenantID, spacem.Data.PublicID.String()),
			},
		},
		{
			LeadingIcon: "tune", // tune or assignment
			Label:       wx.T("Fields"),
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route.ManageProperties(ctx.TenantCtx().TenantID, spacem.Data.PublicID.String()),
			},
		},
		{
			LeadingIcon: "person",
			Label:       wx.T("Users"),
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route.ManageUsersOfSpace(ctx.TenantCtx().TenantID, spacem.Data.PublicID.String()),
			},
		},
	}

	return &wx.Menu{
		Items: menuItems,
	}
}
