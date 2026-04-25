package ctxx

import (
	"context"

	corectxx "github.com/marcobeierer/go-core/ctxx"
)

type VisitorContext = corectxx.VisitorContext
type MainContext = corectxx.MainContext
type TenantContext = corectxx.TenantContext
type SpaceContext = corectxx.SpaceContext

// Context is the SimpleDMS request context with main, tenant, and space layers.
type Context interface {
	corectxx.Context
	MainCtx() *MainContext
	TenantCtx() *TenantContext
	SpaceCtx() *SpaceContext

	IsVisitorCtx() bool
	IsMainCtx() bool
	IsTenantCtx() bool
	IsSpaceCtx() bool
}

var NewVisitorContext = corectxx.NewVisitorContext
var NewMainContext = corectxx.NewMainContext
var NewTenantContext = corectxx.NewTenantContext
var NewSpaceContext = corectxx.NewSpaceContext

func VisitorCtx(ctx context.Context) (*VisitorContext, bool) {
	return corectxx.VisitorCtx(ctx)
}

func MainCtx(ctx context.Context) (*MainContext, bool) {
	return corectxx.MainCtx(ctx)
}

func TenantCtx(ctx context.Context) (*TenantContext, bool) {
	return corectxx.TenantCtx(ctx)
}

func SpaceCtx(ctx context.Context) (*SpaceContext, bool) {
	return corectxx.SpaceCtx(ctx)
}

func VisitorCtxX(ctx context.Context) *VisitorContext {
	return corectxx.VisitorCtxX(ctx)
}

func MainCtxX(ctx context.Context) *MainContext {
	return corectxx.MainCtxX(ctx)
}

func TenantCtxX(ctx context.Context) *TenantContext {
	return corectxx.TenantCtxX(ctx)
}

func SpaceCtxX(ctx context.Context) *SpaceContext {
	return corectxx.SpaceCtxX(ctx)
}
