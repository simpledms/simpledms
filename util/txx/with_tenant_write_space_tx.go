package txx

import (
	"log"
	"net/http"

	"github.com/marcobeierer/go-core/util/e"
	"github.com/simpledms/simpledms/ctxx"
)

func WithTenantWriteSpaceTx[T any](ctx *ctxx.SpaceContext, fn func(ctx2 *ctxx.SpaceContext) (T, error)) (T, error) {
	var zero T
	if ctx != nil && !ctx.TenantCtx().IsReadOnlyTx() {
		return fn(ctx)
	}

	tenantDB, ok := ctx.UnsafeTenantDB(ctx.TenantCtx().Tenant.ID)
	if !ok {
		log.Println("tenant db not found", ctx.TenantCtx().Tenant.ID)
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Tenant database not found.")
	}

	writeTx, err := tenantDB.Tx(ctx, false)
	if err != nil {
		log.Println(err)
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not start transaction.")
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

	writeTenantCtx := ctxx.NewAppContext(
		ctx.TenantCtx(),
		writeTx,
		false,
		ctx.AppCtx().UnsafeTenantDBs(),
	)
	writeSpace := writeTx.Space.GetX(writeTenantCtx, ctx.SpaceCtx().Space.ID)
	writeSpaceCtx := ctxx.NewSpaceContext(writeTenantCtx, writeSpace)

	result, err := fn(writeSpaceCtx)
	if err != nil {
		return zero, err
	}

	if err := writeTx.Commit(); err != nil {
		log.Println(err)
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not save file.")
	}
	committed = true

	return result, nil
}
