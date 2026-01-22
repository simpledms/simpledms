package download

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
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

	currentVersion := filex.CurrentVersion(ctx)

	/*
		path, err := currentVersion.Path(ctx)
		if err != nil {
			log.Println(err)
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			log.Println(err)
			return e.NewHTTPErrorf(http.StatusInternalServerError, "")
		}
		defer f.Close()
	*/

	f, err := qq.infra.FileSystem().OpenFile(ctx, currentVersion)
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
			url.QueryEscape(filex.Data.Name), // FIXME remove non utf-8 chars
			url.QueryEscape(filex.Data.Name),
		))
	}

	mimeType := currentVersion.Data.MimeType
	rw.Header().Set("Content-Type", mimeType)

	rw.WriteHeader(http.StatusOK)
	return nil
}
