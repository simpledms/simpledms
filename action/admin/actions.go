package admin

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type Actions struct {
	InitAppCmd   *InitAppCmd
	UnlockAppCmd *UnlockAppCmd
	// must also be possible per tenant
	ToggleMaintenanceModeCmd *ToggleMaintenanceModeCmd
	ChangePassphraseCmd      *ChangePassphraseCmd
	RemovePassphraseCmd      *RemovePassphraseCmd
}

func NewActions(infra *common.Infra) *Actions {
	actions := new(Actions)

	*actions = Actions{
		InitAppCmd:               NewInitAppCmd(infra, actions),
		UnlockAppCmd:             NewUnlockAppCmd(infra, actions),
		ToggleMaintenanceModeCmd: NewToggleMaintenanceModeCmd(infra, actions),
		ChangePassphraseCmd:      NewChangePassphraseCmd(infra, actions),
		RemovePassphraseCmd:      NewRemovePassphraseCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.AdminActionsRoute() + path
}
