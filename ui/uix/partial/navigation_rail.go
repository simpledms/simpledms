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
			Label:    "Sign in",
			Icon:     "login",
			IsActive: active == "sign-in",
			Href:     "/",
		})
	}

	if ctx.IsSpaceCtx() {
		if isMetadataRail {
			destinations = append(destinations, &wx.NavigationDestination{
				Label:    "Browse",
				Icon:     "folder_open",
				IsActive: false,
				Href:     route2.BrowseRoot(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
			destinations = append(destinations, &wx.NavigationDestination{
				Label:    "Document types",
				Icon:     "category",
				IsActive: active == "document-types",
				Href:     route2.ManageDocumentTypes(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
			destinations = append(destinations, &wx.NavigationDestination{
				Label:    "Tags",
				Icon:     "label",
				IsActive: active == "tags",
				Href:     route2.ManageTags(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
			destinations = append(destinations, &wx.NavigationDestination{
				Label:    "Fields",
				Icon:     "tune",
				IsActive: active == "fields",
				Href:     route2.ManageProperties(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
		} else {
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
			destinations = append(destinations, &wx.NavigationDestination{
				Label:    "Document types",
				Icon:     "category",
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
