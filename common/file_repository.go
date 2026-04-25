package common

import (
	"github.com/simpledms/simpledms/core/db/entx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
)

type FileRepository struct{}

func NewFileRepository() *FileRepository {
	return &FileRepository{}
}

func (qq *FileRepository) GetWithParentX(ctx ctxx.Context, id string) *filemodel.FileWithParent {
	filex := ctx.TenantCtx().TTx.File.Query().WithParent().Where(file.PublicIDEQ(entx.NewCIText(id))).OnlyX(ctx)

	if filex.ParentID == 0 {
		panic("parent id is 0")
	}

	return filemodel.NewFileWithParent(filemodel.NewFile(filex))
}

func (qq *FileRepository) GetX(ctx ctxx.Context, id string) *filemodel.File {
	/*
		stmt := dd.SELECT(dt.Files.AllColumns).
			FROM(dt.Files).
			WHERE(dt.Files.ID.EQ(dd.Int64(id)))
		dest := &dm.Files{}
		err := stmt.Query(tx, dest)
		if err != nil {
			log.Println(err)
			return nil, err
		}
	*/

	// filex := ctx.TenantCtx().TTx.File.GetX(ctx, id)
	filex := ctx.SpaceCtx().Space.QueryFiles().Where(file.PublicIDEQ(entx.NewCIText(id))).OnlyX(ctx)
	return filemodel.NewFile(filex)
}

func (qq *FileRepository) GetWithDeletedX(ctx ctxx.Context, id string) *filemodel.File {
	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	filex := ctx.SpaceCtx().Space.QueryFiles().Where(file.PublicIDEQ(entx.NewCIText(id))).OnlyX(ctxWithDeleted)
	return filemodel.NewFile(filex)
}

// Deprecated: just a workaround for legacy code
func (qq *FileRepository) GetXX(filex *enttenant.File) *filemodel.File {
	return filemodel.NewFile(filex)
}

/*
func (qq *FileRepo) Save(ctx *ctxx.Context, filex *filemodel.File) error {
	updateStmt := dt.Files.UPDATE(dt.Files.MutableColumns).
		MODEL(filex.Data).
		WHERE(dt.Files.ID.EQ(dd.Int32(filex.Data.ID)))
	_, err := updateStmt.Exec(tx)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
*/
