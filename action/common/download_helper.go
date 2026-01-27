package common

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

func StreamDownload(
	infra *common.Infra,
	ctx ctxx.Context,
	rw httpx.ResponseWriter,
	req *httpx.Request,
	filex *model.File,
	currentVersion *model.StoredFile,
) error {
	if filex.Data.IsDirectory {
		return e.NewHTTPErrorf(http.StatusBadRequest, "cannot download directories")
	}

	f, err := infra.FileSystem().OpenFile(ctx, currentVersion)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "")
	}
	defer f.Close()

	_, err = io.Copy(rw, f)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "")
	}

	if req.URL.Query().Get("inline") == "1" {
		rw.Header().Set("Content-Disposition", "inline")
	} else {
		rw.Header().Set("Content-Disposition", fmt.Sprintf(
			"attachment; filename=\"%s\"; filename*=UTF-8''%s",
			url.QueryEscape(filex.Data.Name),
			url.QueryEscape(filex.Data.Name),
		))
	}

	mimeType := currentVersion.Data.MimeType
	rw.Header().Set("Content-Type", mimeType)

	rw.WriteHeader(http.StatusOK)
	return nil
}
