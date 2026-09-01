package webdav

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"time"

	"golang.org/x/net/webdav"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	mainaccount "github.com/simpledms/simpledms/db/entmain/account"
	mainprivacy "github.com/simpledms/simpledms/db/entmain/privacy"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	entmainwebdavcredential "github.com/simpledms/simpledms/db/entmain/webdavcredential"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/predicate"
	tenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	enttenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/user"
	enttenantwebdavresource "github.com/simpledms/simpledms/db/enttenant/webdavresource"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/i18n"
	tenant2 "github.com/simpledms/simpledms/model/main/tenant"
	credentialmodel "github.com/simpledms/simpledms/model/main/webdavcredential"
	webdavresourcemodel "github.com/simpledms/simpledms/model/tenant/webdavresource"
	"github.com/simpledms/simpledms/util/e"
)

const (
	// Pattern is the WebDAV endpoint route registered on the server mux.
	Pattern             = "/webdav/{tenant_public_id}/{space_public_id}/"
	webDAVAllow         = "OPTIONS, PROPFIND, PUT, LOCK, UNLOCK, MOVE"
	webDAVRealm         = `Basic realm="SimpleDMS WebDAV", charset="UTF-8"`
	webDAVMaxXMLBytes   = 64 * 1024
	webDAVMaxStatusBody = 64 * 1024
)

// Handler serves the SimpleDMS WebDAV ingestion endpoint.
type Handler struct {
	mainDB            *sqlx.MainDB
	tenantDBs         *tenantdbs.TenantDBs
	infra             *common.Infra
	devMode           bool
	metaPath          string
	i18n              *i18n.I18n
	credentialService *credentialmodel.CredentialService
	limiter           *webDAVRateLimiter
	locks             *webDAVLockSystem
	trustedProxies    []netip.Prefix
}

// NewHandler creates a WebDAV endpoint handler with bounded auth and lock state.
func NewHandler(config Config) *Handler {
	return &Handler{
		mainDB:            config.MainDB,
		tenantDBs:         config.TenantDBs,
		infra:             config.Infra,
		devMode:           config.DevMode,
		metaPath:          config.MetaPath,
		i18n:              config.I18n,
		credentialService: credentialmodel.NewCredentialService(),
		limiter:           newWebDAVRateLimiter(4096),
		locks:             newWebDAVLockSystem(),
		trustedProxies:    config.TrustedProxies,
	}
}

// ServeHTTP handles authenticated WebDAV protocol requests.
func (qq *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	tenantPublicID := req.PathValue("tenant_public_id")
	spacePublicID := req.PathValue("space_public_id")
	endpointPrefix := "/webdav/" + tenantPublicID + "/" + spacePublicID

	if !qq.isSecureWebDAVRequest(req) {
		writeWebDAVText(rw, http.StatusForbidden, "Forbidden")
		return
	}

	credentialx, ok := qq.authenticatedWebDAVCredential(rw, req)
	if !ok {
		return
	}
	if !qq.authorizeWebDAVEndpoint(rw, req, credentialx, tenantPublicID, spacePublicID) {
		return
	}

	if !qq.prepareWebDAVRequest(rw, req, endpointPrefix) {
		return
	}

	op := &webDAVOperation{method: req.Method}
	ctx := &webDAVRequestContext{
		Context:        req.Context(),
		handler:        qq,
		credential:     credentialx,
		tenantPublicID: tenantPublicID,
		spacePublicID:  spacePublicID,
		contentLength:  req.ContentLength,
		op:             op,
	}
	req = req.WithContext(context.WithValue(ctx, webDAVContextKey{}, ctx))

	handler := &webdav.Handler{
		Prefix:     endpointPrefix,
		FileSystem: &webDAVFileSystem{},
		LockSystem: qq.locks.forCredential(credentialx.PublicID),
		Logger: func(_ *http.Request, err error) {
			if err != nil && !errors.Is(err, webdav.ErrConfirmationFailed) {
				log.Println("webdav request failed:", err)
			}
		},
	}
	recorder := newWebDAVResponseRecorder(rw)
	handler.ServeHTTP(recorder, req)
	recorder.flush(op)
}

