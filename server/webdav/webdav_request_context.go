package webdav

import (
	"context"
	"io"
	"net/http"

	"golang.org/x/net/webdav"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	enttenantwebdavresource "github.com/simpledms/simpledms/db/enttenant/webdavresource"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	credentialmodel "github.com/simpledms/simpledms/model/main/webdavcredential"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	webdavresourcemodel "github.com/simpledms/simpledms/model/tenant/webdavresource"
)

type webDAVRequestContext struct {
	context.Context
	handler        *Handler
	credential     *credentialmodel.AuthRecord
	tenantPublicID string
	spacePublicID  string
	contentLength  int64
	op             *webDAVOperation
}

func (qq *webDAVRequestContext) openUploadFile(pathx webDAVPath) (webdav.File, error) {
	resourceID, prepared, uploadCtx, err := qq.reserveUpload(pathx)
	if err != nil {
		qq.op.err = err
		return nil, err
	}
	pipeReader, pipeWriter := io.Pipe()
	uploadFile := &webDAVUploadFile{
		requestCtx: qq,
		resourceID: resourceID,
		prepared:   prepared,
		uploadCtx:  uploadCtx,
		pipeReader: pipeReader,
		pipeWriter: pipeWriter,
		done:       make(chan error, 1),
	}
	go uploadFile.upload()
	return uploadFile, nil
}

func (qq *webDAVRequestContext) reserveUpload(
	pathx webDAVPath,
) (int64, *filesystem.PreparedUpload, *ctxx.SpaceContext, error) {
	tenantTx, spaceCtx, err := qq.tenantWriteSpaceContext()
	if err != nil {
		return 0, nil, nil, err
	}
	committed := false
	defer rollbackTenantTx(tenantTx, &committed)

	if _, err := spaceCtx.TTx.WebDAVResource.Delete().
		Where(
			enttenantwebdavresource.CredentialPublicID(entx.NewCIText(qq.credential.PublicID)),
			enttenantwebdavresource.SpaceID(spaceCtx.Space.ID),
			enttenantwebdavresource.DavPath(pathx.canonical),
			enttenantwebdavresource.StateEQ(webdavresourcemodel.Active),
			enttenantwebdavresource.Or(
				enttenantwebdavresource.Not(enttenantwebdavresource.HasFile()),
				enttenantwebdavresource.HasFileWith(
					file.Or(file.DeletedAtNotNil(), file.IsInInbox(false)),
				),
			),
		).
		Exec(spaceCtx); err != nil {
		return 0, nil, nil, err
	}

	conflict, err := spaceCtx.TTx.WebDAVResource.Query().
		Where(
			enttenantwebdavresource.CredentialPublicID(entx.NewCIText(qq.credential.PublicID)),
			enttenantwebdavresource.SpaceID(spaceCtx.Space.ID),
			enttenantwebdavresource.DavPath(pathx.canonical),
			enttenantwebdavresource.StateIn(webdavresourcemodel.Uploading, webdavresourcemodel.Active),
		).
		Exist(spaceCtx)
	if err != nil {
		return 0, nil, nil, err
	}
	if conflict {
		return 0, nil, nil, webDAVStatusError{status: http.StatusConflict, msg: "active alias exists"}
	}

	resourcex, err := spaceCtx.TTx.WebDAVResource.Create().
		SetCredentialPublicID(entx.NewCIText(qq.credential.PublicID)).
		SetSpaceID(spaceCtx.Space.ID).
		SetDavPath(pathx.canonical).
		Save(spaceCtx)
	if err != nil {
		return 0, nil, nil, webDAVStatusError{status: http.StatusConflict, msg: "alias reservation failed"}
	}
	prepared, err := qq.handler.infra.FileSystem().PrepareFileUploadIntentWithSource(
		spaceCtx,
		pathx.filename,
		spaceCtx.SpaceRootDir().ID,
		true,
		filesource.WebDAV,
	)
	if err != nil {
		return 0, nil, nil, err
	}
	resourcex, err = resourcex.Update().SetStoredFileID(prepared.StoredFileID).Save(spaceCtx)
	if err != nil {
		return 0, nil, nil, err
	}
	if err := tenantTx.Commit(); err != nil {
		return 0, nil, nil, err
	}
	committed = true
	return resourcex.ID, prepared, spaceCtx, nil
}

