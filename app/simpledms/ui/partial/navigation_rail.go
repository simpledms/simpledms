package partial

import (
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/ui/route"
	wx "github.com/simpledms/simpledms/ui/widget"
)

// fab must be injected because it differs on each page...
func NewNavigationRail(ctx ctxx.Context, active string, fabs []*wx.FloatingActionButton) *wx.NavigationRail {
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
			Href:     route.BrowseRoot(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
		})
		destinations = append(destinations, &wx.NavigationDestination{
			Label:    "Inbox",
			Icon:     "inbox",
			IsActive: active == "inbox",
			Href:     route.InboxRoot(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
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
		MenuBtn: NewMainMenu(ctx),
		// must be after main block, otherwise margin is added on top
		// and z-index: 1 is necessary on fab
		FABs:         fabs,
		Destinations: destinations,
	}
}
