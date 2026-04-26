package server

import (
	"log"

	"github.com/marcobeierer/structs"

	corectxx "github.com/marcobeierer/go-core/ctxx"
	coreserver "github.com/marcobeierer/go-core/server"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type simpleHandlerFn func(httpx2.ResponseWriter, *httpx2.Request, ctxx.Context) error

type simpleActionable interface {
	Route() string
	Endpoint() string
	FormRoute() string
	IsReadOnly() bool
	UseManualTxManagement() bool
	AllowInSetupSession() bool
	Handler(httpx2.ResponseWriter, *httpx2.Request, ctxx.Context) error
}

type simpleFormActionable interface {
	FormHandler(httpx2.ResponseWriter, *httpx2.Request, ctxx.Context) error
}

type actionAdapter struct {
	action simpleActionable
}

type actionWithFormAdapter struct {
	*actionAdapter
	formHandler func(httpx2.ResponseWriter, *httpx2.Request, corectxx.Context) error
}

func RegisterActions(router *coreserver.Router, actions any) {
	for _, field := range structs.Fields(actions) {
		value := field.Value()
		if actionx, ok := value.(coreserver.Actionable); ok {
			router.RegisterAction(actionx)
			log.Println("Registered action", field.Name())
		} else if actionx, ok := value.(simpleActionable); ok {
			router.RegisterAction(adaptAction(actionx))
			log.Println("Registered action", field.Name())
		} else if field.Tag("actions") != "" {
			RegisterActions(router, value)
		} else {
			// could for example be ListDir, which is just used as partial
			// TODO should that be refactored?
			log.Println("Field is not Actionable", field.Name())
		}
	}
}

func AdaptHandler(handler simpleHandlerFn) func(httpx2.ResponseWriter, *httpx2.Request, corectxx.Context) error {
	return func(rw httpx2.ResponseWriter, req *httpx2.Request, ctx corectxx.Context) error {
		return handler(rw, req, ctxx.WrapContext(ctx))
	}
}

func adaptAction(action simpleActionable) coreserver.Actionable {
	actionx := &actionAdapter{
		action: action,
	}

	formHandler := adaptFormHandler(action)
	if formHandler == nil {
		return actionx
	}

	return &actionWithFormAdapter{
		actionAdapter: actionx,
		formHandler:   formHandler,
	}
}

func adaptFormHandler(
	action simpleActionable,
) func(httpx2.ResponseWriter, *httpx2.Request, corectxx.Context) error {
	if action.FormRoute() == "" {
		return nil
	}

	if actionWithSimpleForm, ok := action.(simpleFormActionable); ok {
		return AdaptHandler(actionWithSimpleForm.FormHandler)
	}

	if actionWithCoreForm, ok := action.(coreserver.FormActionable); ok {
		return actionWithCoreForm.FormHandler
	}

	return nil
}

func (qq *actionAdapter) Route() string {
	return qq.action.Route()
}

func (qq *actionAdapter) Endpoint() string {
	return qq.action.Endpoint()
}

func (qq *actionAdapter) FormRoute() string {
	return qq.action.FormRoute()
}

func (qq *actionAdapter) IsReadOnly() bool {
	return qq.action.IsReadOnly()
}

func (qq *actionAdapter) UseManualTxManagement() bool {
	return qq.action.UseManualTxManagement()
}

func (qq *actionAdapter) AllowInSetupSession() bool {
	return qq.action.AllowInSetupSession()
}

func (qq *actionAdapter) Handler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx corectxx.Context,
) error {
	return AdaptHandler(qq.action.Handler)(rw, req, ctx)
}

func (qq *actionWithFormAdapter) FormHandler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx corectxx.Context,
) error {
	return qq.formHandler(rw, req, ctx)
}
