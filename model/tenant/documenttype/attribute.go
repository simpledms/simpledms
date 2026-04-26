package documenttype

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/attribute"
)

type Attribute struct {
	Data *enttenant.Attribute
}

func NewAttribute(data *enttenant.Attribute) *Attribute {
	return &Attribute{
		Data: data,
	}
}

func QueryAttributeByID(ctx ctxx.Context, attributeID int64) (*Attribute, error) {
	attributex, err := ctx.AppCtx().TTx.Attribute.Query().
		Where(attribute.ID(attributeID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return NewAttribute(attributex), nil
}

func (qq *Attribute) SetIsNameGiving(ctx ctxx.Context, isNameGiving bool) error {
	attributex, err := ctx.AppCtx().TTx.Attribute.UpdateOneID(qq.Data.ID).
		SetIsNameGiving(isNameGiving).
		Save(ctx)
	if err != nil {
		return err
	}

	qq.Data = attributex

	return nil
}

func (qq *Attribute) RenameAndSetIsNameGiving(ctx ctxx.Context, newName string, isNameGiving bool) error {
	attributex, err := ctx.AppCtx().TTx.Attribute.UpdateOneID(qq.Data.ID).
		SetName(newName).
		SetIsNameGiving(isNameGiving).
		Save(ctx)
	if err != nil {
		return err
	}

	qq.Data = attributex

	return nil
}

func (qq *Attribute) Delete(ctx ctxx.Context) error {
	return ctx.AppCtx().TTx.Attribute.DeleteOneID(qq.Data.ID).Exec(ctx)
}
