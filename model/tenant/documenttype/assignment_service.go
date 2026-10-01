package documenttype

import (
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	documenttypequery "github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/util/e"
)

type AssignmentService struct{}

func NewAssignmentService() *AssignmentService {
	return &AssignmentService{}
}

func (qq *AssignmentService) Set(
	ctx ctxx.Context,
	fileID int64,
	documentTypeID int64,
) (*enttenant.DocumentType, error) {
	filex, err := qq.file(ctx, fileID)
	if err != nil {
		return nil, err
	}
	documentTypex, err := ctx.SpaceCtx().Space.QueryDocumentTypes().Where(
		documenttypequery.ID(documentTypeID),
	).Only(ctx)
	if err != nil {
		return nil, err
	}
	if filex.DocumentTypeID != documentTypex.ID {
		if err := filex.Update().SetDocumentTypeID(documentTypex.ID).Exec(ctx); err != nil {
			return nil, err
		}
	}
	return documentTypex, nil
}

func (qq *AssignmentService) Clear(ctx ctxx.Context, fileID int64) (bool, error) {
	filex, err := qq.file(ctx, fileID)
	if err != nil || filex.DocumentTypeID == 0 {
		return false, err
	}
	return true, filex.Update().ClearDocumentTypeID().Exec(ctx)
}

func (qq *AssignmentService) Toggle(
	ctx ctxx.Context,
	fileID int64,
	documentTypeID int64,
) (bool, *enttenant.DocumentType, error) {
	filex, err := qq.file(ctx, fileID)
	if err != nil {
		return false, nil, err
	}
	if filex.DocumentTypeID == documentTypeID {
		_, err := qq.Clear(ctx, fileID)
		return false, nil, err
	}
	documentTypex, err := qq.Set(ctx, fileID, documentTypeID)
	return err == nil, documentTypex, err
}

func (qq *AssignmentService) file(ctx ctxx.Context, fileID int64) (*enttenant.File, error) {
	filex, err := ctx.SpaceCtx().Space.QueryFiles().Where(file.ID(fileID)).Only(ctx)
	if err != nil {
		return nil, err
	}
	if filex.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File is a directory.")
	}
	return filex, nil
}
