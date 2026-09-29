package dashboard

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/mcpcredential"
	"github.com/simpledms/simpledms/model/main/systemconfig"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type CreateMCPCredentialCmd struct {
	infra       *common.Infra
	actions     *Actions
	credentialx *mcpcredential.CredentialService
	*actionx.Config
}

func NewCreateMCPCredentialCmd(infra *common.Infra, actions *Actions) *CreateMCPCredentialCmd {
	config := actionx.NewConfig(actions.Route("create-mcp-credential-cmd"), false).
		EnableCommittedResponse()
	return &CreateMCPCredentialCmd{
		infra:       infra,
		actions:     actions,
		credentialx: mcpcredential.NewCredentialService(),
		Config:      config,
	}
}

func (qq *CreateMCPCredentialCmd) Data(destination string) *CreateMCPCredentialCmdData {
	return &CreateMCPCredentialCmdData{
		Destination: destination,
	}
}

func (qq *CreateMCPCredentialCmd) ModalLinkAttrs(
	data *CreateMCPCredentialCmdData,
) widget.HTMXAttrs {
	return widget.HTMXAttrs{
		HxPost:        qq.FormEndpointWithParams(actionx.ResponseWrapperDialog, "closest dialog"),
		HxVals:        util.JSON(data),
		LoadInPopover: true,
	}
}

func (qq *CreateMCPCredentialCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	if ctx.VisitorCtx().IsTemporarySession {
		return e.NewHTTPErrorf(http.StatusForbidden, "A full Session is required.")
	}
	data, err := autil.FormData[CreateMCPCredentialCmdData](rw, req, ctx)
	if err != nil {
		return err
	}
	tenantPublicID, spacePublicID, ok := strings.Cut(data.Destination, ":")
	if !ok {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Form validation failed.")
	}
	token, err := qq.credentialx.Create(
		ctx.MainCtx(),
		tenantPublicID,
		spacePublicID,
		data.Label,
		!data.AllowWrites,
	)
	if err != nil {
		return err
	}

	overview, err := qq.createdCredentialOverview(rw, req, ctx, data.Destination)
	if err != nil {
		return err
	}

	rw.Header().Set("Cache-Control", "no-store")
	rw.AddRenderables(widget.NewSnackbarf("MCP credential created."))
	return qq.infra.Renderer().Render(rw, ctx, &widget.View{
		Children: []widget.IWidget{
			qq.secretDialog(ctx, req, token),
			overview,
		},
	})
}

// createdCredentialOverview refreshes the credential list out-of-band with the tab of the new
// credential's Space selected. AccountUpdated is not triggered instead because HX-Trigger
// events fire before the swap, so its refresh would race and restore the previous tab.
func (qq *CreateMCPCredentialCmd) createdCredentialOverview(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	destinationValue string,
) (*widget.Container, error) {
	destinationKey, err := webDAVCredentialDestinationKeyByValue(ctx, destinationValue)
	if err != nil {
		return nil, err
	}
	state := autil.StateX[MCPCredentialListPartialData](rw, req)
	return qq.actions.MCPCredentialListPartial.WidgetOOB(
		ctx,
		req,
		qq.actions.MCPCredentialListPartial.Data(
			destinationKey,
			state.CredentialStatusValues...,
		),
	)
}

func (qq *CreateMCPCredentialCmd) FormHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
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
	destinationItems := make([]*widget.ListItem, 0, len(destinations))
	for index, destination := range destinations {
		destinationItems = append(destinationItems, &widget.ListItem{
			RadioGroupName: "Destination",
			RadioValue:     destination.value(),
			IsSelected: data.Destination == destination.value() ||
				(data.Destination == "" && index == 0),
			Headline:       widget.Tu(destination.spaceName),
			SupportingText: widget.Tu(destination.tenantName),
			Leading:        widget.NewIcon("folder_open"),
		})
	}
	if len(destinationItems) == 0 {
		destinationItems = append(destinationItems, &widget.ListItem{
			Headline: widget.T("No spaces available yet."),
			Type:     widget.ListItemTypeHelper,
		})
	}

	form := &widget.Form{
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxTarget: "closest dialog",
			HxSwap:   "outerHTML",
		},
		Children: []widget.IWidget{
			&widget.TextField{
				Label:        widget.T("Client label"),
				Name:         "Label",
				Type:         "text",
				IsRequired:   true,
				HasAutofocus: true,
				DefaultValue: data.Label,
			},
			&widget.Switch{
				Label:          widget.T("Allow writes"),
				SupportingText: widget.T("Without writes, the MCP client can only read documents."),
				Name:           "AllowWrites",
				Value:          "true",
				IsChecked:      data.AllowWrites,
				CheckedIcon:    widget.NewIcon("check"),
			},
			&widget.Label{Text: widget.T("Space"), Type: widget.LabelTypeLg},
			&widget.ScrollableContent{
				Children: &widget.List{Children: destinationItems},
			},
		},
	}

	submitLabel := widget.T("Create")
	if len(destinations) == 0 {
		submitLabel = nil
	}
	rw.Header().Set("Cache-Control", "no-store")
	return qq.infra.Renderer().Render(rw, ctx, autil.WrapWidget(
		widget.T("Create MCP credential"),
		submitLabel,
		form,
		actionx.ResponseWrapper(req.URL.Query().Get("wrapper")),
		widget.DialogLayoutStable,
	))
}

func (qq *CreateMCPCredentialCmd) secretDialog(
	ctx ctxx.Context,
	req *httpx.Request,
	token string,
) *widget.Dialog {
	return &widget.Dialog{
		Headline:     widget.T("MCP credential created"),
		SubmitLabel:  nil,
		IsOpenOnLoad: true,
		Layout:       widget.DialogLayoutDefault,
		Child: &widget.Column{
			GapYSize:         widget.Gap3,
			NoOverflowHidden: true,
			AutoHeight:       true,
			Children: []widget.IWidget{
				widget.T("Copy the secret now. It will not be shown again.").SetWrap(),
				credentialValue(ctx, "MCP URL", mcpCredentialURL(qq.infra.SystemConfig(), req)),
				credentialValue(ctx, "Token", token),
			},
		},
	}
}

// destinations lists only Spaces for which a credential can currently be created, for example
// excluding tenants in maintenance mode.
func (qq *CreateMCPCredentialCmd) destinations(
	ctx ctxx.Context,
) ([]*webDAVCredentialDestination, error) {
	destinations, err := webDAVCredentialDestinations(ctx)
	if err != nil {
		return nil, err
	}
	active := make([]*webDAVCredentialDestination, 0, len(destinations))
	for _, destination := range destinations {
		_, tx, err := qq.credentialx.Scope(
			ctx.MainCtx(),
			destination.tenantPublicID,
			destination.spacePublicID,
		)
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

func mcpCredentialURL(config *systemconfig.SystemConfig, req *httpx.Request) string {
	path := "/mcp"
	if absoluteURL := config.AbsoluteURL(path); absoluteURL != path {
		return absoluteURL
	}
	if req == nil || req.Host == "" {
		return path
	}
	scheme := "https"
	if req.TLS == nil {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s%s", scheme, req.Host, path)
}
