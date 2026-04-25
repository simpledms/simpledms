package property

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/model/common/fieldtype"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/property"
)

type PropertyService struct{}

func NewPropertyService() *PropertyService {
	return &PropertyService{}
}

func (qq *PropertyService) Create(
	ctx ctxx.Context,
	spaceID int64,
	name string,
	propertyType fieldtype.FieldType,
	unit string,
) (*enttenant.Property, error) {
	return ctx.SpaceCtx().TTx.Property.Create().
		SetName(name).
		SetType(propertyType).
		SetUnit(unit).
		SetSpaceID(spaceID).
		Save(ctx)
}

func (qq *PropertyService) Edit(
	ctx ctxx.Context,
	space *enttenant.Space,
	propertyID int64,
	name string,
	unit string,
) (*enttenant.Property, error) {
	propertyx, err := space.QueryProperties().Where(property.ID(propertyID)).Only(ctx)
	if err != nil {
		return nil, err
	}

	return propertyx.Update().
		SetName(name).
		SetUnit(unit).
		Save(ctx)
}

func (qq *PropertyService) Delete(ctx ctxx.Context, space *enttenant.Space, propertyID int64) error {
	propertyx, err := space.QueryProperties().Where(property.ID(propertyID)).Only(ctx)
	if err != nil {
		return err
	}

	return ctx.SpaceCtx().TTx.Property.DeleteOne(propertyx).Exec(ctx)
}
