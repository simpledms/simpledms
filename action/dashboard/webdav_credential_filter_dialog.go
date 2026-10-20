package dashboard

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type WebDAVCredentialFilterDialog struct {
	infra *common.Infra
	*actionx.Config
}

func NewWebDAVCredentialFilterDialog(
	infra *common.Infra,
	actions *Actions,
) *WebDAVCredentialFilterDialog {
	return &WebDAVCredentialFilterDialog{
		infra: infra,
		Config: actionx.NewConfig(
			actions.Route("webdav-credential-filter-dialog"),
			true,
		),
	}
}

func (qq *WebDAVCredentialFilterDialog) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	state := autil.StateX[WebDAVCredentialListPartialData](rw, req)
	if _, _, err := state.statusFilter(); err != nil {
		return err
	}
	return qq.infra.Renderer().Render(rw, ctx, qq.Widget(state))
}

func (qq *WebDAVCredentialFilterDialog) Widget(
	state *WebDAVCredentialListPartialData,
) *widget.Dialog {
	return newCredentialStatusFilterDialog(
		qq.ID(),
		widget.T("Filter WebDAV credentials"),
		"webDAVCredentialStatusFilter",
		state.CredentialStatusValues,
		event.WebDAVCredentialFilterChanged,
	)
}

func (qq *WebDAVCredentialFilterDialog) ID() string {
	return "webDAVCredentialFilterDialog"
}