func (qq *webDAVRequestContext) renameActiveResource(oldPath webDAVPath, newPath webDAVPath) error {
	tenantTx, spaceCtx, err := qq.tenantWriteSpaceContext()
	if err != nil {
		return err
	}
	committed := false
	defer rollbackTenantTx(tenantTx, &committed)

	destinationActive, err := tenantTx.WebDAVResource.Query().
		Where(
			enttenantwebdavresource.CredentialPublicID(entx.NewCIText(qq.credential.PublicID)),
			enttenantwebdavresource.SpaceID(spaceCtx.Space.ID),
			enttenantwebdavresource.DavPath(newPath.canonical),
			enttenantwebdavresource.StateIn(webdavresourcemodel.Uploading, webdavresourcemodel.Active),
		).
		Exist(spaceCtx)
	if err != nil {
		return err
	}
	if destinationActive {
		return webDAVStatusError{status: http.StatusConflict, msg: "destination active"}
	}

	resourcex, err := tenantTx.WebDAVResource.Query().
		Where(
			enttenantwebdavresource.CredentialPublicID(entx.NewCIText(qq.credential.PublicID)),
			enttenantwebdavresource.SpaceID(spaceCtx.Space.ID),
			enttenantwebdavresource.DavPath(oldPath.canonical),
			enttenantwebdavresource.StateEQ(webdavresourcemodel.Active),
		).
		Only(spaceCtx)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return webDAVStatusError{status: http.StatusConflict, msg: "source inactive"}
		}
		return err
	}
	if resourcex.FileID == nil {
		_ = tenantTx.WebDAVResource.DeleteOneID(resourcex.ID).Exec(spaceCtx)
		_ = tenantTx.Commit()
		committed = true
		return webDAVStatusError{status: http.StatusConflict, msg: "source missing file"}
	}
	filex, err := tenantTx.File.Query().
		Where(
			file.ID(*resourcex.FileID),
			file.SpaceID(spaceCtx.Space.ID),
			file.IsDirectory(false),
			file.IsInInbox(true),
		).
		Only(spaceCtx)
	if err != nil {
		if enttenant.IsNotFound(err) {
			_ = tenantTx.WebDAVResource.DeleteOneID(resourcex.ID).Exec(spaceCtx)
			_ = tenantTx.Commit()
			committed = true
			return webDAVStatusError{status: http.StatusConflict, msg: "source left inbox"}
		}
		return err
	}
	filename, err := uniqueInboxFilename(spaceCtx, newPath.filename)
	if err != nil {
		return err
	}
	if _, err := filex.Update().SetName(filename).Save(spaceCtx); err != nil {
		return err
	}
	if _, err := resourcex.Update().SetDavPath(newPath.canonical).Save(spaceCtx); err != nil {
		return err
	}
	if err := tenantTx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (qq *webDAVRequestContext) tenantWriteSpaceContext() (*enttenant.Tx, *ctxx.SpaceContext, error) {
	mainTx, err := qq.handler.mainDB.Tx(qq, true)
	if err != nil {
		return nil, nil, err
	}
	committedMain := false
	defer rollbackMainTx(mainTx, &committedMain)
	spaceCtx, tenantTx, err := qq.handler.webDAVSpaceContext(
		qq,
		mainTx,
		qq.credential,
		qq.tenantPublicID,
		qq.spacePublicID,
		false,
		nil,
	)
	if err != nil {
		return nil, nil, err
	}
	if err := mainTx.Commit(); err != nil {
		_ = tenantTx.Rollback()
		return nil, nil, err
	}
	committedMain = true
	return tenantTx, spaceCtx, nil
}