func (qq *Handler) authorizeWebDAVEndpoint(
	rw http.ResponseWriter,
	req *http.Request,
	credentialx *credentialmodel.AuthRecord,
	tenantPublicID string,
	spacePublicID string,
) bool {
	if credentialx.TenantPublicID != tenantPublicID || credentialx.SpacePublicID != spacePublicID {
		writeWebDAVText(rw, http.StatusNotFound, "Not found")
		return false
	}
	if err := qq.authorizeWebDAVRequest(req.Context(), credentialx, tenantPublicID, spacePublicID); err != nil {
		status := http.StatusForbidden
		if errors.Is(err, sql.ErrNoRows) || entmain.IsNotFound(err) || enttenant.IsNotFound(err) {
			status = http.StatusForbidden
		}
		writeWebDAVText(rw, status, http.StatusText(status))
		return false
	}
	qq.credentialService.TouchLastUsed(
		req.Context(),
		qq.mainDB.ReadWriteConn,
		credentialx.ID,
		credentialx.LastUsedAt,
	)
	return true
}

func (qq *Handler) prepareWebDAVRequest(
	rw http.ResponseWriter,
	req *http.Request,
	endpointPrefix string,
) bool {
	pathx, ok := parseWebDAVPath(req, endpointPrefix)
	if !ok {
		writeWebDAVText(rw, webDAVInvalidPathStatus(req.Method), http.StatusText(webDAVInvalidPathStatus(req.Method)))
		return false
	}
	if req.Method == http.MethodOptions {
		writeWebDAVOptions(rw)
		return false
	}
	if (req.Method == http.MethodGet || req.Method == http.MethodHead) && pathx.isFile {
		writeWebDAVText(rw, http.StatusNotFound, "Not found")
		return false
	}
	if !webDAVMethodAllowed(req.Method) {
		writeWebDAVMethodNotAllowed(rw)
		return false
	}
	if err := qq.preflightWebDAVRequest(req, endpointPrefix, pathx); err != nil {
		writeWebDAVText(rw, webDAVHTTPStatus(err), http.StatusText(webDAVHTTPStatus(err)))
		return false
	}
	return true
}

func (qq *Handler) authenticatedWebDAVCredential(
	rw http.ResponseWriter,
	req *http.Request,
) (*credentialmodel.AuthRecord, bool) {
	username, secret, ok := req.BasicAuth()
	rateLimitAddr := qq.webDAVRateLimitRemoteAddr(req)
	if !ok || username == "" || qq.limiter.blocked(rateLimitAddr, username) {
		writeWebDAVChallenge(rw)
		return nil, false
	}

	credentialx, ok, err := qq.authenticateWebDAVCredential(req.Context(), username, secret)
	if err != nil {
		log.Println(err)
		qq.limiter.allow(rateLimitAddr, username)
		writeWebDAVChallenge(rw)
		return nil, false
	}
	if !ok || credentialx.RevokedAt != nil {
		qq.limiter.allow(rateLimitAddr, username)
		writeWebDAVChallenge(rw)
		return nil, false
	}
	return credentialx, true
}

func (qq *Handler) authenticateWebDAVCredential(
	ctx context.Context,
	username string,
	secret string,
) (*credentialmodel.AuthRecord, bool, error) {
	ctx = mainprivacy.DecisionContext(ctx, mainprivacy.Allow)
	record, found, err := qq.credentialService.AuthRecordByUsername(ctx, qq.mainDB.ReadOnlyConn, username)
	if err != nil {
		return nil, false, err
	}
	if !qq.credentialService.VerifySecret(record, secret) || !found {
		return nil, false, nil
	}
	return record, true, nil
}

func (qq *Handler) authorizeWebDAVRequest(
	ctx context.Context,
	credentialx *credentialmodel.AuthRecord,
	tenantPublicID string,
	spacePublicID string,
) error {
	mainTx, err := qq.mainDB.Tx(ctx, true)
	if err != nil {
		return err
	}
	committedMain := false
	defer rollbackMainTx(mainTx, &committedMain)

	spaceCtx, tenantTx, err := qq.webDAVSpaceContext(ctx, mainTx, credentialx, tenantPublicID, spacePublicID, true, nil)
	if err != nil {
		return err
	}
	committedTenant := false
	defer rollbackTenantTx(tenantTx, &committedTenant)

	if _, err := tenantTx.Space.Query().Where(space.ID(spaceCtx.Space.ID)).Only(spaceCtx); err != nil {
		return err
	}
	// Commit the independent credential update first so a main DB failure cannot
	// leave a finalized tenant row whose temporary object is then cleaned up.
	if err := mainTx.Commit(); err != nil {
		return err
	}
	committedMain = true
	if err := tenantTx.Commit(); err != nil {
		return err
	}
	committedTenant = true
	return nil
}

