package managespaceusers

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/common/spacerole"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type UserAssignmentContextMenuWidget struct {
	actions *Actions
}

func NewUserAssignmentContextMenuWidget(actions *Actions) *UserAssignmentContextMenuWidget {
	return &UserAssignmentContextMenuWidget{
		actions: actions,
	}
}

func (qq *UserAssignmentContextMenuWidget) Widget(ctx ctxx.Context, userAssignment *enttenant.SpaceUserAssignment) *wx.Menu {
	var items []*wx.MenuItem

	if ctx.SpaceCtx().UserRoleInSpace() == spacerole.Owner {
		items = append(items, &wx.MenuItem{
			LeadingIcon: "delete",
			Label:       wx.T("Unassign"),
			HTMXAttrs: wx.HTMXAttrs{
				HxPost: qq.actions.UnassignUserFromSpaceCmd.Endpoint(),
				HxVals: util.JSON(qq.actions.UnassignUserFromSpaceCmd.Data(userAssignment.ID)),
			},
		})
	}

	if len(items) == 0 {
		return nil
	}

	return &wx.Menu{
		Items: items,
	}
}
