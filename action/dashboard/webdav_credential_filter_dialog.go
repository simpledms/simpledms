package dashboard

import (
	"html/template"

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
	showActive, showRevoked, _ := state.statusFilter()
	return &widget.Dialog{
		Widget: widget.Widget[widget.Dialog]{
			ID: qq.ID(),
		},
		Headline:     widget.T("Filter WebDAV credentials"),
		IsOpenOnLoad: true,
		Layout:       widget.DialogLayoutSideSheet,
		Child: &widget.Container{
			Widget: widget.Widget[widget.Container]{
				ID: "webDAVCredentialStatusFilter",
			},
			Child: []*widget.FilterChip{
				qq.statusChip(widget.T("Active"), webDAVCredentialStatusActive, showActive),
				qq.statusChip(widget.T("Revoked"), webDAVCredentialStatusRevoked, showRevoked),
			},
		},
	}
}

func (qq *WebDAVCredentialFilterDialog) statusChip(
	label *widget.Text,
	status string,
	isChecked bool,
) *widget.FilterChip {
	return &widget.FilterChip{
		Type:      widget.FilterChipTypeCheckbox,
		Label:     label,
		Name:      "CredentialStatusValues",
		Value:     status,
		IsChecked: isChecked,
		HTMXAttrs: widget.HTMXAttrs{
			HxOn: &widget.HxOn{
				Event: "change",
				Handler: template.JS(
					"if (event.target.checked) { " +
						"if (!new URLSearchParams(window.location.search).has('credential_status')) { " +
						"_appendQueryParamSliceValue('credential_status', '" +
						webDAVCredentialStatusActive + "'); } " +
						"_appendQueryParamSliceValue('credential_status', '" + status + "'); " +
						"} else { _deleteQueryParamSliceValue('credential_status', '" + status + "'); } " +
						"this.dispatchEvent(new CustomEvent('" +
						event.WebDAVCredentialFilterChanged.String() + "', { bubbles: true }))",
				),
			},
		},
	}
}

func (qq *WebDAVCredentialFilterDialog) ID() string {
	return "webDAVCredentialFilterDialog"
}
