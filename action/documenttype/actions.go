package documenttype

import (
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/uix/route"
)

type Actions struct {
	Common *acommon.Actions
	Page   *Page

	ListDocumentTypes *ListDocumentTypes
	Create            *Create
	Delete            *Delete
	Rename            *Rename
	Details           *Details

	Properties            *Attributes
	CreateAttribute       *CreateAttribute
	AddPropertyAttribute  *AddPropertyAttribute
	DeleteAttribute       *DeleteAttribute
	EditTagAttribute      *EditTagAttribute
	EditPropertyAttribute *EditPropertyAttribute
}

func NewActions(infra *common.Infra, commonActions *acommon.Actions) *Actions {
	actions := new(Actions)

	*actions = Actions{
		Common: commonActions,
		Page:   NewPage(infra, actions),

		ListDocumentTypes: NewListDocumentTypes(infra, actions),
		Create:            NewCreate(infra, actions),
		Delete:            NewDelete(infra, actions),
		Rename:            NewRename(infra, actions),

		Details: NewDetails(infra, actions),

		Properties:            NewAttributes(infra, actions),
		CreateAttribute:       NewCreateAttribute(infra, actions),
		AddPropertyAttribute:  NewAddPropertyAttribute(infra, actions),
		DeleteAttribute:       NewDeleteAttribute(infra, actions),
		EditTagAttribute:      NewEditTagAttribute(infra, actions),
		EditPropertyAttribute: NewEditPropertyAttribute(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.ManageDocumentTypesActionsRoute() + path
}
