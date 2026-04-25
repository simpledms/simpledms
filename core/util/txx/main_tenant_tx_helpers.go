package txx

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/core/db/entmain"

	"github.com/simpledms/simpledms/core/util/e"
	"github.com/simpledms/simpledms/ctxx"
	ctxx2 "github.com/simpledms/simpledms/ctxx"
)

func WithTenantWriteSpaceTx[T any](ctx *ctxx.SpaceContext, fn func(*ctxx.SpaceContext) (T, error)) (T, error) {
	var zero T
	if ctx != nil && !ctx.TenantCtx().IsReadOnlyTx() {
		return fn(ctx)
	}

	tenantDB, ok := ctx.UnsafeTenantDB()
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

	writeTenantCtx := ctxx2.NewTenantContext(ctx.MainCtx(), writeTx, ctx.TenantCtx().Tenant, false)
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

func WithMainWriteTx[T any](ctx ctxx.Context, fn func(*entmain.Tx) (T, error)) (T, error) {
	var zero T
	if ctx != nil && ctx.MainCtx() != nil && !ctx.MainCtx().IsReadOnlyTx() {
		return fn(ctx.MainCtx().MainTx)
	}

	mainDB := ctx.MainCtx().UnsafeMainDB()
	if mainDB == nil {
		log.Println("main db not found")
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not start transaction.")
	}

	writeTx, err := mainDB.Tx(ctx, false)
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

	result, err := fn(writeTx)
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
