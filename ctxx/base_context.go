package ctxx

import (
	corectxx "github.com/marcobeierer/go-core/ctxx"
)

type BaseContext struct {
	corectxx.Context
}

func WrapContext(ctx corectxx.Context) Context {
	if appCtx, ok := ctx.(Context); ok {
		return appCtx
	}

	return &BaseContext{
		Context: ctx,
	}
}

func (qq *BaseContext) AppCtx() *AppContext {
	panic("context not available")
}

func (qq *BaseContext) IsAppCtx() bool {
	return false
}

func (qq *BaseContext) SpaceCtx() *SpaceContext {
	panic("context not available")
}

func (qq *BaseContext) IsSpaceCtx() bool {
	return false
}
