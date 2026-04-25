package partial

import (
	"fmt"
	"log"

	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	account2 "github.com/marcobeierer/go-core/model/account"
	"github.com/marcobeierer/go-core/ui/uix/route"
	"github.com/marcobeierer/go-core/ui/widget"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
)

func NewMainMenu(ctx ctxx.Context, infra *common.Infra) *widget.IconButton {
	var items []*widget.MenuItem

	if ctx.IsMainCtx() {
		accountm := account2.NewAccount(ctx.MainCtx().Account)
		passkeyPolicy, err := accountm.PasskeyPolicy(ctx)
		if err != nil {
			log.Println(err)
			passkeyPolicy = account2.NewPasskeyPolicy(false, false, false)
		}
		isTenantPasskeyEnrollmentRequired := passkeyPolicy.IsTenantPasskeyEnrollmentRequired()
		if isTenantPasskeyEnrollmentRequired {
			return &widget.IconButton{
				Icon:    "menu",
				Tooltip: widget.T("Open main menu"),
				Children: &widget.Menu{
					Position: widget.PositionRight,
					Items: []*widget.MenuItem{
						{
							LeadingIcon: "dashboard",
							Label:       widget.T("Dashboard"),
							HTMXAttrs: widget.HTMXAttrs{
								HxGet: route.Dashboard(),
							},
						},
						{
							IsDivider: true,
						},
						{
							LeadingIcon: "logout",
							Label:       widget.T("Sign out"),
							HTMXAttrs: widget.HTMXAttrs{
								HxPost: route.SignOutCmd(),
							},
						},
					},
				},
			}
		}

		items = append(items, []*widget.MenuItem{
			{
				LeadingIcon: "dashboard",
				Label:       widget.T("Dashboard"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route.Dashboard(),
				},
			},
			{
				IsDivider: true,
			}}...,
		)

		spacesByTenant, err := ctx.MainCtx().ReadOnlyAccountSpacesByTenant()
		if err != nil {
			// TODO returning an error would probably be better...
			log.Println(err)
		} else {
			for tenantx, spaces := range spacesByTenant {
				items = append(items, &widget.MenuItem{
					LeadingIcon: "hub",
					// TODO or `all spaces` or `manage spaces`? `|` or «»
					Label: widget.Tuf("%s «%s»", widget.T("Spaces").String(ctx), tenantx.Name),
					HTMXAttrs: widget.HTMXAttrs{
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
					items = append(items, &widget.MenuItem{
						LeadingIcon: leadingIcon,
						// TrailingIcon: trailingIcon,
						// TODO tenant name as label or supporting text or tooltip?
						Label: widget.Tu(fmt.Sprintf("%s", spacex.Name)),
						HTMXAttrs: widget.HTMXAttrs{
							HxGet: route2.BrowseRoot(tenantx.PublicID.String(), spacex.PublicID.String()),
						},
					})
				}
				items = append(items, &widget.MenuItem{
					IsDivider: true,
				})
			}
		}
	}

	if ctx.IsSpaceCtx() {
		// near duplicate in SpaceContextMenu
		// TODO implement submenu or add label?
		items = append(items, []*widget.MenuItem{
			{
				// better from usability point of view if after tags and properties because they must
				// be configured first
				LeadingIcon: "category", // TODO category or description?
				Label:       widget.T("Document types"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route2.ManageDocumentTypes(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				LeadingIcon: "label",
				Label:       widget.T("Tags"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route2.ManageTags(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				LeadingIcon: "tune", // tune or assignment
				Label:       widget.T("Fields"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route2.ManageProperties(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				LeadingIcon: "person",
				Label:       widget.T("Users"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route2.ManageUsersOfSpace(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				LeadingIcon: "delete",
				Label:       widget.T("Trash"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route2.TrashRoot(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				IsDivider: true,
			},
		}...)
	}

	/*
		if ctx.IsTenantCtx() && tenantmainmodel.NewTenant(ctx.TenantCtx().Tenant).IsOwner(accountmainmodel.NewAccount(ctx.TenantCtx().Account)) {
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

	if ctx.IsMainCtx() {
		items = append(items,
			&widget.MenuItem{
				LeadingIcon: "logout",
				Label:       widget.T("Sign out"),
				HTMXAttrs: widget.HTMXAttrs{
					HxPost: route.SignOutCmd(),
				},
			},
		)
	}

	items = infra.PluginRegistry().ExtendMenuItems(ctx, items)

	if !ctx.VisitorCtx().CommercialLicenseEnabled {
		// 0 on login page
		if len(items) > 0 {
			items = append(items, &widget.MenuItem{
				IsDivider: true,
			})
		}
		items = append(items,
			&widget.MenuItem{
				LeadingIcon: "info",
				Label:       widget.T("About SimpleDMS"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route.AboutPage(),
				},
			})
	}

	return &widget.IconButton{
		Icon:    "menu",
		Tooltip: widget.T("Open main menu"),
		Children: &widget.Menu{
			Position: widget.PositionRight,
			Items:    items,
		},
	}
}
