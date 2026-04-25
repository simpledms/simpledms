package spaces

import (
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	spacemodel "github.com/simpledms/simpledms/model/tenant/space"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
)

type SpaceContextMenuWidget struct {
	actions *Actions
}

func NewSpaceContextMenuWidget(actions *Actions) *SpaceContextMenuWidget {
	return &SpaceContextMenuWidget{
		actions: actions,
	}
}

func (qq *SpaceContextMenuWidget) Widget(ctx ctxx.Context, spacem *spacemodel.Space) *widget.Menu {
	renameItem := &widget.MenuItem{
		LeadingIcon: "edit",
		Label:       widget.T("Edit"),
		HTMXAttrs: qq.actions.EditSpaceCmd.ModalLinkAttrs(
			qq.actions.EditSpaceCmd.Data(
				spacem.Data.PublicID.String(),
				spacem.Data.Name,
				spacem.Data.Description,
			),
			"",
		),
	}

	deleteItem := &widget.MenuItem{
		LeadingIcon: "delete",
		Label:       widget.T("Delete"),
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:    qq.actions.DeleteSpaceCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.DeleteSpaceCmd.Data(spacem.Data.PublicID.String())),
			HxConfirm: widget.T("Are you sure?").String(ctx),
		},
	}

	menuItems := []*widget.MenuItem{
		renameItem,
		{
			IsDivider: true,
		},
		deleteItem,
		{
			IsDivider: true,
		},
		// similar code in SpaceContextMenuWidget and MainMenu
		{
			LeadingIcon: "category", // TODO category or description?
			Label:       widget.T("Document types"),
			HTMXAttrs: widget.HTMXAttrs{
				HxGet: route2.ManageDocumentTypes(ctx.TenantCtx().TenantID, spacem.Data.PublicID.String()),
			},
		},
		{
			LeadingIcon: "label",
			Label:       widget.T("Tags"),
			HTMXAttrs: widget.HTMXAttrs{
				HxGet: route2.ManageTags(ctx.TenantCtx().TenantID, spacem.Data.PublicID.String()),
			},
		},
		{
			LeadingIcon: "tune", // tune or assignment
			Label:       widget.T("Fields"),
			HTMXAttrs: widget.HTMXAttrs{
				HxGet: route2.ManageProperties(ctx.TenantCtx().TenantID, spacem.Data.PublicID.String()),
			},
		},
		{
			LeadingIcon: "person",
			Label:       widget.T("Users"),
			HTMXAttrs: widget.HTMXAttrs{
				HxGet: route2.ManageUsersOfSpace(ctx.TenantCtx().TenantID, spacem.Data.PublicID.String()),
			},
		},
	}

	return &widget.Menu{
		Items: menuItems,
	}
}
