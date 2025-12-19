package model

import (
	"github.com/simpledms/simpledms/db/enttenant"
)

type Space struct {
	Data *enttenant.Space
}

func NewSpace(space *enttenant.Space) *Space {
	return &Space{space}
}

// Enable a document type and tags library
// TODO Enable or Subscribe?
func (qq *Space) EnableLibrary() {
	// TODO
}
