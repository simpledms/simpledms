package browse

import (
	"log"
	"net/http"
	"path/filepath"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/filesystem"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type UploadFileCmdData struct {
	ParentDirID string `form_attr_type:"hidden"`
	File        []byte `schema:"-"`
	// for renaming
	// TODO preset to uploaded file name
	// TODO option to quickly rename according to pattern defined for folder
	Filename   string // TODO only if in FolderMode
	AddToInbox bool   // TODO only in non-folder mode
}

type UploadFileCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[UploadFileCmdData]
}

func NewUploadFileCmd(
	infra *common.Infra,
	actions *Actions,
) *UploadFileCmd {
	config := actionx.NewConfig(
		actions.Route("upload-file-cmd"),
		false,
	).EnableManualTxManagement()

	formHelper := autil.NewFormHelper[UploadFileCmdData](
		infra,
		config,
		wx.T("Upload file"),
		// "#fileList",
	)
	formHelper.SetIsMultipartFormData(true)

	return &UploadFileCmd{
		infra,
		actions,
		config,
		formHelper,
	}
}

func (qq *UploadFileCmd) Data(parentDirID string, filename string, addToInbox bool) *UploadFileCmdData {
	return &UploadFileCmdData{
		ParentDirID: parentDirID,
		File:        []byte(""),
		Filename:    filename,
		AddToInbox:  addToInbox,
	}
}

// very similar to UploadFileVersionCmd
func (qq *UploadFileCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[UploadFileCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	if data.ParentDirID == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "No parent dir provided.")
	}

	uploadedFile, uploadedFileHeader, err := req.FormFile("File")
	if err != nil {
		// TODO also triggers if no file provided
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "could not read file")
	}
	defer func() {
		if err := uploadedFile.Close(); err != nil {
			log.Println(err)
		}
	}()

	filename := uploadedFileHeader.Filename
	if data.Filename != "" {
		filename = data.Filename
	}
	filename = filepath.Clean(filename)

	parentDir := qq.infra.FileRepo.GetX(ctx, data.ParentDirID)

	if err := autil.EnsureFileDoesNotExist(ctx, filename, parentDir.Data.ID, data.AddToInbox); err != nil {
		return err
	}

	prepared, filex, err := func() (*filesystem.PreparedUpload, *enttenant.File, error) {
		tenantDB, ok := ctx.SpaceCtx().UnsafeTenantDB()
		if !ok {
			log.Println("tenant db not found", ctx.TenantCtx().Tenant.ID)
			return nil, nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Tenant database not found.")
		}

		writeTx, err := tenantDB.Tx(ctx, false)
		if err != nil {
			log.Println(err)
			return nil, nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not start transaction.")
		}
		committed := false
		defer func() {
			if committed {
				return
			}
			if err := writeTx.Rollback(); err != nil {
				log.Println(err)
			}
		}()

		writeTenantCtx := ctxx.NewTenantContext(ctx.MainCtx(), writeTx, ctx.TenantCtx().Tenant)
		writeSpace := writeTx.Space.GetX(writeTenantCtx, ctx.SpaceCtx().Space.ID)
		writeSpaceCtx := ctxx.NewSpaceContext(writeTenantCtx, writeSpace)

		prepared, filex, err := qq.infra.FileSystem().PrepareFileUpload(
			writeSpaceCtx,
			filename,
			parentDir.Data.ID,
			data.AddToInbox,
		)
		if err != nil {
			return nil, nil, err
		}

		if err := writeTx.Commit(); err != nil {
			log.Println(err)
			return nil, nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not save file.")
		}
		committed = true

		return prepared, filex, nil
	}()
	if err != nil {
		return err
	}

	fileInfo, fileSize, err := qq.infra.FileSystem().UploadPreparedFile(ctx, uploadedFile, prepared)
	if err != nil {
		autil.HandleStoredFileUploadFailure(ctx.SpaceCtx(), qq.infra.FileSystem(), prepared, err, true)
		return err
	}

	_, err = autil.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*struct{}, error) {
		return nil, qq.infra.FileSystem().FinalizePreparedUpload(writeCtx, prepared, fileInfo, fileSize)
	})
	if err != nil {
		autil.HandleStoredFileUploadFailure(ctx.SpaceCtx(), qq.infra.FileSystem(), prepared, err, false)
		return err
	}

	rw.AddRenderables(wx.NewSnackbarf("«%s» uploaded.", filex.Name))
	// TODO does triggering event have an effect? request comes from uppy and isn't a HTMX request...
	rw.Header().Add("HX-Trigger", event.FileUploaded.String())

	return nil
}
