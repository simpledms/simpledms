package dashboard

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type PasskeyContextMenuWidget struct {
	actions *Actions
}

func NewPasskeyContextMenuWidget(actions *Actions) *PasskeyContextMenuWidget {
	return &PasskeyContextMenuWidget{
		actions: actions,
	}
}

func (qq *PasskeyContextMenuWidget) Widget(ctx ctxx.Context, passkeyID string, passkeyName string) *wx.Menu {
	return &wx.Menu{
		Items: []*wx.MenuItem{
			{
				LeadingIcon: "edit",
				Label:       wx.T("Rename"),
				HTMXAttrs: qq.actions.AuthActions.RenamePasskeyCmd.ModalLinkAttrs(
					qq.actions.AuthActions.RenamePasskeyCmd.Data(passkeyID, passkeyName),
					"",
				),
			},
			{
				IsDivider: true,
			},
			{
				LeadingIcon: "delete",
				Label:       wx.T("Remove"),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:    qq.actions.AuthActions.DeletePasskeyCmd.Endpoint(),
					HxVals:    util.JSON(qq.actions.AuthActions.DeletePasskeyCmd.Data(passkeyID)),
					HxConfirm: wx.T("Are you sure?").String(ctx),
					HxSwap:    "none",
				},
			},
		},
	}
}
