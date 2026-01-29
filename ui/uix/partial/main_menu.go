package partial

import (
	"fmt"

	"github.com/simpledms/simpledms/ctxx"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
)

func NewMainMenu(ctx ctxx.Context) *wx.IconButton {
	var items []*wx.MenuItem

	items = append(items, []*wx.MenuItem{
		{
			LeadingIcon: "dashboard",
			Label:       wx.T("Dashboard"),
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route2.Dashboard(),
			},
		},
		{
			IsDivider: true,
		}}...,
	)

	for tenantx, spaces := range ctx.MainCtx().ReadOnlyAccountSpacesByTenant() {
		items = append(items, &wx.MenuItem{
			LeadingIcon: "hub",
			// TODO or `all spaces` or `manage spaces`? `|` or «»
			Label: wx.Tuf("%s «%s»", wx.T("Spaces").String(ctx), tenantx.Name),
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route2.SpacesRoot(tenantx.PublicID.String()),
			},
		})
		// TODO add Label with Tenant name
		for _, spacex := range spaces {
			// trailingIcon := ""
			leadingIcon := "check_box_outline_blank"
			isCurrent := ctx.IsSpaceCtx() && ctx.SpaceCtx().SpaceID == spacex.PublicID.String()
			if isCurrent {
				// trailingIcon = "check"
				leadingIcon = "check_box"
			}
			items = append(items, &wx.MenuItem{
				LeadingIcon: leadingIcon,
				// TrailingIcon: trailingIcon,
				// TODO tenant name as label or supporting text or tooltip?
				Label: wx.Tu(fmt.Sprintf("%s", spacex.Name)),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route2.BrowseRoot(tenantx.PublicID.String(), spacex.PublicID.String()),
				},
			})
		}
		items = append(items, &wx.MenuItem{
			IsDivider: true,
		})
	}

	if ctx.IsSpaceCtx() {
		// near duplicate in SpaceContextMenu
		// TODO implement submenu or add label?
		items = append(items, []*wx.MenuItem{
			{
				// better from usability point of view if after tags and properties because they must
				// be configured first
				LeadingIcon: "category", // TODO category or description?
				Label:       wx.T("Document types"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route2.ManageDocumentTypes(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				LeadingIcon: "label",
				Label:       wx.T("Tags"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route2.ManageTags(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				LeadingIcon: "tune", // tune or assignment
				Label:       wx.T("Fields"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route2.ManageProperties(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				LeadingIcon: "person",
				Label:       wx.T("Users"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route2.ManageUsersOfSpace(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				LeadingIcon: "delete",
				Label:       wx.T("Trash"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route2.TrashRoot(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				IsDivider: true,
			},
		}...)
	}

	/*
		if ctx.IsTenantCtx() && modelmain.NewTenant(ctx.TenantCtx().Tenant).IsOwner(modelmain.NewAccount(ctx.TenantCtx().Account)) {
			// TODO implement submenu or add label?
			items = append(items, []*wx.MenuItem{
				{
					LeadingIcon: "domain",
					Label:       wx.T("Tenant"),
					HTMXAttrs:   wx.HTMXAttrs{
						// HxGet: route.ManageDocumentTypes(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
					},
				},
				{
					LeadingIcon: "manage_accounts", // admin_panel_settings, manage_accounts, badge
					Label:       wx.T("Accounts"),
					HTMXAttrs:   wx.HTMXAttrs{
						// HxGet: route.ManageDocumentTypes(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
					},
				},
				{
					IsDivider: true,
				},
			}...)
		}
	*/

	items = append(items,
		&wx.MenuItem{
			LeadingIcon: "logout",
			Label:       wx.T("Sign out"),
			HTMXAttrs: wx.HTMXAttrs{
				HxPost: route2.SignOutCmd(),
			},
		},
	)

	if !ctx.VisitorCtx().CommercialLicenseEnabled {
		items = append(items,
			&wx.MenuItem{
				IsDivider: true,
			},
			&wx.MenuItem{
				LeadingIcon: "info",
				Label:       wx.T("About SimpleDMS"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route2.AboutPage(),
				},
			})
	}

	return &wx.IconButton{
		Icon:    "menu",
		Tooltip: wx.T("Open main menu"),
		Children: &wx.Menu{
			Position: wx.PositionRight,
			Items:    items,
		},
	}
}
