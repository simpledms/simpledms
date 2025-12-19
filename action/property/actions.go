package property

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/uix/route"
)

type Actions struct {
	PropertiesPage *PropertiesPage
	PropertyList   *PropertyList

	CreateProperty *CreateProperty
	EditProperty   *EditProperty
	DeleteProperty *DeleteProperty
}

func NewActions(infra *common.Infra) *Actions {
	actions := new(Actions)

	*actions = Actions{
		PropertiesPage: NewPropertiesPage(infra, actions),
		PropertyList:   NewPropertyList(infra, actions),

		CreateProperty: NewCreateProperty(infra, actions),
		EditProperty:   NewEditProperty(infra, actions),
		DeleteProperty: NewDeleteProperty(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.PropertyActionsRoute() + path
}
