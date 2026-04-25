package dashboard

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
)

type SpaceContextMenuWidget struct {
	actions *Actions
}

func NewSpaceContextMenuWidget(actions *Actions) *SpaceContextMenuWidget {
	return &SpaceContextMenuWidget{
		actions: actions,
	}
}

func (qq *SpaceContextMenuWidget) Widget(ctx ctxx.Context, tenantID, spaceID string) *widget.Menu {
	return &widget.Menu{
		// near duplicate in MainMenu and space.SpaceContextMenuWidget
		Items: []*widget.MenuItem{
			{
				LeadingIcon: "edit",
				Label:       widget.T("Edit in «Spaces» view"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route2.SpacesRoot(tenantID),
				},
			},
			{
				IsDivider: true,
			},
			{
				LeadingIcon: "category", // TODO category or description?
				Label:       widget.T("Document types"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route2.ManageDocumentTypes(tenantID, spaceID),
				},
			},
			{
				LeadingIcon: "label",
				Label:       widget.T("Tags"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route2.ManageTags(tenantID, spaceID),
				},
			},
			{
				LeadingIcon: "tune", // tune or assignment
				Label:       widget.T("Fields"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route2.ManageProperties(tenantID, spaceID),
				},
			},
			{
				LeadingIcon: "person",
				Label:       widget.T("Users"),
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route2.ManageUsersOfSpace(tenantID, spaceID),
				},
			},
		},
	}
}
