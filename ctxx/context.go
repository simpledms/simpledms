package ctxx

import (
	"context"

	ctxx2 "github.com/simpledms/simpledms/core/ctxx"
)

type Context interface {
	context.Context
	VisitorCtx() *ctxx2.VisitorContext
	MainCtx() *MainContext
	TenantCtx() *ctxx2.TenantContext
	SpaceCtx() *SpaceContext

	IsVisitorCtx() bool
	IsMainCtx() bool
	IsTenantCtx() bool
	IsSpaceCtx() bool
}

// TODO is struct{} better than string value?
var (
	visitorCtxKey = "visitor_ctx"
	mainCtxKey    = "main_ctx"
	tenantCtxKey  = "tenant_ctx"
	spaceCtxKey   = "space_ctx"
)

// Necessary in case ctx gets wrapped with value by another library; in this case, SpaceCtx is
// no longer accessible directly, just via Value() method
// TODO or Space?
func VisitorCtx(ctx context.Context) (*ctxx2.VisitorContext, bool) {
	val, ok := ctx.Value(visitorCtxKey).(*ctxx2.VisitorContext)
	return val, ok
}
func MainCtx(ctx context.Context) (*MainContext, bool) {
	val, ok := ctx.Value(mainCtxKey).(*MainContext)
	return val, ok
}
func TenantCtx(ctx context.Context) (*ctxx2.TenantContext, bool) {
	val, ok := ctx.Value(tenantCtxKey).(*ctxx2.TenantContext)
	return val, ok
}
func SpaceCtx(ctx context.Context) (*SpaceContext, bool) {
	val, ok := ctx.Value(spaceCtxKey).(*SpaceContext)
	return val, ok
}

// TODO or SpaceX
func VisitorCtxX(ctx context.Context) *ctxx2.VisitorContext {
	return ctx.Value(visitorCtxKey).(*ctxx2.VisitorContext)
}
func MainCtxX(ctx context.Context) *MainContext {
	return ctx.Value(mainCtxKey).(*MainContext)
}
func TenantCtxX(ctx context.Context) *ctxx2.TenantContext {
	return ctx.Value(tenantCtxKey).(*ctxx2.TenantContext)
}
func SpaceCtxX(ctx context.Context) *SpaceContext {
	return ctx.Value(spaceCtxKey).(*SpaceContext)
}
