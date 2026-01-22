package admin

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/util/actionx"
)

type ToggleMaintenanceModeCmdData struct{}

type ToggleMaintenanceModeCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewToggleMaintenanceModeCmd(infra *common.Infra, actions *Actions) *ToggleMaintenanceModeCmd {
	config := actionx.NewConfig(actions.Route("toggle-maintenance-mode"), false)
	return &ToggleMaintenanceModeCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}
