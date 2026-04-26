package property

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type Actions struct {
	PropertiesPage      *PropertiesPage
	PropertyListPartial *PropertyListPartial

	CreatePropertyCmd *CreatePropertyCmd
	EditPropertyCmd   *EditPropertyCmd
	DeletePropertyCmd *DeletePropertyCmd
}

func NewActions(infra *common.Infra) *Actions {
	actions := new(Actions)

	*actions = Actions{
		PropertiesPage:      NewPropertiesPage(infra, actions),
		PropertyListPartial: NewPropertyListPartial(infra, actions),

		CreatePropertyCmd: NewCreatePropertyCmd(infra, actions),
		EditPropertyCmd:   NewEditPropertyCmd(infra, actions),
		DeletePropertyCmd: NewDeletePropertyCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.PropertyActionsRoute() + path
}
