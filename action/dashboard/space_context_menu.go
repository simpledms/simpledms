package dashboard

import (
	"github.com/simpledms/simpledms/ctxx"
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

func (qq *SpaceContextMenu) Widget(ctx ctxx.Context, tenantID, spaceID string) *wx.Menu {
	return &wx.Menu{
		// near duplicate in MainMenu and space.SpaceContextMenu
		Items: []*wx.MenuItem{
			{
				LeadingIcon: "edit",
				Label:       wx.T("Edit in «Spaces» view"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route.SpacesRoot(tenantID),
				},
			},
			{
				IsDivider: true,
			},
			{
				LeadingIcon: "category", // TODO category or description?
				Label:       wx.T("Document types"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route.ManageDocumentTypes(tenantID, spaceID),
				},
			},
			{
				LeadingIcon: "label",
				Label:       wx.T("Tags"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route.ManageTags(tenantID, spaceID),
				},
			},
			{
				LeadingIcon: "tune", // tune or assignment
				Label:       wx.T("Fields"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route.ManageProperties(tenantID, spaceID),
				},
			},
			{
				LeadingIcon: "person",
				Label:       wx.T("Users"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route.ManageUsersOfSpace(tenantID, spaceID),
				},
			},
		},
	}
}
