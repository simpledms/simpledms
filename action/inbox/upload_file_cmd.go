package inbox

// package action

import (
	"log"
	"net/http"
	"path/filepath"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/filesystem"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type uploadPrepareResult struct {
	prepared *filesystem.PreparedUpload
	file     *enttenant.File
}

type UploadFileCmdData struct {
	File []byte `schema:"-"`
}

type UploadFileCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[UploadFileCmdData]
	// inboxDirInfo *ent.FileInfo
}

func NewUploadFileCmd(infra *common.Infra, actions *Actions) *UploadFileCmd {
	config := actionx.NewConfig(
		actions.Route("upload-file-cmd"),
		false,
	).EnableManualTxManagement()
	formHelper := autil.NewFormHelper[UploadFileCmdData](
		infra,
		config,
		wx.T("Upload file"),
	)
	formHelper.SetIsMultipartFormData(true)
	return &UploadFileCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: formHelper,
		// inboxDirInfo: infra.Factory().InboxDirInfo(),
	}
}

func (qq *UploadFileCmd) Data() *UploadFileCmdData {
	return &UploadFileCmdData{}
}

func (qq *UploadFileCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	_, err := autil.FormData[UploadFileCmdData](rw, req, ctx)
	if err != nil {
		return err
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
	filename = filepath.Clean(filename)

	if err := autil.EnsureFileDoesNotExist(ctx, filename, ctx.SpaceCtx().SpaceRootDir().ID, true); err != nil {
		return err
	}

	prep, err := autil.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*uploadPrepareResult, error) {
		prepared, filex, err := qq.infra.FileSystem().PrepareFileUpload(
			writeCtx,
			filename,
			ctx.SpaceCtx().SpaceRootDir().ID,
			true,
		)
		if err != nil {
			return nil, err
		}
		return &uploadPrepareResult{prepared: prepared, file: filex}, nil
	})
	if err != nil {
		return err
	}

	fileInfo, fileSize, err := qq.infra.FileSystem().UploadPreparedFileWithExpectedSize(
		ctx,
		uploadedFile,
		prep.prepared,
		uploadedFileHeader.Size,
	)
	if err != nil {
		autil.HandleStoredFileUploadFailure(ctx.SpaceCtx(), qq.infra.FileSystem(), prep.prepared, err, true)
		return err
	}

	_, err = autil.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*struct{}, error) {
		return nil, qq.infra.FileSystem().FinalizePreparedUpload(writeCtx, prep.prepared, fileInfo, fileSize)
	})
	if err != nil {
		autil.HandleStoredFileUploadFailure(ctx.SpaceCtx(), qq.infra.FileSystem(), prep.prepared, err, false)
		return err
	}

	rw.Header().Set("HX-Retarget", "#innerContent")
	rw.Header().Set("HX-Reswap", "innerHTML")

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.actions.InboxPage.WidgetHandler(rw, req, ctx, prep.file.PublicID.String()),
		wx.NewSnackbarf("«%s» uploaded.", prep.file.Name),
	)
}
