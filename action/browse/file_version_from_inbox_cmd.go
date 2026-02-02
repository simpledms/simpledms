package browse

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"entgo.io/ent/dialect/sql"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileVersionFromInboxFormData struct {
	TargetFileID string `form_attr_type:"hidden"`
	SourceFileID string `form_attr_type:"hidden"`
}

type FileVersionFromInboxCmdData struct {
	FileVersionFromInboxFormData `structs:",flatten"`
	ConfirmWarning               bool
}

type FileVersionFromInboxCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileVersionFromInboxCmd(infra *common.Infra, actions *Actions) *FileVersionFromInboxCmd {
	config := actionx.NewConfig(actions.Route("file-version-from-inbox-cmd"), false)
	return &FileVersionFromInboxCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileVersionFromInboxCmd) Data(targetFileID, sourceFileID string) *FileVersionFromInboxCmdData {
	return &FileVersionFromInboxCmdData{
		FileVersionFromInboxFormData: FileVersionFromInboxFormData{
			TargetFileID: targetFileID,
			SourceFileID: sourceFileID,
		},
	}
}

func (qq *FileVersionFromInboxCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileVersionFromInboxCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	if !data.ConfirmWarning {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Please confirm that the source file metadata will be lost.")
	}

	if data.TargetFileID == "" || data.SourceFileID == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Source and target files are required.")
	}

	sourceFile, err := qq.actions.FileVersionFromInboxListPartial.findInboxFile(ctx, data.SourceFileID)
	if err != nil {
		log.Println(err)
		return err
	}

	targetFile := qq.infra.FileRepo.GetX(ctx, data.TargetFileID)

	_, err = qq.mergeFromInbox(ctx, sourceFile, targetFile.Data)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.AddRenderables(
		wx.NewSnackbarf("Added new version from inbox."),
	)

	rw.Header().Set("HX-Trigger", fmt.Sprintf("%s, %s, %s, %s", event.FileUploaded.String(), event.FileUpdated.String(), event.FileDeleted.String(), event.CloseDialog.String()))

	return nil
}

func (qq *FileVersionFromInboxCmd) mergeFromInbox(
	ctx ctxx.Context,
	fileToMerge *enttenant.File,
	targetFile *enttenant.File,
) (*enttenant.File, error) {
	if fileToMerge == nil || targetFile == nil {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source and target files are required.")
	}

	if fileToMerge.ID == targetFile.ID {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source and target must be different files.")
	}

	if fileToMerge.SpaceID != ctx.SpaceCtx().Space.ID || targetFile.SpaceID != ctx.SpaceCtx().Space.ID {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File does not belong to current space.")
	}

	if fileToMerge.IsDirectory || targetFile.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot merge directories.")
	}

	if !fileToMerge.DeletedAt.IsZero() {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source file is deleted.")
	}

	sourceVersion, err := fileToMerge.QueryFileVersions().
		Order(fileversion.ByVersionNumber(sql.OrderDesc())).
		WithStoredFile().
		First(ctx)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source file has no versions.")
		}
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not read source version.")
	}

	if sourceVersion.Edges.StoredFile == nil {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source file has no stored file.")
	}

	// TODO should be from model
	latestVersion, err := targetFile.QueryFileVersions().
		Order(fileversion.ByVersionNumber(sql.OrderDesc())).
		First(ctx)
	if err != nil && !enttenant.IsNotFound(err) {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not read target versions.")
	}

	versionNumber := 1
	if err == nil {
		versionNumber = latestVersion.VersionNumber + 1
	}

	ctx.TenantCtx().TTx.FileVersion.Create().
		SetFileID(targetFile.ID).
		SetStoredFileID(sourceVersion.Edges.StoredFile.ID).
		SetVersionNumber(versionNumber).
		SaveX(ctx)

	update := targetFile.Update().
		SetName(fileToMerge.Name).
		SetOcrRetryCount(0).
		SetOcrLastTriedAt(time.Time{})
	if fileToMerge.OcrSuccessAt != nil {
		update.SetOcrContent(fileToMerge.OcrContent)
		update.SetOcrSuccessAt(*fileToMerge.OcrSuccessAt)
	} else {
		update.SetOcrContent("")
		update.ClearOcrSuccessAt()
	}
	targetFile, err = update.Save(ctx)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not update target file.")
	}

	// hard delete the file in the inbox
	if !fileToMerge.IsInInbox {
		// safety check, we never want to chance to delete a file that is not in the inbox
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source file is not in inbox.")
	}
	_, err = ctx.TenantCtx().TTx.FileVersion.Delete().Where(fileversion.FileID(fileToMerge.ID)).Exec(ctx)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not remove source versions.")
	}

	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	err = ctx.TenantCtx().TTx.File.DeleteOneID(fileToMerge.ID).Exec(ctxWithDeleted)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not delete source file.")
	}

	return targetFile, nil
}
