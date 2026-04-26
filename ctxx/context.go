package ctxx

import (
	"context"

	corectxx "github.com/marcobeierer/go-core/ctxx"
)

// Context is the SimpleDMS request context with main, tenant, and space layers.
type Context interface {
	corectxx.Context
	SpaceCtx() *SpaceContext
	AppCtx() *AppContext
	IsSpaceCtx() bool
	IsAppCtx() bool
}

var (
	spaceCtxKey = "space_ctx"
	appCtxKey   = "app_ctx"
)

/*
var NewVisitorContext = corectxx.NewVisitorContext
var NewMainContext = corectxx.NewMainContext
var NewTenantContext = corectxx.NewTenantContext
*/

func AppCtx(ctx context.Context) (*AppContext, bool) {
	val, ok := ctx.Value(appCtxKey).(*AppContext)
	return val, ok
}

func SpaceCtx(ctx context.Context) (*SpaceContext, bool) {
	val, ok := ctx.Value(spaceCtxKey).(*SpaceContext)
	return val, ok
}
