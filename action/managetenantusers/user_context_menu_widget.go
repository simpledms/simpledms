package managetenantusers

import (
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	"github.com/simpledms/simpledms/ui/util"
)

type UserContextMenuWidget struct {
	actions *Actions
}

func NewUserContextMenuWidget(actions *Actions) *UserContextMenuWidget {
	return &UserContextMenuWidget{
		actions: actions,
	}
}

func (qq *UserContextMenuWidget) Widget(
	ctx ctxx.Context,
	userx *enttenant.User,
	isOwningTenantAssignment bool,
) *widget.Menu {
	if ctx.TenantCtx().User.Role != tenantrole.Owner {
		return nil
	}
	if userx.AccountID == ctx.MainCtx().Account.ID {
		return nil
	}

	// Member accounts are only removed from this organization; accounts owned by it are deleted.
	icon := "person_remove"
	label := widget.T("Remove")
	hxConfirm := widget.T("Remove this user from the organization?").String(ctx)
	if isOwningTenantAssignment {
		icon = "delete"
		label = widget.T("Delete")
		hxConfirm = widget.T(
			"Remove this user from the organization and delete their account globally?",
		).String(ctx)
	}

	return &widget.Menu{
		Items: []*widget.MenuItem{
			{
				LeadingIcon: icon,
				Label:       label,
				HTMXAttrs: widget.HTMXAttrs{
					HxPost:    qq.actions.DeleteUserCmd.Endpoint(),
					HxSwap:    "none",
					HxVals:    util.JSON(qq.actions.DeleteUserCmd.Data(userx.PublicID.String())),
					HxConfirm: hxConfirm,
				},
			},
		},
	}
}
