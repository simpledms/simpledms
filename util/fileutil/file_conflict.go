package fileutil

import (
	"net/http"

	"github.com/marcobeierer/go-core/util/e"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
)

func EnsureFileDoesNotExist(ctx ctxx.Context, filename string, parentDirID int64, isInInbox bool) error {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return nil
	}

	fileExists := ctx.SpaceCtx().Space.QueryFiles().
		Where(file.Name(filename), file.ParentID(parentDirID), file.IsInInbox(isInInbox)).
		ExistX(ctx)
	if fileExists {
		return e.NewHTTPErrorf(http.StatusBadRequest, "File already exists.")
	}

	return nil
}
