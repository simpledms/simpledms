package mcpcredential

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/simpledms/simpledms/common/execution"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entmain/mcpcredential"
	"github.com/simpledms/simpledms/db/entmain/privacy"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/i18n"
	"github.com/simpledms/simpledms/util/e"
)

type CredentialService struct{}

func NewCredentialService() *CredentialService {
	return &CredentialService{}
}

// Create returns the token only to the transaction owner, who must commit before displaying it.
func (qq *CredentialService) Create(
	ctx *ctxx.MainContext, tenantID, spaceID, label string, isReadOnly bool,
) (string, error) {
	label = strings.TrimSpace(label)
	if ctx.IsTemporarySession || !isValidLabel(label) {
		return "", e.NewHTTPErrorf(http.StatusBadRequest, "Form validation failed.")
	}
	spaceCtx, tenantTx, err := qq.Scope(ctx, tenantID, spaceID)
	if err != nil {
		return "", err
	}
	defer rollback(tenantTx)
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		log.Println(err)
		return "", err
	}
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)
	hash := sha256.Sum256([]byte(secret))
	credential, err := ctx.MainTx.MCPCredential.Create().
		SetAccountID(ctx.Account.ID).
		SetTenantID(spaceCtx.Tenant.ID).
		SetSpacePublicID(spaceCtx.Space.PublicID).
		SetLabel(label).
		SetIsReadOnly(isReadOnly).
		SetSecretHash(hex.EncodeToString(hash[:])).
		Save(ctx)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return "sdmcp_" + credential.PublicID.String() + "." + secret, nil
}

func (qq *CredentialService) Revoke(ctx *ctxx.MainContext, publicID string) (bool, error) {
	if ctx.IsTemporarySession {
		return false, e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to access this tenant.")
	}
	credential, err := ctx.MainTx.MCPCredential.Query().Where(
		mcpcredential.PublicID(entx.NewCIText(publicID)),
		mcpcredential.AccountID(ctx.Account.ID),
	).Only(ctx)
	if err != nil {
		log.Println(err)
		if entmain.IsNotFound(err) {
			return false, e.NewHTTPErrorf(http.StatusNotFound, "Credential not found.")
		}
		return false, err
	}
	if credential.RevokedAt != nil {
		return false, nil
	}
	err = credential.Update().SetRevokedAt(time.Now()).Exec(ctx)
	if err != nil {
		log.Println(err)
	}
	return err == nil, err
}

// EditLabel renames an owned credential; scope, mode and secret stay unchanged.
func (qq *CredentialService) EditLabel(
	ctx *ctxx.MainContext,
	publicID string,
	label string,
) error {
	if ctx.IsTemporarySession {
		return e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to access this tenant.")
	}
	label = strings.TrimSpace(label)
	if !isValidLabel(label) {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Form validation failed.")
	}
	credential, err := ctx.MainTx.MCPCredential.Query().Where(
		mcpcredential.PublicID(entx.NewCIText(publicID)),
		mcpcredential.AccountID(ctx.Account.ID),
	).Only(ctx)
	if err != nil {
		log.Println(err)
		if entmain.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusNotFound, "Credential not found.")
		}
		return err
	}
	if err := credential.Update().SetLabel(label).Exec(ctx); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// Scope shares the browser's context construction without inheriting bootstrap privacy bypasses.
func (qq *CredentialService) Scope(
	ctx *ctxx.MainContext, tenantID, spaceID string,
) (*ctxx.SpaceContext, *enttenant.Tx, error) {
	if tenantID == "" || spaceID == "" {
		return nil, nil, e.NewHTTPErrorf(http.StatusForbidden, "Destination unavailable.")
	}
	tenantx, err := ctx.MainTx.Tenant.Query().Where(
		tenant.PublicID(entx.NewCIText(tenantID)), tenant.DeletedAtIsNil(),
	).Only(ctx)
	if err != nil {
		log.Println(err)
		return nil, nil, e.NewHTTPErrorf(http.StatusForbidden, "Destination unavailable.")
	}
	if tenantx.InitializedAt == nil || tenantx.MaintenanceModeEnabledAt != nil {
		return nil, nil, e.NewHTTPErrorf(http.StatusServiceUnavailable, "Destination unavailable.")
	}
	db, ok := ctx.UnsafeTenantDB(tenantx.ID)
	if !ok {
		return nil, nil, e.NewHTTPErrorf(http.StatusServiceUnavailable, "Destination unavailable.")
	}
	tx, err := db.Tx(ctx, true)
	if err != nil {
		log.Println(err)
		return nil, nil, err
	}
	resolved, err := execution.NewScopeResolver().Resolve(ctx, tx, tenantx, spaceID, true)
	if err != nil {
		rollback(tx)
		return nil, nil, err
	}
	return resolved.SpaceCtx(), tx, nil
}

