package property

import (
	"github.com/simpledms/simpledms/db/enttenant"
)

type Property struct {
	Data *enttenant.Property
}

func NewProperty(data *enttenant.Property) *Property {
	return &Property{data}
}
