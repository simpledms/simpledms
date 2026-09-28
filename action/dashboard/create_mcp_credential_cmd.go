package dashboard

import (
	"errors"
	"log"
	"net/http"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/mcpcredential"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type CreateMCPCredentialCmd struct {
	infra   *common.Infra
	service *mcpcredential.CredentialService
	*actionx.Config
}

func NewCreateMCPCredentialCmd(infra *common.Infra, actions *Actions) *CreateMCPCredentialCmd {
	return &CreateMCPCredentialCmd{
		infra:   infra,
		service: mcpcredential.NewCredentialService(),
		Config: actionx.NewConfig(actions.Route("create-mcp-credential-cmd"), false).
			EnableCommittedResponse(),
	}
}

func (qq *CreateMCPCredentialCmd) Handler(
	rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
) error {
	if ctx.VisitorCtx().IsTemporarySession {
		return e.NewHTTPErrorf(http.StatusForbidden, "A full Session is required.")
	}
	data, err := autil.FormData[CreateMCPCredentialCmdData](rw, req, ctx)
	if err != nil {
		return err
	}
	tenantID, spaceID, ok := strings.Cut(data.Destination, ":")
	if !ok {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Form validation failed.")
	}
	token, err := qq.service.Create(ctx.MainCtx(), tenantID, spaceID, data.Label, !data.AllowWrites)
	if err != nil {
		return err
	}
	endpoint := qq.infra.SystemConfig().AbsoluteURL("/mcp")
	if endpoint == "/mcp" {
		scheme := "http"
		if req.TLS != nil || req.URL.Scheme == "https" {
			scheme = "https"
		}
		endpoint = scheme + "://" + req.Host + "/mcp"
	}
	rw.Header().Set("Cache-Control", "no-store")
	rw.Header().Set("HX-Trigger", "mcpCredentialsChanged")
	return qq.infra.Renderer().Render(rw, ctx, &widget.Dialog{
		Headline:     widget.T("MCP credential created"),
		SubmitLabel:  nil,
		IsOpenOnLoad: true,
		Layout:       widget.DialogLayoutDefault,
		Child: &widget.Column{
			GapYSize:         widget.Gap3,
			AutoHeight:       true,
			NoOverflowHidden: true,
			Children: []widget.IWidget{
				widget.T("Copy the secret now. It will not be shown again.").SetWrap(),
				credentialValue(ctx, "MCP URL", endpoint),
				credentialValue(ctx, "Token", token),
			},
		},
	})
}

func (qq *CreateMCPCredentialCmd) FormHandler(
	rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
) error {
	if ctx.VisitorCtx().IsTemporarySession {
		return e.NewHTTPErrorf(http.StatusForbidden, "A full Session is required.")
	}
	data, err := autil.FormDataX[CreateMCPCredentialCmdData](rw, req, ctx, true)
	if err != nil {
		return err
	}
	destinations, err := qq.destinations(ctx)
	if err != nil {
		return err
	}
	var options []*widget.SelectOption
	for _, destination := range destinations {
		options = append(options, &widget.SelectOption{
			Value: destination.value(),
			Label: widget.Tu(destination.label),
		})
	}
	var content renderable.Renderable = &widget.EmptyState{Headline: widget.T("No spaces available yet.")}
	submit := widget.T("Create")
	if len(options) == 0 {
		submit = nil
	} else {
		content = &widget.Form{
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:   qq.Endpoint(),
				HxTarget: "closest dialog",
				HxSwap:   "outerHTML",
			},
			Children: []widget.IWidget{
				&widget.TextField{
					Label:        widget.T("Label"),
					Name:         "Label",
					Type:         "text",
					IsRequired:   true,
					HasAutofocus: true,
					DefaultValue: data.Label,
				},
				&widget.Checkbox{
					Label:     widget.T("Allow writes"),
					Name:      "AllowWrites",
					IsChecked: data.AllowWrites,
				},
				widget.T("This credential can access only the selected Space.").SetWrap(),
				&widget.SelectField{
					Label:        widget.T("Space"),
					Name:         "Destination",
					DefaultValue: data.Destination,
					Options:      options,
					IsRequired:   true,
				},
			},
		}
	}
	rw.Header().Set("Cache-Control", "no-store")
	return qq.infra.Renderer().Render(rw, ctx, autil.WrapWidget(
		widget.T("Create MCP credential"), submit, content,
		actionx.ResponseWrapperDialog, widget.DialogLayoutStable,
	))
}

func (qq *CreateMCPCredentialCmd) destinations(
	ctx ctxx.Context,
) ([]*webDAVCredentialDestination, error) {
	destinations, err := webDAVCredentialDestinations(ctx)
	if err != nil {
		return nil, err
	}
	active := make([]*webDAVCredentialDestination, 0, len(destinations))
	for _, destination := range destinations {
		_, tx, err := qq.service.Scope(ctx.MainCtx(), destination.tenantPublicID, destination.spacePublicID)
		if err != nil {
			var httpErr *e.HTTPError
			if errors.As(err, &httpErr) && (httpErr.StatusCode() == http.StatusForbidden ||
				httpErr.StatusCode() == http.StatusServiceUnavailable) {
				continue
			}
			log.Println(err)
			return nil, err
		}
		if err := tx.Rollback(); err != nil {
			log.Println(err)
			return nil, err
		}
		active = append(active, destination)
	}
	return active, nil
}
