package txx

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/model/main/tenantaccess"
	"github.com/simpledms/simpledms/util/e"
)

func WithTenantWriteSpaceTx[T any](ctx *ctxx.SpaceContext, fn func(*ctxx.SpaceContext) (T, error)) (T, error) {
	var zero T
	if ctx == nil {
		log.Println("space context not found")
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Space context not found.")
	}
	if !ctx.TenantCtx().IsReadOnlyTx() {
		return fn(ctx)
	}
	return withNewTenantWriteSpaceTx(ctx, fn)
}

func WithFreshAuthorizedTenantWriteSpaceTx[T any](
	ctx *ctxx.SpaceContext,
	fn func(*ctxx.SpaceContext) (T, error),
) (T, error) {
	var zero T
	if ctx == nil {
		log.Println("space context not found")
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Space context not found.")
	}

	// Hold the main write lock through tenant finalization so assignment changes serialize after it.
	mainTx := ctx.MainCtx().MainTx
	ownsMainTx := false
	if ctx.MainCtx().IsReadOnlyTx() {
		mainDB := ctx.MainCtx().UnsafeMainDB()
		if mainDB == nil {
			log.Println("main db not found")
			return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not verify access.")
		}
		var err error
		mainTx, err = mainDB.Tx(ctx, false)
		if err != nil {
			log.Println(err)
			return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not verify access.")
		}
		ownsMainTx = true
	}
	if mainTx == nil {
		log.Println("main transaction not found")
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not verify access.")
	}
	rolledBack := false
	defer func() {
		if !ownsMainTx || rolledBack {
			return
		}
		if err := mainTx.Rollback(); err != nil {
			log.Println(err)
		}
	}()
	hasAccess, authErr := tenantaccess.NewTenantAccessService().HasActiveTenantAssignmentForTenant(
		ctx,
		mainTx,
		ctx.MainCtx().Account.ID,
		ctx.TenantCtx().Tenant.ID,
	)
	if authErr != nil {
		log.Println(authErr)
		return zero, authErr
	}
	if !hasAccess {
		return zero, e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to access this tenant.")
	}

	var result T
	var err error
	if ctx.TenantCtx().IsReadOnlyTx() {
		result, err = withNewTenantWriteSpaceTx(ctx, fn)
	} else {
		result, err = fn(ctx)
	}
	if err != nil {
		return zero, err
	}
	if !ownsMainTx {
		return result, nil
	}
	if err := mainTx.Rollback(); err != nil {
		log.Println(err)
	}
	rolledBack = true
	return result, nil
}

func withNewTenantWriteSpaceTx[T any](
	ctx *ctxx.SpaceContext,
	fn func(*ctxx.SpaceContext) (T, error),
) (T, error) {
	var zero T

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

	userx, err := writeTx.User.Query().Where(
		user.AccountID(ctx.MainCtx().Account.ID),
		user.DeletedAtIsNil(),
	).Only(ctx)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return zero, e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to access this tenant.")
		}
		log.Println(err)
		return zero, err
	}
	writeTenantCtx := ctxx.NewTenantContextWithUser(
		ctx.MainCtx(),
		writeTx,
		ctx.TenantCtx().Tenant,
		userx,
		false,
	)
	writeSpace, err := writeTx.Space.Get(writeTenantCtx, ctx.SpaceCtx().Space.ID)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return zero, e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to access this space.")
		}
		log.Println(err)
		return zero, err
	}
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

func WithTenantReadSpaceTx[T any](ctx *ctxx.SpaceContext, fn func(*ctxx.SpaceContext) (T, error)) (T, error) {
	var zero T
	if ctx != nil && !ctx.TenantCtx().IsReadOnlyTx() {
		return fn(ctx)
	}

	tenantDB, ok := ctx.UnsafeTenantDB()
	if !ok {
		log.Println("tenant db not found", ctx.TenantCtx().Tenant.ID)
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Tenant database not found.")
	}

	readTx, err := tenantDB.Tx(ctx, true)
	if err != nil {
		log.Println(err)
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not start transaction.")
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		if err := readTx.Rollback(); err != nil {
			log.Println(err)
		}
	}()

	readTenantCtx := ctxx.NewTenantContext(ctx.MainCtx(), readTx, ctx.TenantCtx().Tenant, true)
	readSpace := readTx.Space.GetX(readTenantCtx, ctx.SpaceCtx().Space.ID)
	readSpaceCtx := ctxx.NewSpaceContext(readTenantCtx, readSpace)

	result, err := fn(readSpaceCtx)
	if err != nil {
		return zero, err
	}

	if err := readTx.Commit(); err != nil {
		log.Println(err)
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not read data.")
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
