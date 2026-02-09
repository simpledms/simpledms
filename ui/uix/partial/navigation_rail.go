package partial

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
)

// fab must be injected because it differs on each page...
func NewNavigationRail(ctx ctxx.Context, infra *common.Infra, active string, fabs []*wx.FloatingActionButton) *wx.NavigationRail {
	var destinations []*wx.NavigationDestination
	/*
		{
			Label:    "Files", // TODO or Finder?
			Icon:     "folder_open",
			IsActive: active == "find",
			Href:     route.FindRoot(),
		},

	*/

	if ctx.IsSpaceCtx() {
		destinations = append(destinations, &wx.NavigationDestination{
			Label:    "Browse", // TODO Files or Browse?
			Icon:     "folder_open",
			IsActive: active == "browse",
			Href:     route2.BrowseRoot(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
		})
		destinations = append(destinations, &wx.NavigationDestination{
			Label:    "Inbox",
			Icon:     "inbox",
			IsActive: active == "inbox",
			Href:     route2.InboxRoot(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
		})
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
