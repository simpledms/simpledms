package dashboard

import (
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
)

type PasskeyContextMenuWidget struct {
	actions *Actions
}

func NewPasskeyContextMenuWidget(actions *Actions) *PasskeyContextMenuWidget {
	return &PasskeyContextMenuWidget{
		actions: actions,
	}
}

func (qq *PasskeyContextMenuWidget) Widget(ctx ctxx.Context, passkeyID string, passkeyName string) *widget.Menu {
	return &widget.Menu{
		Items: []*widget.MenuItem{
			{
				LeadingIcon: "edit",
				Label:       widget.T("Rename"),
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
				Label:       widget.T("Remove"),
				HTMXAttrs: widget.HTMXAttrs{
					HxPost:    qq.actions.AuthActions.DeletePasskeyCmd.Endpoint(),
					HxVals:    util.JSON(qq.actions.AuthActions.DeletePasskeyCmd.Data(passkeyID)),
					HxConfirm: widget.T("Are you sure?").String(ctx),
					HxSwap:    "none",
				},
			},
		},
	}
}
