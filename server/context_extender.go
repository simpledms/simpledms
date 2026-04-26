package server

import (
	"errors"
	"log"
	"net/http"

	"github.com/mattn/go-sqlite3"

	corectxx "github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/db/entx"
	tenantmodel "github.com/marcobeierer/go-core/model/tenant"
	coreserver "github.com/marcobeierer/go-core/server"
	"github.com/marcobeierer/go-core/util/e"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
)

type ContextExtender struct {
	tenantDBs *tenantdbs.TenantDBs
	devMode   bool
	metaPath  string
}

func NewContextExtender(
	tenantDBs *tenantdbs.TenantDBs,
	devMode bool,
	metaPath string,
) *ContextExtender {
	return &ContextExtender{
		tenantDBs: tenantDBs,
		devMode:   devMode,
		metaPath:  metaPath,
	}
}

func (qq *ContextExtender) ExtendContext(
	input coreserver.ContextInput,
) (corectxx.Context, []coreserver.TxFinalizer, error) {
	if input.Tenant == nil {
		return input.CurrentCtx, nil, nil
	}
	tenantCtx := input.CurrentCtx.TenantCtx()

	tenantClient, ok := qq.tenantDBs.Load(input.Tenant.ID)
	if !ok {
		// IMPORTANT don't initialize here because this could trigger concurrency issues...
		tenantm := tenantmodel.NewTenant(input.Tenant)

		var err error
		tenantClient, err = tenantm.OpenDB(qq.devMode, qq.metaPath)
		if err != nil {
			log.Println(err)
			return input.CurrentCtx, nil, err
		}

		qq.tenantDBs.Store(input.Tenant.ID, tenantClient)
	}

	tenantTx, err := tenantClient.Tx(input.CurrentCtx, input.IsReadOnly)
	if err != nil {
		log.Println(err)
		return input.CurrentCtx, nil, err
	}
	extraTxs := []coreserver.TxFinalizer{tenantTx}

	appCtx, err := ctxx.NewAppContextWithError(
		tenantCtx,
		tenantTx,
		input.IsReadOnly,
		qq.tenantDBs,
	)
	if err != nil {
		log.Println(err)
		return input.CurrentCtx, extraTxs, err
	}
	if input.SpaceID == "" {
		return appCtx, extraTxs, nil
	}

	spacex, err := tenantTx.Space.
		Query().
		Where(space.PublicID(entx.NewCIText(input.SpaceID))).
		Only(appCtx)
	if err != nil {
		log.Println(err)
		return appCtx, extraTxs, err
	}

	spaceCtx := ctxx.NewSpaceContext(
		appCtx,
		spacex,
	)

	return spaceCtx, extraTxs, nil
}

func (qq *ContextExtender) HTTPError(err error) (*e.HTTPError, bool) {
	var entTenantCErr *enttenant.ConstraintError
	if errors.As(err, &entTenantCErr) {
		var sqlErr *sqlite3.Error
		if errors.As(err, &sqlErr) {
			switch sqlErr.ExtendedCode {
			case sqlite3.ErrConstraintUnique:
				return e.NewHTTPErrorf(
					http.StatusBadRequest,
					"A similar entity already exists.",
				), true
			case sqlite3.ErrConstraintForeignKey:
				return e.NewHTTPErrorf(
					http.StatusBadRequest,
					"Cannot delete an entity still in use.",
				), true
			}
		}

		log.Println(entTenantCErr.Unwrap())
		return e.NewHTTPErrorf(
			http.StatusInternalServerError,
			"A database constraint violation happened. Please contact the support.",
		), true
	}

	var entTValErr *enttenant.ValidationError
	if errors.As(err, &entTValErr) {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusBadRequest, "Data validation failed."), true
	}

	return nil, false
}
