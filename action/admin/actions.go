package admin

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type Actions struct {
	InitAppCmd   *InitAppCmd
	UnlockAppCmd *UnlockAppCmd
	// must also be possible per tenant
	ToggleMaintenanceModeCmd         *ToggleMaintenanceModeCmd
	ChangePassphraseCmd              *ChangePassphraseCmd
	RemovePassphraseCmd              *RemovePassphraseCmd
	SetGlobalUploadLimitForm         *SetGlobalUploadLimitForm
	SetGlobalUploadLimitCmd          *SetGlobalUploadLimitCmd
	SetTenantUploadLimitOverrideForm *SetTenantUploadLimitOverrideForm
	SetTenantUploadLimitOverrideCmd  *SetTenantUploadLimitOverrideCmd
}

func NewActions(infra *common.Infra) *Actions {
	actions := new(Actions)

	*actions = Actions{
		InitAppCmd:                       NewInitAppCmd(infra, actions),
		UnlockAppCmd:                     NewUnlockAppCmd(infra, actions),
		ToggleMaintenanceModeCmd:         NewToggleMaintenanceModeCmd(infra, actions),
		ChangePassphraseCmd:              NewChangePassphraseCmd(infra, actions),
		RemovePassphraseCmd:              NewRemovePassphraseCmd(infra, actions),
		SetGlobalUploadLimitForm:         NewSetGlobalUploadLimitForm(infra, actions),
		SetGlobalUploadLimitCmd:          NewSetGlobalUploadLimitCmd(infra, actions),
		SetTenantUploadLimitOverrideForm: NewSetTenantUploadLimitOverrideForm(infra, actions),
		SetTenantUploadLimitOverrideCmd:  NewSetTenantUploadLimitOverrideCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.AdminActionsRoute() + path
}
