package managetenantusers

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/model/common/tenantrole"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/simpledms/simpledms/db/enttenant"
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

	hxConfirm := widget.T("Are you sure? This user will be removed from this organization only.").String(ctx)
	if isOwningTenantAssignment {
		hxConfirm = widget.T("Are you sure? This user will be removed from this organization and the account will be deleted globally.").String(ctx)
	}

	return &widget.Menu{
		Items: []*widget.MenuItem{
			{
				LeadingIcon: "delete",
				Label:       widget.T("Delete"),
				HTMXAttrs: widget.HTMXAttrs{
					HxPost:    qq.actions.DeleteUserCmd.Endpoint(),
					HxVals:    util.JSON(qq.actions.DeleteUserCmd.Data(userx.PublicID.String())),
					HxConfirm: hxConfirm,
				},
			},
		},
	}
}
