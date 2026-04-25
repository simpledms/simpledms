package server

import (
	"log"

	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type SimpleDMSMainMenuPlugin struct{}

func NewSimpleDMSMainMenuPlugin() *SimpleDMSMainMenuPlugin {
	return &SimpleDMSMainMenuPlugin{}
}

func (qq *SimpleDMSMainMenuPlugin) Name() string {
	return "simpledms-main-menu"
}

func (qq *SimpleDMSMainMenuPlugin) ExtendMenuItems(
	ctx ctxx.Context,
	items []*widget.MenuItem,
) []*widget.MenuItem {
	if ctx.IsMainCtx() {
		spacesByTenant, err := ctx.MainCtx().ReadOnlyAccountSpacesByTenant()
		if err != nil {
			log.Println(err)
		} else {
			for tenantx, spaces := range spacesByTenant {
				items = append(items, &widget.MenuItem{
					LeadingIcon: "hub",
					Label:       widget.Tuf("%s «%s»", widget.T("Spaces").String(ctx), tenantx.Name),
					HTMXAttrs: widget.HTMXAttrs{
						HxGet: route.SpacesRoot(tenantx.PublicID.String()),
					},
				})

				for _, spacex := range spaces {
					leadingIcon := "check_box_outline_blank"
					isCurrent := ctx.IsSpaceCtx() && ctx.SpaceCtx().SpaceID == spacex.PublicID.String()
					if isCurrent {
						leadingIcon = "check_box"
					}

					items = append(items, &widget.MenuItem{
						LeadingIcon: leadingIcon,
						Label:       widget.Tu(spacex.Name),
						HTMXAttrs: widget.HTMXAttrs{
							HxGet: route.BrowseRoot(tenantx.PublicID.String(), spacex.PublicID.String()),
						},
					})
				}

				items = append(items, &widget.MenuItem{IsDivider: true})
			}
		}
	}

	if ctx.IsSpaceCtx() {
		items = append(items, []*widget.MenuItem{
			{
				LeadingIcon: "category",
				Label:       widget.T("Document types"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route.ManageDocumentTypes(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				LeadingIcon: "label",
				Label:       widget.T("Tags"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route.ManageTags(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				LeadingIcon: "tune",
				Label:       widget.T("Fields"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route.ManageProperties(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				LeadingIcon: "person",
				Label:       widget.T("Users"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route.ManageUsersOfSpace(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				LeadingIcon: "delete",
				Label:       widget.T("Trash"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route.TrashRoot(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				},
			},
			{
				IsDivider: true,
			},
		}...)
	}

	return items
}
