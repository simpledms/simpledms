package webdav

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/webdav"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	tenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	enttenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/space"
	enttenantwebdavresource "github.com/simpledms/simpledms/db/enttenant/webdavresource"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	webdavresourcemodel "github.com/simpledms/simpledms/model/tenant/webdavresource"
)

const (
	webDAVResourceHeartbeatInterval = 30 * time.Second
	webDAVResourceHeartbeatTimeout  = 2 * time.Second
)

type webDAVUploadFile struct {
	requestCtx *webDAVRequestContext
	resourceID int64
	prepared   *filesystem.PreparedUpload
	uploadCtx  *ctxx.SpaceContext
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	done       chan error
	nextBeat   time.Time
	written    int64
	closed     bool
}

func (qq *webDAVUploadFile) upload() {
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("webdav upload failed: %v", r)
		}
		if err != nil {
			qq.setUploadError(err)
			qq.cleanupFailedUpload()
		}
		qq.done <- err
	}()

	result, err := qq.uploadPreparedFile()
	if err == nil && qq.requestCtx.contentLength > 0 && result.FileSize != qq.requestCtx.contentLength {
		err = webDAVStatusError{status: http.StatusBadRequest, msg: "upload size mismatch"}
	}
	if err == nil {
		err = qq.finalize(result)
	}
}

func (qq *webDAVUploadFile) setUploadError(err error) {
	if err == nil {
		return
	}
	if qq.requestCtx != nil && qq.requestCtx.op != nil {
		qq.requestCtx.op.err = err
	}
	if closeErr := qq.pipeReader.CloseWithError(err); closeErr != nil && !errors.Is(closeErr, io.ErrClosedPipe) {
		log.Println(closeErr)
	}
}

func (qq *webDAVUploadFile) uploadPreparedFile() (*filesystem.PreparedUploadResult, error) {
	return qq.requestCtx.handler.infra.FileSystem().UploadPreparedFile(
		qq.uploadCtx,
		qq.pipeReader,
		qq.prepared,
	)
}

func (qq *webDAVUploadFile) finalize(result *filesystem.PreparedUploadResult) error {
	err := qq.requestCtx.handler.withFinalizationContexts(
		qq.requestCtx,
		qq.requestCtx.credential,
		qq.requestCtx.tenantPublicID,
		qq.requestCtx.spacePublicID,
		qq.resourceID,
		func(spaceCtx *ctxx.SpaceContext) error {
			filename, err := uniqueInboxFilename(spaceCtx, qq.prepared.OriginalFilename)
			if err != nil {
				return err
			}
			if err := qq.requestCtx.handler.infra.FileSystem().FinalizePreparedUploadAsWithoutMime(
				spaceCtx,
				qq.prepared,
				result,
				filename,
			); err != nil {
				return err
			}
			return spaceCtx.TTx.WebDAVResource.UpdateOneID(qq.resourceID).
				SetState(webdavresourcemodel.Active).
				SetFileID(qq.prepared.FileID).
				SetStoredFileID(qq.prepared.StoredFileID).
				SetFinalizedAt(time.Now()).
				Exec(spaceCtx)
		},
	)
	if err != nil {
		return err
	}
	if _, err := qq.requestCtx.handler.infra.FileSystem().UpdateMimeTypeAfterFinalization(
		qq.uploadCtx,
		true,
		qq.prepared.StoredFileID,
	); err != nil {
		log.Println(err)
	}
	return nil
}

func (qq *webDAVUploadFile) cleanupFailedUpload() {
	if qq.requestCtx == nil || qq.requestCtx.handler == nil || qq.prepared == nil {
		return
	}
	if err := qq.requestCtx.handler.infra.FileSystem().RemoveTemporaryObject(
		qq.requestCtx,
		qq.prepared.TemporaryStoragePath,
		qq.prepared.TemporaryStorageFilename,
	); err != nil {
		log.Println(err)
		qq.markCleanupPending()
		return
	}

	if err := qq.requestCtx.handler.cleanupFailedWebDAVUpload(
		qq.requestCtx,
		qq.requestCtx.credential,
		qq.requestCtx.spacePublicID,
		qq.resourceID,
		qq.prepared.StoredFileID,
	); err != nil {
		log.Println(err)
	}
}

