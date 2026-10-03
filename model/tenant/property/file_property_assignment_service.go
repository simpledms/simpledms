package property

import (
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	propertyquery "github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	"github.com/simpledms/simpledms/util/e"
)

type FilePropertyAssignmentService struct{}

func NewFilePropertyAssignmentService() *FilePropertyAssignmentService {
	return &FilePropertyAssignmentService{}
}

func (qq *FilePropertyAssignmentService) Set(
	ctx ctxx.Context,
	fileID int64,
	propertyID int64,
	value FilePropertyValue,
) (*enttenant.Property, *enttenant.FilePropertyAssignment, error) {
	filex, propertyx, err := qq.resolve(ctx, fileID, propertyID)
	if err != nil {
		return nil, nil, err
	}
	if propertyx.Type != value.typex {
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, "Value does not match the field type.")
	}

	assignment, err := ctx.SpaceCtx().TTx.FilePropertyAssignment.Query().Where(
		filepropertyassignment.FileID(filex.ID),
		filepropertyassignment.PropertyID(propertyx.ID),
	).Only(ctx)
	if err != nil && !enttenant.IsNotFound(err) {
		return nil, nil, err
	}
	if enttenant.IsNotFound(err) {
		create := ctx.SpaceCtx().TTx.FilePropertyAssignment.Create().
			SetSpaceID(ctx.SpaceCtx().Space.ID).
			SetFileID(filex.ID).
			SetPropertyID(propertyx.ID)
		applyFilePropertyCreate(create, value)
		assignment, err = create.Save(ctx)
	} else {
		update := assignment.Update().
			ClearTextValue().
			ClearNumberValue().
			ClearDateValue().
			ClearBoolValue()
		applyFilePropertyUpdate(update, value)
		assignment, err = update.Save(ctx)
	}
	return propertyx, assignment, err
}

func (qq *FilePropertyAssignmentService) Remove(
	ctx ctxx.Context,
	fileID int64,
	propertyID int64,
) (*enttenant.Property, bool, error) {
	filex, propertyx, err := qq.resolve(ctx, fileID, propertyID)
	if err != nil {
		return nil, false, err
	}
	deleted, err := ctx.SpaceCtx().TTx.FilePropertyAssignment.Delete().Where(
		filepropertyassignment.FileID(filex.ID),
		filepropertyassignment.PropertyID(propertyx.ID),
	).Exec(ctx)
	return propertyx, deleted > 0, err
}

func (qq *FilePropertyAssignmentService) resolve(
	ctx ctxx.Context,
	fileID int64,
	propertyID int64,
) (*enttenant.File, *enttenant.Property, error) {
	filex, err := ctx.SpaceCtx().Space.QueryFiles().Where(file.ID(fileID)).Only(ctx)
	if err != nil {
		return nil, nil, err
	}
	if filex.IsDirectory {
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, "File is a folder.")
	}
	propertyx, err := ctx.SpaceCtx().Space.QueryProperties().Where(
		propertyquery.ID(propertyID),
	).Only(ctx)
	return filex, propertyx, err
}

func applyFilePropertyCreate(
	query *enttenant.FilePropertyAssignmentCreate,
	value FilePropertyValue,
) {
	switch value.typex {
	case fieldtype.Text:
		query.SetTextValue(value.text)
	case fieldtype.Number, fieldtype.Money:
		query.SetNumberValue(value.number)
	case fieldtype.Date:
		query.SetDateValue(value.date)
	case fieldtype.Checkbox:
		query.SetBoolValue(value.checkbox)
	}
}

func applyFilePropertyUpdate(
	query *enttenant.FilePropertyAssignmentUpdateOne,
	value FilePropertyValue,
) {
	switch value.typex {
	case fieldtype.Text:
		query.SetTextValue(value.text)
	case fieldtype.Number, fieldtype.Money:
		query.SetNumberValue(value.number)
	case fieldtype.Date:
		query.SetDateValue(value.date)
	case fieldtype.Checkbox:
		query.SetBoolValue(value.checkbox)
	}
}
