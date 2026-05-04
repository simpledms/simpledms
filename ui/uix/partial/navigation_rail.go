package partial

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
)

// fab must be injected because it differs on each page...
func NewNavigationRail(ctx ctxx.Context, infra *common.Infra, active string, fabs []*wx.FloatingActionButton) *wx.NavigationRail {
	var destinations []*wx.NavigationDestination
	isMetadataRail := active == "document-types" || active == "tags" || active == "fields"
	/*
		{
			Label:    "Files", // TODO or Finder?
			Icon:     "folder_open",
			IsActive: active == "find",
			Href:     route.FindRoot(),
		},

	*/

	// check for IsMainCtx instead of IsVisitorCtx because IsVisitor
	// is true in all contexts
	if !ctx.IsMainCtx() {
		destinations = append(destinations, &wx.NavigationDestination{
			Label:    wx.T("Sign in [subject]").String(ctx),
			Icon:     "login",
			IsActive: active == "sign-in",
			Href:     "/",
		})
	}

	if ctx.IsMainCtx() && !ctx.IsTenantCtx() && !ctx.IsSpaceCtx() {
		destinations = append(destinations, &wx.NavigationDestination{
			Label:    wx.T("Dashboard").String(ctx),
			Icon:     "dashboard",
			IsActive: active == "dashboard",
			Href:     route2.Dashboard(),
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route2.Dashboard(),
			},
		})
		destinations = append(destinations, &wx.NavigationDestination{
			Label:    wx.T("Account").String(ctx),
			Icon:     "account_circle",
			IsActive: active == "account",
			Href:     route2.Account(),
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route2.Account(),
			},
		})

		if ctx.MainCtx().Account.Role == mainrole.Admin {
			destinations = append(destinations, &wx.NavigationDestination{
				Label:    wx.T("System").String(ctx),
				Icon:     "settings",
				IsActive: active == "system",
				Href:     route2.System(),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route2.System(),
				},
			})
			destinations = infra.PluginRegistry().ExtendNavigationDestinations(ctx, destinations)
			destinations = appendTenantsDestinationFromMenuItems(ctx, infra, active, destinations)
		}
	}

	if ctx.IsSpaceCtx() {
		if isMetadataRail {
			destinations = append(destinations, &wx.NavigationDestination{
				Label:    wx.T("Files").String(ctx),
				Icon:     "folder_open",
				IsActive: false,
				Href:     route2.BrowseRoot(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
			destinations = append(destinations, &wx.NavigationDestination{
				Label:    wx.T("Document types").String(ctx),
				Icon:     "category",
				IsActive: active == "document-types",
				Href:     route2.ManageDocumentTypes(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
			destinations = append(destinations, &wx.NavigationDestination{
				Label:    wx.T("Tags").String(ctx),
				Icon:     "label",
				IsActive: active == "tags",
				Href:     route2.ManageTags(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
			destinations = append(destinations, &wx.NavigationDestination{
				Label:    wx.T("Fields").String(ctx),
				Icon:     "tune",
				IsActive: active == "fields",
				Href:     route2.ManageProperties(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
		} else {
			destinations = append(destinations, &wx.NavigationDestination{
				Label:    wx.T("Files").String(ctx), // TODO Files or Browse?
				Icon:     "folder_open",
				IsActive: active == "browse",
				Href:     route2.BrowseRoot(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
			destinations = append(destinations, &wx.NavigationDestination{
				Label:    wx.T("Inbox").String(ctx),
				Icon:     "inbox",
				IsActive: active == "inbox",
				Href:     route2.InboxRoot(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
			destinations = append(destinations, &wx.NavigationDestination{
				Label:    wx.T("Metadata").String(ctx),
				Icon:     "database",
				IsActive: active == "document-types",
				Href:     route2.ManageDocumentTypes(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
		}
		/*
			destinations = append(destinations, &wx.NavigationDestination{
				// TODO or Bookmarks or Collections? Favorites may cover both
				Label:    "Favorites",
				Icon:     "favorite",
				IsActive: active == "favorites",
			})
			destinations = append(destinations, &wx.NavigationDestination{
				Label:    "Share",
				Icon:     "share",
				IsActive: active == "share",
				// Href:     route.InboxRoot(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
		*/
	}

	/*
		if ctx.IsTenantCtx() {
			destinations = append(destinations, &wx.NavigationDestination{
				// TODO move to secondary menu?
				Label:    "Spaces",
				Icon:     "hub",
				IsActive: active == "spaces",
				Href:     route.SpacesRoot(ctx.TenantCtx().TenantID),
			})
		}
	*/

	return &wx.NavigationRail{
		MenuBtn: NewMainMenu(ctx, infra),
		// must be after main block, otherwise margin is added on top
		// and z-index: 1 is necessary on fab
		FABs:         fabs,
		Destinations: destinations,
	}
}

func appendTenantsDestinationFromMenuItems(
	ctx ctxx.Context,
	infra *common.Infra,
	active string,
	destinations []*wx.NavigationDestination,
) []*wx.NavigationDestination {
	for _, item := range infra.PluginRegistry().ExtendMenuItems(ctx, nil) {
		if item.IsDivider || item.Label == nil {
			continue
		}
		label := item.Label.String(ctx)
		if label != "Tenants" && label != "Manage tenants" {
			continue
		}

		return append(destinations, &wx.NavigationDestination{
			Label:     wx.T("Tenants").String(ctx),
			Icon:      "apartment",
			IsActive:  active == "tenants",
			HTMXAttrs: item.HTMXAttrs,
		})
	}

	return destinations
}
