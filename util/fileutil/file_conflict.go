package fileutil

import (
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/util/e"
)

func EnsureFileDoesNotExist(ctx ctxx.Context, filename string, parentDirID int64, isInInbox bool) error {
	if !ctx.IsSpaceCtx() {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Space context is required.")
	}

	if !ctx.SpaceCtx().Space.IsFolderMode {
		return nil
	}

	readRepo := filemodel.NewEntSpaceFileReadRepository(ctx.SpaceCtx().Space.ID)
	fileExists := readRepo.FileExistsByNameAndParentX(ctx, filename, parentDirID, isInInbox)
	if fileExists {
		return e.NewHTTPErrorf(http.StatusBadRequest, "File already exists.")
	}

	return nil
}
