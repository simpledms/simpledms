package documenttype

import (
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type Actions struct {
	Common                  *acommon.Actions
	DocumentTypePage        *DocumentTypePage
	ManageDocumentTypesPage *ManageDocumentTypesPage

	ListDocumentTypesPartial *ListDocumentTypesPartial
	CreateCmd                *CreateCmd
	DeleteCmd                *DeleteCmd
	RenameCmd                *RenameCmd
	DetailsPartial           *DetailsPartial

	Properties               *AttributesPartial
	CreateAttributeCmd       *CreateAttributeCmd
	AddPropertyAttributeCmd  *AddPropertyAttributeCmd
	DeleteAttributeCmd       *DeleteAttributeCmd
	EditTagAttributeCmd      *EditTagAttributeCmd
	EditPropertyAttributeCmd *EditPropertyAttributeCmd
}

func NewActions(infra *common.Infra, commonActions *acommon.Actions) *Actions {
	actions := new(Actions)

	*actions = Actions{
		Common:                  commonActions,
		DocumentTypePage:        NewDocumentTypePage(infra, actions),
		ManageDocumentTypesPage: NewManageDocumentTypesPage(infra, actions),

		ListDocumentTypesPartial: NewListDocumentTypesPartial(infra, actions),
		CreateCmd:                NewCreateCmd(infra, actions),
		DeleteCmd:                NewDeleteCmd(infra, actions),
		RenameCmd:                NewRenameCmd(infra, actions),

		DetailsPartial: NewDetailsPartial(infra, actions),

		Properties:               NewAttributesPartial(infra, actions),
		CreateAttributeCmd:       NewCreateAttributeCmd(infra, actions),
		AddPropertyAttributeCmd:  NewAddPropertyAttributeCmd(infra, actions),
		DeleteAttributeCmd:       NewDeleteAttributeCmd(infra, actions),
		EditTagAttributeCmd:      NewEditTagAttributeCmd(infra, actions),
		EditPropertyAttributeCmd: NewEditPropertyAttributeCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.ManageDocumentTypesActionsRoute() + path
}
