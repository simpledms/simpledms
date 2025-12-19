package admin

import (
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/util/actionx"
)

type ToggleMaintenanceModeData struct{}

type ToggleMaintenanceMode struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewToggleMaintenanceMode(infra *common.Infra, actions *Actions) *ToggleMaintenanceMode {
	config := actionx.NewConfig(actions.Route("toggle-maintenance-mode"), false)
	return &ToggleMaintenanceMode{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}