// Execute authenticates afresh and owns the initial bounded read transactions for one operation.
func (qq *CredentialService) Execute(
	ctx context.Context,
	mainDB *sqlx.MainDB,
	dbs *tenantdbs.TenantDBs,
	i18nx *i18n.I18n,
	commercial bool,
	token string,
	fn func(*ctxx.SpaceContext, *entmain.MCPCredential) error,
) (ok bool, err error) {
	mainTx, err := mainDB.Tx(ctx, true)
	if err != nil {
		log.Println(err)
		return false, err
	}
	defer rollback(mainTx)
	defer func() {
		if recovered := recover(); recovered != nil {
			// Legacy Ent *X methods may panic; never include the recovered data in a tool result.
			log.Printf("MCP operation panic: %T", recovered)
			ok = false
			err = errors.New("MCP operation failed")
		}
	}()
	credential, err := qq.authenticate(ctx, mainTx, token)
	if err != nil {
		return false, err
	}
	// Only bootstrap identity lookup bypasses privacy. All subsequent reads use the original context.
	bootstrap := privacy.DecisionContext(ctx, privacy.Allow)
	actor, err := mainTx.Account.Query().Where(
		account.ID(credential.AccountID), account.DeletedAtIsNil(),
	).Only(bootstrap)
	if err != nil {
		log.Println(err)
		return false, e.NewHTTPErrorf(http.StatusForbidden, "Destination unavailable.")
	}
	visitor := ctxx.NewVisitorContext(ctx, mainTx, i18nx, "", "UTC", false, false, commercial)
	mainCtx := ctxx.NewMainContext(visitor, actor, i18nx, mainDB, dbs, true)
	tenantx, err := mainTx.Tenant.Get(mainCtx, credential.TenantID)
	if err != nil {
		log.Println(err)
		return false, e.NewHTTPErrorf(http.StatusForbidden, "Destination unavailable.")
	}
	sc, tenantTx, err := qq.Scope(mainCtx, tenantx.PublicID.String(), credential.SpacePublicID.String())
	if err != nil {
		return false, err
	}
	defer rollback(tenantTx)
	if err := fn(sc, credential); err != nil {
		log.Printf("MCP operation failed: %T", err)
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := tenantTx.Commit(); err != nil {
		log.Println(err)
		return false, err
	}
	if err := mainTx.Commit(); err != nil {
		log.Println(err)
		return false, err
	}
	return true, nil
}

// AuthorizeFinalization rechecks that a credential can write under the fresh main write lock.
func (qq *CredentialService) AuthorizeFinalization(
	ctx context.Context,
	tx *entmain.Tx,
	credential *entmain.MCPCredential,
) error {
	active, err := tx.MCPCredential.Query().Where(
		mcpcredential.ID(credential.ID),
		mcpcredential.AccountID(credential.AccountID),
		mcpcredential.TenantID(credential.TenantID),
		mcpcredential.SpacePublicID(credential.SpacePublicID),
		mcpcredential.IsReadOnly(false),
		mcpcredential.RevokedAtIsNil(),
	).Exist(ctx)
	if err != nil {
		log.Println(err)
		return err
	}
	if !active {
		return e.NewHTTPErrorf(http.StatusForbidden, "MCP credential cannot write.")
	}
	return nil
}

func (qq *CredentialService) authenticate(
	ctx context.Context, tx *entmain.Tx, token string,
) (*entmain.MCPCredential, error) {
	publicID, secret, found := strings.Cut(strings.TrimPrefix(token, "sdmcp_"), ".")
	if !strings.HasPrefix(token, "sdmcp_") || !found || len(secret) != 43 || len(publicID) > 100 {
		return nil, e.NewHTTPErrorf(http.StatusUnauthorized, "Invalid MCP credential.")
	}
	bootstrap := privacy.DecisionContext(ctx, privacy.Allow)
	credential, err := tx.MCPCredential.Query().Where(
		mcpcredential.PublicID(entx.NewCIText(publicID)), mcpcredential.RevokedAtIsNil(),
	).Only(bootstrap)
	if err != nil {
		if !entmain.IsNotFound(err) {
			log.Println(err)
			return nil, err
		}
		return nil, e.NewHTTPErrorf(http.StatusUnauthorized, "Invalid MCP credential.")
	}
	hash := sha256.Sum256([]byte(secret))
	if subtle.ConstantTimeCompare([]byte(hex.EncodeToString(hash[:])), []byte(credential.SecretHash)) != 1 {
		return nil, e.NewHTTPErrorf(http.StatusUnauthorized, "Invalid MCP credential.")
	}
	return credential, nil
}

func rollback(tx interface{ Rollback() error }) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		log.Println(err)
	}
}

func isValidLabel(label string) bool {
	return label != "" && utf8.RuneCountInString(label) <= 100
}