func (qq *Handler) webDAVSpaceContext(
	ctx context.Context,
	mainTx *entmain.Tx,
	credentialx *credentialmodel.AuthRecord,
	tenantPublicID string,
	spacePublicID string,
	isReadOnly bool,
	beforeTenantAuth func(context.Context, *enttenant.Tx) error,
) (*ctxx.SpaceContext, *enttenant.Tx, error) {
	ctx = mainprivacy.DecisionContext(ctx, mainprivacy.Allow)
	accountx, err := mainTx.Account.Query().
		Where(mainaccount.ID(credentialx.AccountID), mainaccount.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		return nil, nil, err
	}
	tenantx, err := mainTx.Tenant.Query().
		Where(
			tenant.ID(credentialx.TenantID),
			tenant.PublicID(entx.NewCIText(tenantPublicID)),
			tenant.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	hasAssignment, err := mainTx.TenantAccountAssignment.Query().
		Where(
			tenantaccountassignment.AccountID(accountx.ID),
			tenantaccountassignment.TenantID(tenantx.ID),
			tenantaccountassignment.Or(
				tenantaccountassignment.ExpiresAtIsNil(),
				tenantaccountassignment.ExpiresAtGT(now),
			),
		).
		Exist(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !hasAssignment {
		return nil, nil, webDAVStatusError{status: http.StatusForbidden, msg: "tenant assignment not active"}
	}

	tenantDB, err := qq.webDAVTenantDB(ctx, tenantx)
	if err != nil {
		return nil, nil, err
	}
	tenantTx, err := tenantDB.Tx(ctx, isReadOnly)
	if err != nil {
		return nil, nil, err
	}
	if beforeTenantAuth != nil {
		if err := beforeTenantAuth(ctx, tenantTx); err != nil {
			_ = tenantTx.Rollback()
			return nil, nil, err
		}
	}

	visitorCtx := ctxx.NewVisitorContext(
		ctx,
		mainTx,
		qq.i18n,
		"",
		"",
		false,
		false,
		qq.infra.SystemConfig().CommercialLicenseEnabled(),
	)
	mainCtx := ctxx.NewMainContext(visitorCtx, accountx, qq.i18n, qq.mainDB, qq.tenantDBs, isReadOnly)
	userx, err := tenantTx.User.Query().
		Where(user.AccountID(accountx.ID), user.DeletedAtIsNil()).
		Only(mainCtx)
	if err != nil {
		_ = tenantTx.Rollback()
		return nil, nil, err
	}
	tenantCtx := ctxx.NewTenantContextWithUser(mainCtx, tenantTx, tenantx, userx, isReadOnly)
	spacex, err := tenantTx.Space.Query().
		Where(space.PublicID(entx.NewCIText(spacePublicID))).
		Only(tenantCtx)
	if err != nil {
		_ = tenantTx.Rollback()
		return nil, nil, err
	}

	return ctxx.NewSpaceContext(tenantCtx, spacex), tenantTx, nil
}

func (qq *Handler) webDAVTenantDB(ctx context.Context, tenantx *entmain.Tenant) (*sqlx.TenantDB, error) {
	tenantDB, ok := qq.tenantDBs.Load(tenantx.ID)
	if ok {
		return tenantDB, nil
	}
	tenantm := tenant2.NewTenant(tenantx)
	tenantDB, err := tenantm.OpenDB(qq.devMode, qq.metaPath)
	if err != nil {
		return nil, err
	}
	qq.tenantDBs.Store(tenantx.ID, tenantDB)
	return tenantDB, nil
}

func (qq *Handler) withFinalizationContexts(
	ctx context.Context,
	credentialx *credentialmodel.AuthRecord,
	tenantPublicID string,
	spacePublicID string,
	resourceID int64,
	fn func(*ctxx.SpaceContext) error,
) error {
	mainTx, err := qq.mainDB.Tx(ctx, false)
	if err != nil {
		return err
	}
	committedMain := false
	defer rollbackMainTx(mainTx, &committedMain)

	ctx = mainprivacy.DecisionContext(ctx, mainprivacy.Allow)
	touched, err := mainTx.WebDAVCredential.Update().
		Where(
			entmainwebdavcredential.ID(credentialx.ID),
			entmainwebdavcredential.RevokedAtIsNil(),
		).
		SetLastUsedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return err
	}
	if touched != 1 {
		return webDAVStatusError{status: http.StatusForbidden, msg: "credential inactive"}
	}

	spaceCtx, tenantTx, err := qq.webDAVSpaceContext(
		ctx,
		mainTx,
		credentialx,
		tenantPublicID,
		spacePublicID,
		false,
		func(ctx context.Context, tenantTx *enttenant.Tx) error {
			ctxWithIncomplete := tenantprivacy.DecisionContext(
				enttenantschema.WithUnfinishedUploads(ctx),
				tenantprivacy.Allow,
			)
			touched, err := tenantTx.WebDAVResource.Update().
				Where(
					enttenantwebdavresource.ID(resourceID),
					enttenantwebdavresource.CredentialPublicID(entx.NewCIText(credentialx.PublicID)),
					enttenantwebdavresource.StateEQ(webdavresourcemodel.Uploading),
				).
				SetLastProgressAt(time.Now()).
				Save(ctxWithIncomplete)
			if err != nil {
				return err
			}
			if touched != 1 {
				return webDAVStatusError{status: http.StatusConflict, msg: "reservation inactive"}
			}
			return nil
		},
	)
	if err != nil {
		return err
	}
	committedTenant := false
	defer rollbackTenantTx(tenantTx, &committedTenant)

	ctxWithIncomplete := tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(ctx),
		tenantprivacy.Allow,
	)
	active, err := tenantTx.WebDAVResource.Query().
		Where(
			enttenantwebdavresource.ID(resourceID),
			enttenantwebdavresource.CredentialPublicID(entx.NewCIText(credentialx.PublicID)),
			enttenantwebdavresource.SpaceID(spaceCtx.Space.ID),
			enttenantwebdavresource.StateEQ(webdavresourcemodel.Uploading),
		).
		Exist(ctxWithIncomplete)
	if err != nil {
		return err
	}
	if !active {
		return webDAVStatusError{status: http.StatusConflict, msg: "reservation inactive"}
	}

	if err := fn(spaceCtx); err != nil {
		return err
	}
	// Commit the independent credential update first so a main DB failure cannot
	// leave a finalized tenant row whose temporary object is then cleaned up.
	if err := mainTx.Commit(); err != nil {
		return err
	}
	committedMain = true
	if err := tenantTx.Commit(); err != nil {
		return err
	}
	committedTenant = true
	return nil
}

