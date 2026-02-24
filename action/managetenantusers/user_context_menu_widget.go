package managetenantusers

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/common/tenantrole"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
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
) *wx.Menu {
	if ctx.TenantCtx().User.Role != tenantrole.Owner {
		return nil
	}
	if userx.AccountID == ctx.MainCtx().Account.ID {
		return nil
	}

	hxConfirm := wx.T("Are you sure? This user will be removed from this organization only.").String(ctx)
	if isOwningTenantAssignment {
		hxConfirm = wx.T("Are you sure? This user will be removed from this organization and the account will be deleted globally.").String(ctx)
	}

	return &wx.Menu{
		Items: []*wx.MenuItem{
			{
				LeadingIcon: "delete",
				Label:       wx.T("Delete"),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:    qq.actions.DeleteUserCmd.Endpoint(),
					HxVals:    util.JSON(qq.actions.DeleteUserCmd.Data(userx.PublicID.String())),
					HxConfirm: hxConfirm,
				},
			},
		},
	}
}
