package download

import (
	"net/http"
	"strconv"

	commonaction "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/storedfile"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

// TODO is this a good name and is `page` package the correct location?
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
	filex := qq.infra.FileRepo.GetX(ctx, fileIDStr)

	if filex.Data.IsDirectory {
		// TODO impl support for this? download as zip archive?
		return e.NewHTTPErrorf(http.StatusBadRequest, "cannot download directories")
	}

	versionID := req.URL.Query().Get("version_id")
	if versionID != "" {
		versionInt, err := strconv.ParseInt(versionID, 10, 64)
		if err != nil {
			return e.NewHTTPErrorf(http.StatusBadRequest, "invalid version id")
		}
		version, err := filex.Data.QueryVersions().Where(storedfile.ID(versionInt)).Only(ctx)
		if err != nil {
			if enttenant.IsNotFound(err) {
				return e.NewHTTPErrorf(http.StatusNotFound, "version not found")
			}
			return err
		}
		return commonaction.StreamDownload(qq.infra, ctx, rw, req, filex, model.NewStoredFile(version))
	}

	currentVersion := filex.CurrentVersion(ctx)
	return commonaction.StreamDownload(qq.infra, ctx, rw, req, filex, currentVersion)
}
