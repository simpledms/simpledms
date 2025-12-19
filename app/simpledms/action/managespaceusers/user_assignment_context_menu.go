package managespaceusers

import (
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/enttenant"
	"github.com/simpledms/simpledms/app/simpledms/model/common/spacerole"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type UserAssignmentContextMenu struct {
	actions *Actions
}

func NewUserAssignmentContextMenu(actions *Actions) *UserAssignmentContextMenu {
	return &UserAssignmentContextMenu{
		actions: actions,
	}
}

func (qq *UserAssignmentContextMenu) Widget(ctx ctxx.Context, userAssignment *enttenant.SpaceUserAssignment) *wx.Menu {
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
