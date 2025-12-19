package admin

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/uix/route"
)

type Actions struct {
	InitApp   *InitApp
	UnlockApp *UnlockApp
	// must also be possible per tenant
	ToggleMaintenanceMode *ToggleMaintenanceMode
	ChangePassphrase      *ChangePassphrase
	RemovePassphrase      *RemovePassphrase
}

func NewActions(infra *common.Infra) *Actions {
	actions := new(Actions)

	*actions = Actions{
		InitApp:               NewInitApp(infra, actions),
		UnlockApp:             NewUnlockApp(infra, actions),
		ToggleMaintenanceMode: NewToggleMaintenanceMode(infra, actions),
		ChangePassphrase:      NewChangePassphrase(infra, actions),
		RemovePassphrase:      NewRemovePassphrase(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.AdminActionsRoute() + path
}
