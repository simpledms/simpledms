package partial

import (
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
)

// fab must be injected because it differs on each page...
func NewNavigationRail(ctx ctxx.Context, infra *common.Infra, active string, fabs []*widget.FloatingActionButton) *widget.NavigationRail {
	var destinations []*widget.NavigationDestination
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
		destinations = append(destinations, &widget.NavigationDestination{
			Label:    widget.T("Sign in [subject]").String(ctx),
			Icon:     "login",
			IsActive: active == "sign-in",
			Href:     "/",
		})
	}

	if ctx.IsSpaceCtx() {
		if isMetadataRail {
			destinations = append(destinations, &widget.NavigationDestination{
				Label:    widget.T("Files").String(ctx),
				Icon:     "folder_open",
				IsActive: false,
				Href:     route2.BrowseRoot(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
			destinations = append(destinations, &widget.NavigationDestination{
				Label:    widget.T("Document types").String(ctx),
				Icon:     "category",
				IsActive: active == "document-types",
				Href:     route2.ManageDocumentTypes(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
			destinations = append(destinations, &widget.NavigationDestination{
				Label:    widget.T("Tags").String(ctx),
				Icon:     "label",
				IsActive: active == "tags",
				Href:     route2.ManageTags(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
			destinations = append(destinations, &widget.NavigationDestination{
				Label:    widget.T("Fields").String(ctx),
				Icon:     "tune",
				IsActive: active == "fields",
				Href:     route2.ManageProperties(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
		} else {
			destinations = append(destinations, &widget.NavigationDestination{
				Label:    widget.T("Files").String(ctx), // TODO Files or Browse?
				Icon:     "folder_open",
				IsActive: active == "browse",
				Href:     route2.BrowseRoot(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
			destinations = append(destinations, &widget.NavigationDestination{
				Label:    widget.T("Inbox").String(ctx),
				Icon:     "inbox",
				IsActive: active == "inbox",
				Href:     route2.InboxRoot(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
			})
			destinations = append(destinations, &widget.NavigationDestination{
				Label:    widget.T("Metadata").String(ctx),
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

	return &widget.NavigationRail{
		MenuBtn: NewMainMenu(ctx, infra),
		// must be after main block, otherwise margin is added on top
		// and z-index: 1 is necessary on fab
		FABs:         fabs,
		Destinations: destinations,
	}
}