func (qq *Handler) cleanupFailedWebDAVUpload(
	ctx context.Context,
	credentialx *credentialmodel.AuthRecord,
	spacePublicID string,
	resourceID int64,
	storedFileID int64,
) error {
	tenantTx, cleanupCtx, err := qq.webDAVCleanupTenantTx(ctx, credentialx)
	if err != nil {
		return err
	}
	committed := false
	defer rollbackTenantTx(tenantTx, &committed)

	deleted, err := tenantTx.WebDAVResource.Delete().
		Where(webDAVUploadResourceScope(credentialx, spacePublicID, resourceID, storedFileID)...).
		Exec(cleanupCtx)
	if err != nil {
		return err
	}
	if deleted == 1 {
		if err := tenantTx.StoredFile.DeleteOneID(storedFileID).Exec(cleanupCtx); err != nil && !enttenant.IsNotFound(err) {
			log.Println(err)
		}
	}
	if err := tenantTx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (qq *Handler) markWebDAVUploadCleanupPending(
	ctx context.Context,
	credentialx *credentialmodel.AuthRecord,
	spacePublicID string,
	resourceID int64,
	storedFileID int64,
) error {
	tenantTx, cleanupCtx, err := qq.webDAVCleanupTenantTx(ctx, credentialx)
	if err != nil {
		return err
	}
	committed := false
	defer rollbackTenantTx(tenantTx, &committed)

	_, err = tenantTx.WebDAVResource.Update().
		Where(webDAVUploadResourceScope(credentialx, spacePublicID, resourceID, storedFileID)...).
		SetState(webdavresourcemodel.CleanupPending).
		SetLastProgressAt(time.Now()).
		Save(cleanupCtx)
	if err != nil {
		return err
	}
	if err := tenantTx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (qq *Handler) webDAVCleanupTenantTx(
	ctx context.Context,
	credentialx *credentialmodel.AuthRecord,
) (*enttenant.Tx, context.Context, error) {
	if credentialx == nil {
		return nil, nil, errors.New("missing WebDAV credential")
	}
	tenantDB, ok := qq.tenantDBs.Load(credentialx.TenantID)
	if !ok {
		if qq.mainDB == nil {
			return nil, nil, errors.New("tenant db missing")
		}
		mainCtx := mainprivacy.DecisionContext(ctx, mainprivacy.Allow)
		tenantx, err := qq.mainDB.ReadOnlyConn.Tenant.Get(mainCtx, credentialx.TenantID)
		if err != nil {
			return nil, nil, err
		}
		tenantDB, err = qq.webDAVTenantDB(mainCtx, tenantx)
		if err != nil {
			return nil, nil, err
		}
	}
	cleanupCtx := tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(ctx),
		tenantprivacy.Allow,
	)
	tenantTx, err := tenantDB.Tx(cleanupCtx, false)
	if err != nil {
		return nil, nil, err
	}
	return tenantTx, cleanupCtx, nil
}

func webDAVUploadResourceScope(
	credentialx *credentialmodel.AuthRecord,
	spacePublicID string,
	resourceID int64,
	storedFileID int64,
) []predicate.WebDAVResource {
	return []predicate.WebDAVResource{
		enttenantwebdavresource.ID(resourceID),
		enttenantwebdavresource.CredentialPublicID(entx.NewCIText(credentialx.PublicID)),
		enttenantwebdavresource.StoredFileID(storedFileID),
		enttenantwebdavresource.StateIn(webdavresourcemodel.Uploading, webdavresourcemodel.CleanupPending),
		enttenantwebdavresource.HasSpaceWith(space.PublicIDEQ(entx.NewCIText(spacePublicID))),
	}
}

func (qq *Handler) preflightWebDAVRequest(req *http.Request, endpointPrefix string, pathx webDAVPath) error {
	switch req.Method {
	case "PROPFIND", "LOCK":
		req.Body = http.MaxBytesReader(nil, req.Body, webDAVMaxXMLBytes)
	case "PUT":
		if !pathx.isFile {
			return webDAVStatusError{status: http.StatusMethodNotAllowed, msg: "put requires file"}
		}
		if req.ContentLength == 0 {
			return webDAVStatusError{status: http.StatusBadRequest, msg: "empty upload"}
		}
		if maxBytes := qq.infra.SystemConfig().MaxUploadSizeBytes(); maxBytes > 0 && req.ContentLength > maxBytes {
			return webDAVStatusError{status: http.StatusRequestEntityTooLarge, msg: "upload too large"}
		}
	case "MOVE":
		if !pathx.isFile {
			return webDAVStatusError{status: http.StatusMethodNotAllowed, msg: "move requires file"}
		}
		destinationPath, ok := parseWebDAVDestination(req, endpointPrefix)
		if !ok || !destinationPath.isFile {
			return webDAVStatusError{status: http.StatusConflict, msg: "invalid destination"}
		}
	}
	return nil
}

func webDAVFromContext(ctx context.Context) (*webDAVRequestContext, bool) {
	requestCtx, ok := ctx.Value(webDAVContextKey{}).(*webDAVRequestContext)
	return requestCtx, ok
}

func writeWebDAVChallenge(rw http.ResponseWriter) {
	rw.Header().Set("WWW-Authenticate", webDAVRealm)
	writeWebDAVText(rw, http.StatusUnauthorized, "Unauthorized")
}

func writeWebDAVOptions(rw http.ResponseWriter) {
	rw.Header().Set("DAV", "1, 2")
	rw.Header().Set("MS-Author-Via", "DAV")
	rw.Header().Set("Allow", webDAVAllow)
	rw.WriteHeader(http.StatusOK)
}

func writeWebDAVMethodNotAllowed(rw http.ResponseWriter) {
	rw.Header().Set("Allow", webDAVAllow)
	writeWebDAVText(rw, http.StatusMethodNotAllowed, "Method not allowed")
}

func writeWebDAVText(rw http.ResponseWriter, status int, msg string) {
	if status == http.StatusMethodNotAllowed {
		rw.Header().Set("Allow", webDAVAllow)
	}
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(status)
	_, _ = io.WriteString(rw, msg)
}

func webDAVHTTPStatus(err error) int {
	var statusErr webDAVStatusError
	if errors.As(err, &statusErr) {
		return statusErr.status
	}
	var httpErr *e.HTTPError
	if errors.As(err, &httpErr) {
		if isTenantQuotaError(httpErr) {
			return http.StatusInsufficientStorage
		}
		if httpErr.StatusCode() == http.StatusRequestEntityTooLarge {
			return http.StatusRequestEntityTooLarge
		}
		if httpErr.StatusCode() == http.StatusInsufficientStorage {
			return http.StatusInsufficientStorage
		}
		if httpErr.StatusCode() >= 400 && httpErr.StatusCode() < 500 {
			return httpErr.StatusCode()
		}
	}
	return http.StatusInternalServerError
}

func isTenantQuotaError(err *e.HTTPError) bool {
	return err.StatusCode() == http.StatusRequestEntityTooLarge &&
		strings.HasPrefix(err.Message(), "Storage limit reached for this organization.")
}

func webDAVInvalidPathStatus(method string) int {
	if method == "PUT" || method == "MOVE" || method == "LOCK" || method == "UNLOCK" {
		return http.StatusConflict
	}
	return http.StatusNotFound
}

func webDAVMethodAllowed(method string) bool {
	switch method {
	case http.MethodOptions, "PROPFIND", "PUT", "LOCK", "UNLOCK", "MOVE":
		return true
	default:
		return false
	}
}

func (qq *Handler) isSecureWebDAVRequest(req *http.Request) bool {
	if qq.devMode || req.TLS != nil {
		return true
	}
	if !strings.EqualFold(req.Header.Get("X-Forwarded-Proto"), "https") {
		return false
	}
	return qq.isTrustedProxy(req.RemoteAddr)
}

func (qq *Handler) webDAVRateLimitRemoteAddr(req *http.Request) string {
	if !qq.isTrustedProxy(req.RemoteAddr) {
		return req.RemoteAddr
	}
	forwardedFor := strings.Split(req.Header.Get("X-Forwarded-For"), ",")
	for _, f := range slices.Backward(forwardedFor) {
		addr, err := netip.ParseAddr(strings.TrimSpace(f))
		if err != nil {
			return req.RemoteAddr
		}
		if !qq.isTrustedProxyAddr(addr) {
			return addr.String()
		}
	}
	return req.RemoteAddr
}

func (qq *Handler) isTrustedProxy(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return false
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	return qq.isTrustedProxyAddr(addr)
}

func (qq *Handler) isTrustedProxyAddr(remoteAddr netip.Addr) bool {
	for _, prefix := range qq.trustedProxies {
		if prefix.Contains(remoteAddr) {
			return true
		}
	}
	return false
}

func rollbackMainTx(tx *entmain.Tx, committed *bool) {
	if !*committed {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Println(err)
		}
	}
}

func rollbackTenantTx(tx *enttenant.Tx, committed *bool) {
	if !*committed {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Println(err)
		}
	}
}

func webDAVMappedHTTPStatus(err error) (int, bool) {
	var statusErr webDAVStatusError
	if errors.As(err, &statusErr) {
		return statusErr.status, true
	}
	var httpErr *e.HTTPError
	if errors.As(err, &httpErr) {
		return webDAVHTTPStatus(err), true
	}
	return 0, false
}
