package trash

import (
	"net/http"

	"entgo.io/ent/dialect/sql"
	commonaction "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
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
	repos := qq.infra.SpaceFileRepoFactory().ForSpaceX(ctx)
	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	fileDTO := repos.Read.FileByPublicIDWithDeletedX(ctx, fileIDStr)

	if fileDTO.IsDirectory {
		return e.NewHTTPErrorf(http.StatusBadRequest, "cannot download directories")
	}

	version, err := ctx.TenantCtx().TTx.FileVersion.Query().
		Where(fileversion.FileID(fileDTO.ID)).
		Order(fileversion.ByVersionNumber(sql.OrderDesc())).
		WithStoredFile().
		First(ctxWithDeleted)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusNotFound, "version not found")
		}
		return err
	}

	return commonaction.StreamDownload(
		qq.infra,
		ctx,
		rw,
		req,
		fileDTO.Name,
		fileDTO.IsDirectory,
		storedfilemodel.NewStoredFile(version.Edges.StoredFile),
	)
}
