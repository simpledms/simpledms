package common

import (
	"errors"
	"log"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/entx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
)

type FileRepository struct{}

func NewFileRepository() *FileRepository {
	return &FileRepository{}
}

func (qq *FileRepository) GetWithParent(ctx ctxx.Context, id string) (*filemodel.FileWithParent, error) {
	filex, err := ctx.TenantCtx().TTx.File.Query().
		WithParent().
		Where(file.PublicIDEQ(entx.NewCIText(id))).
		Only(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if filex.ParentID == 0 {
		err = errors.New("parent id is 0")
		log.Println(err)
		return nil, err
	}

	return filemodel.NewFileWithParent(filemodel.NewFile(filex)), nil
}

func (qq *FileRepository) Get(ctx ctxx.Context, id string) (*filemodel.File, error) {
	filex, err := filemodel.NewFileReader().Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return filemodel.NewFile(filex), nil
}

// Deprecated: panics on error; only kept for widget builders without an error result. Use Get.
func (qq *FileRepository) GetX(ctx ctxx.Context, id string) *filemodel.File {
	filex, err := qq.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return filex
}

func (qq *FileRepository) GetWithDeleted(ctx ctxx.Context, id string) (*filemodel.File, error) {
	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	filex, err := ctx.SpaceCtx().Space.QueryFiles().
		Where(file.PublicIDEQ(entx.NewCIText(id))).
		Only(ctxWithDeleted)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return filemodel.NewFile(filex), nil
}

// Deprecated: panics on error; only kept for widget builders without an error result. Use
// GetWithDeleted.
func (qq *FileRepository) GetWithDeletedX(ctx ctxx.Context, id string) *filemodel.File {
	filex, err := qq.GetWithDeleted(ctx, id)
	if err != nil {
		panic(err)
	}
	return filex
}