func (qq *webDAVUploadFile) markCleanupPending() {
	if qq.requestCtx == nil || qq.requestCtx.handler == nil || qq.prepared == nil {
		return
	}
	if err := qq.requestCtx.handler.markWebDAVUploadCleanupPending(
		qq.requestCtx,
		qq.requestCtx.credential,
		qq.requestCtx.spacePublicID,
		qq.resourceID,
		qq.prepared.StoredFileID,
	); err != nil {
		log.Println(err)
	}
}

func (qq *webDAVUploadFile) Close() error {
	if qq.closed {
		return nil
	}
	qq.closed = true
	closeErr := qq.pipeWriter.Close()
	uploadErr := <-qq.done
	if closeErr != nil {
		return closeErr
	}
	if uploadErr != nil {
		return uploadErr
	}
	return nil
}

func (qq *webDAVUploadFile) Read([]byte) (int, error)           { return 0, io.EOF }
func (qq *webDAVUploadFile) Readdir(int) ([]os.FileInfo, error) { return nil, os.ErrInvalid }
func (qq *webDAVUploadFile) Seek(int64, int) (int64, error)     { return 0, os.ErrInvalid }
func (qq *webDAVUploadFile) Stat() (os.FileInfo, error) {
	return webDAVFileInfo{name: qq.prepared.OriginalFilename, size: qq.written}, nil
}
func (qq *webDAVUploadFile) Write(p []byte) (int, error) {
	if len(p) > 0 {
		if err := qq.refreshResourceHeartbeat(time.Now()); err != nil {
			qq.setUploadError(err)
			return 0, err
		}
	}
	n, err := qq.pipeWriter.Write(p)
	qq.written += int64(n)
	return n, err
}

func (qq *webDAVUploadFile) refreshResourceHeartbeat(now time.Time) error {
	if !qq.nextBeat.IsZero() && now.Before(qq.nextBeat) {
		return nil
	}
	if qq.requestCtx == nil || qq.requestCtx.handler == nil || qq.requestCtx.credential == nil {
		return errors.New("missing WebDAV upload context")
	}
	tenantDB, ok := qq.requestCtx.handler.tenantDBs.Load(qq.requestCtx.credential.TenantID)
	if !ok {
		return errors.New("tenant db missing")
	}
	heartbeatCtx, cancel := context.WithTimeout(qq.requestCtx, webDAVResourceHeartbeatTimeout)
	defer cancel()
	heartbeatCtx = tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(heartbeatCtx),
		tenantprivacy.Allow,
	)
	touched, err := tenantDB.ReadWriteConn.WebDAVResource.Update().
		Where(
			enttenantwebdavresource.ID(qq.resourceID),
			enttenantwebdavresource.CredentialPublicID(
				entx.NewCIText(qq.requestCtx.credential.PublicID),
			),
			enttenantwebdavresource.StateEQ(webdavresourcemodel.Uploading),
			enttenantwebdavresource.HasSpaceWith(
				space.PublicIDEQ(entx.NewCIText(qq.requestCtx.spacePublicID)),
			),
		).
		SetLastProgressAt(now).
		Save(heartbeatCtx)
	if err != nil {
		return err
	}
	if touched != 1 {
		return webDAVStatusError{status: http.StatusConflict, msg: "reservation inactive"}
	}
	qq.nextBeat = now.Add(webDAVResourceHeartbeatInterval)
	return nil
}

func uniqueInboxFilename(ctx *ctxx.SpaceContext, desired string) (string, error) {
	names, err := ctx.TTx.File.Query().
		Where(file.SpaceID(ctx.Space.ID), file.IsInInbox(true), file.IsDirectory(false), file.DeletedAtIsNil()).
		Select(file.FieldName).
		Strings(ctx)
	if err != nil {
		return "", err
	}
	used := make(map[string]struct{}, len(names))
	for _, name := range names {
		used[name] = struct{}{}
	}
	if _, ok := used[desired]; !ok {
		return desired, nil
	}
	ext := filepath.Ext(desired)
	base := strings.TrimSuffix(desired, ext)
	for i := 2; ; i++ {
		candidate := base + " (" + strconv.Itoa(i) + ")" + ext
		if _, ok := used[candidate]; !ok {
			return candidate, nil
		}
	}
}

var _ webdav.File = (*webDAVUploadFile)(nil)
