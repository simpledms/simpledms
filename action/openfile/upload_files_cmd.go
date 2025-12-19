package openfile

import (
	"io"
	"log"
	"net/http"
	"time"

	gonanoid "github.com/matoous/go-nanoid"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type UploadFilesCmdState struct {
	// Cmd string `validate:"required"`
}

type UploadFilesCmdData struct {
	// File []byte `schema:"-"`
}

type UploadFilesCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewUploadFilesCmd(infra *common.Infra, actions *Actions) *UploadFilesCmd {
	config := actionx.NewConfig(
		actions.Route("upload-files"),
		false,
	)
	return &UploadFilesCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *UploadFilesCmd) Data() *UploadFilesCmdData {
	return &UploadFilesCmdData{}
}

func (qq *UploadFilesCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	// don't use autil.FormData to parse form, would load all files into memory, we use
	// multipart reader instead

	// state := autil.StateX[UploadFilesCmdState](rw, req)

	// share target api on mobile
	uploadToken, err := qq.processSharedFiles(rw, req, ctx)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Processing of shared files failed.")
	}

	rw.AddRenderables(wx.NewSnackbarf("Files uploaded, please select a space.")) // not sure if shown
	// not via HX-Redirect because this command is called directly from external (phone) and is thus not a htmx request
	http.Redirect(rw, req.Request, route.SelectSpace(uploadToken), http.StatusFound)

	return nil

}

// FIXME check file size
// FIXME limit number of files?
func (qq *UploadFilesCmd) processSharedFiles(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) (string, error) {
	reader, err := req.MultipartReader()
	if err != nil {
		log.Println(err)
		return "", err
	}

	// just small letters for safety on case insensitive file systems
	// can be relatively short because we check account ID when reading files
	// used identifies an upload of an user
	uploadToken, err := gonanoid.Generate("0123456789abcdefghijklmnopqrstuvwxyz_", 16)
	if err != nil {
		log.Println(err)
		return "", err
	}

	qi := 0
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break // no more parts
		}
		if err != nil {
			log.Println(err)
			return "", err
			// continue // TODO continue or break?
		}

		if part.FormName() != "file" {
			part.Close()
			continue
		}

		// don't use len(files) in case a file goes wrong and there are some incomplete leftovers...
		qi++

		// TODO to long? user has 15 minutes to select a space
		expiresAt := time.Now().Add(15 * time.Minute)

		_, err = qq.infra.FileSystem().SaveTemporaryFileToAccount(ctx, part, part.FileName(), uploadToken, qi, expiresAt)
		if err != nil {
			part.Close()
			// returning instead of continue to enforce cleanup, transaction doesn't get committed and thus
			// no orphaned entries in db; with continue, it could happen that a file is added to db, but upload failed;
			// just is an issue because user could open multiple files at once...
			return "", err
		}

		part.Close()
	}

	// qq.cache.Set(uploadToken, files, cache.DefaultExpiration)
	return uploadToken, nil
}
