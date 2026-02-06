package dashboard

import (
	"github.com/simpledms/simpledms/ctxx"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type SpaceContextMenuWidget struct {
	actions *Actions
}

func NewSpaceContextMenuWidget(actions *Actions) *SpaceContextMenuWidget {
	return &SpaceContextMenuWidget{
		actions: actions,
	}
}

func (qq *SpaceContextMenuWidget) Widget(ctx ctxx.Context, tenantID, spaceID string) *wx.Menu {
	return &wx.Menu{
		// near duplicate in MainMenu and space.SpaceContextMenuWidget
		Items: []*wx.MenuItem{
			{
				LeadingIcon: "edit",
				Label:       wx.T("Edit in «Spaces» view"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route2.SpacesRoot(tenantID),
				},
			},
			{
				IsDivider: true,
			},
			{
				LeadingIcon: "category", // TODO category or description?
				Label:       wx.T("Document types"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route2.ManageDocumentTypes(tenantID, spaceID),
				},
			},
			{
				LeadingIcon: "label",
				Label:       wx.T("Tags"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route2.ManageTags(tenantID, spaceID),
				},
			},
			{
				LeadingIcon: "tune", // tune or assignment
				Label:       wx.T("Fields"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route2.ManageProperties(tenantID, spaceID),
				},
			},
			{
				LeadingIcon: "person",
				Label:       wx.T("Users"),
				HTMXAttrs: wx.HTMXAttrs{
					HxGet: route2.ManageUsersOfSpace(tenantID, spaceID),
				},
			},
		},
	}
}
