package managespaceusers

import (
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/tenant/common/spacerole"
)

type UserAssignmentContextMenuWidget struct {
	actions *Actions
}

func NewUserAssignmentContextMenuWidget(actions *Actions) *UserAssignmentContextMenuWidget {
	return &UserAssignmentContextMenuWidget{
		actions: actions,
	}
}

func (qq *UserAssignmentContextMenuWidget) Widget(ctx ctxx.Context, userAssignment *enttenant.SpaceUserAssignment) *widget.Menu {
	var items []*widget.MenuItem

	if ctx.SpaceCtx().UserRoleInSpace() == spacerole.Owner {
		items = append(items, &widget.MenuItem{
			LeadingIcon: "delete",
			Label:       widget.T("Unassign"),
			HTMXAttrs: widget.HTMXAttrs{
				HxPost: qq.actions.UnassignUserFromSpaceCmd.Endpoint(),
				HxVals: util.JSON(qq.actions.UnassignUserFromSpaceCmd.Data(userAssignment.ID)),
			},
		})
	}

	if len(items) == 0 {
		return nil
	}

	return &widget.Menu{
		Items: items,
	}
}
