package trash

import (
	"net/http"

	commonaction "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type Download struct {
	infra *common.Infra
}

func NewDownload(infra *common.Infra) *Download {
	return &Download{
		infra: infra,
	}
}

func (qq *Download) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	fileIDStr := req.PathValue("file_id")
	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	filex := qq.infra.FileRepo.GetWithDeletedX(ctx, fileIDStr)

	if filex.Data.IsDirectory {
		return e.NewHTTPErrorf(http.StatusBadRequest, "cannot download directories")
	}

	currentVersion := filex.CurrentVersion(ctxWithDeleted)
	return commonaction.StreamDownload(qq.infra, ctx, rw, req, filex, currentVersion)
}
