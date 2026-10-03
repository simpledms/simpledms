package dashboard

import (
	"html/template"
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/mcpcredential"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type MCPCredentialListPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewMCPCredentialListPartial(
	infra *common.Infra,
	actions *Actions,
) *MCPCredentialListPartial {
	return &MCPCredentialListPartial{
		infra:   infra,
		actions: actions,
		Config:  actionx.NewConfig(actions.Route("mcp-credential-list-partial"), true),
	}
}

func (qq *MCPCredentialListPartial) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[MCPCredentialListPartialData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[MCPCredentialListPartialData](rw, req)
	data.CredentialStatusValues = state.CredentialStatusValues
	if data.CreatedDestination != "" {
		data.Destination, err = webDAVCredentialDestinationKeyByValue(ctx, data.CreatedDestination)
		if err != nil {
			return err
		}
		data.CreatedDestination = ""
	}
	overview, err := qq.Widget(ctx, req, data)
	if err != nil {
		log.Println(err)
		return err
	}
	rw.Header().Set("Cache-Control", "no-store")
	return qq.infra.Renderer().Render(rw, ctx, &widget.View{
		Children: []widget.IWidget{
			overview,
			// keeps the filter indicator in the app bar in sync with the applied filter
			qq.actions.MCPCredentialsPage.filterButton(data, true),
		},
	})
}

func (qq *MCPCredentialListPartial) Widget(
	ctx ctxx.Context,
	req *httpx.Request,
	data *MCPCredentialListPartialData,
) (*widget.Container, error) {
	if ctx.VisitorCtx().IsTemporarySession {
		return nil, e.NewHTTPErrorf(http.StatusForbidden, "A full Session is required.")
	}
	showActive, showRevoked, err := data.statusFilter()
	if err != nil {
		return nil, err
	}
	query := ctx.MainCtx().MainTx.MCPCredential.Query().
		Where(mcpcredential.AccountID(ctx.MainCtx().Account.ID)).
		Order(mcpcredential.ByCreatedAt())
	if showActive && !showRevoked {
		query.Where(mcpcredential.RevokedAtIsNil())
	}
	if showRevoked && !showActive {
		query.Where(mcpcredential.RevokedAtNotNil())
	}
	credentials, err := query.All(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var content widget.IWidget = &widget.EmptyState{
		Icon:        widget.NewIcon("smart_toy"),
		Headline:    widget.T("No MCP credentials"),
		Description: widget.T("Create a credential to connect an MCP client to a Space."),
		Actions: []widget.IWidget{
			&widget.Button{
				Icon:  widget.NewIcon("add"),
				Label: widget.T("Create MCP credential"),
				HTMXAttrs: qq.actions.CreateMCPCredentialCmd.ModalLinkAttrs(
					qq.actions.CreateMCPCredentialCmd.Data(""),
				),
			},
		},
	}
	if len(credentials) > 0 {
		destinations, err := webDAVCredentialDestinations(ctx)
		if err != nil {
			return nil, err
		}
		content = qq.tabbedCredentials(ctx, req, data, credentials, destinations)
	}

	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: qq.id(),
		},
		MaxHeight: true,
		Child:     content,
		HTMXAttrs: widget.HTMXAttrs{
			HxTrigger: event.HxTrigger(
				event.MCPCredentialChanged,
				event.MCPCredentialFilterChanged,
			),
			HxPost: qq.Endpoint(),
			// hx-vals can also be evaluated by nested form submissions; guard the event.
			HxVals: template.JS("js:{..." + string(util.JSON(data)) +
				",CreatedDestination:(event && event.type === 'mcpCredentialChanged' && " +
				"event.detail) ? (event.detail.destination || '') : ''}"),
			HxTarget: "#" + qq.id(),
			HxSwap:   "outerHTML",
		},
	}, nil
}

func (qq *MCPCredentialListPartial) Data(
	destination string,
	credentialStatusValues ...string,
) *MCPCredentialListPartialData {
	return &MCPCredentialListPartialData{
		Destination:            destination,
		CredentialStatusValues: credentialStatusValues,
	}
}

func (qq *MCPCredentialListPartial) tabbedCredentials(
	ctx ctxx.Context,
	req *httpx.Request,
	data *MCPCredentialListPartialData,
	credentials []*entmain.MCPCredential,
	destinations []*webDAVCredentialDestination,
) *widget.TabBar {
	credentialsByDestination := make(map[string][]*entmain.MCPCredential)
	usedKeys := make(map[string]bool)
	for _, credentialx := range credentials {
		key := webDAVCredentialDestinationKey(
			credentialx.TenantID,
			credentialx.SpacePublicID.String(),
		)
		credentialsByDestination[key] = append(credentialsByDestination[key], credentialx)
		usedKeys[key] = true
	}

	tabs := newCredentialDestinationTabs(ctx, destinations, usedKeys)
	activeKey := tabs.ActiveKey(data.Destination)
	destination, _ := tabs.Destination(activeKey)
	return tabs.TabBar(
		activeKey,
		func(key string) widget.HTMXAttrs {
			return widget.HTMXAttrs{
				HxPost:   qq.Endpoint(),
				HxVals:   util.JSON(qq.Data(key)),
				HxTarget: "#" + qq.id(),
				HxSwap:   "outerHTML",
			}
		},
		qq.destinationContent(ctx, req, credentialsByDestination[activeKey], destination),
	)
}

func (qq *MCPCredentialListPartial) destinationContent(
	ctx ctxx.Context,
	req *httpx.Request,
	credentials []*entmain.MCPCredential,
	destination *webDAVCredentialDestination,
) *widget.ScrollableContent {
	// The MCP URL is the same for all Spaces; the token selects the Space.
	url := mcpCredentialURL(qq.infra.SystemConfig(), req)
	toolbar := widget.NewToolbar(
		widget.T("MCP URL").String(ctx),
		&widget.Row{
			// Same icon-to-text gap as the app bar, so the URL aligns with the page title.
			GapXSize: widget.Gap4,
			Children: []widget.IWidget{
				widget.NewIcon("link"),
				&widget.Link{
					Href:              "#",
					Child:             widget.Tu(url).SetWrap(),
					CopyValue:         url,
					CopyTooltip:       widget.T("Copy MCP URL"),
					CopiedMessage:     widget.T("MCP URL copied to clipboard."),
					CopyFailedMessage: widget.T("Could not copy MCP URL."),
				},
			},
		},
	)

	items := make([]*widget.ListItem, 0, len(credentials)+1)
	if destination != nil {
		items = append(items, &widget.ListItem{
			Leading:  widget.NewIcon("add"),
			Headline: widget.T("Create MCP credential"),
			Type:     widget.ListItemTypeHelper,
			HTMXAttrs: qq.actions.CreateMCPCredentialCmd.ModalLinkAttrs(
				qq.actions.CreateMCPCredentialCmd.Data(destination.value()),
			),
		})
	}
	for _, credentialx := range credentials {
		items = append(items, qq.credentialListItem(ctx, credentialx))
	}
	return &widget.ScrollableContent{
		PaddingX: true,
		Children: &widget.Column{
			GapYSize:         widget.Gap3,
			NoOverflowHidden: true,
			AutoHeight:       true,
			Children:         &widget.List{Children: items},
		},
		Toolbar: toolbar,
	}
}

func (qq *MCPCredentialListPartial) credentialListItem(
	ctx ctxx.Context,
	credentialx *entmain.MCPCredential,
) *widget.ListItem {
	mode := widget.T("Read-only").String(ctx)
	if !credentialx.IsReadOnly {
		mode = widget.T("Read/write").String(ctx)
	}

	supportingText := widget.Tf(
		"%s · Created: %s",
		mode,
		formatCredentialTime(ctx, credentialx.CreatedAt),
	)
	var trailing widget.IWidget
	if credentialx.RevokedAt != nil {
		supportingText = widget.Tf(
			"%s · Revoked: %s",
			mode,
			formatCredentialTime(ctx, *credentialx.RevokedAt),
		)
	} else {
		trailing = &widget.IconButton{
			Icon:    "more_vert",
			Tooltip: widget.T("Actions"),
			Label:   widget.T("Actions"),
			Children: &widget.Menu{
				Items: []*widget.MenuItem{
					{
						LeadingIcon: "edit",
						Label:       widget.T("Edit"),
						HTMXAttrs: qq.actions.EditMCPCredentialCmd.ModalLinkAttrs(
							qq.actions.EditMCPCredentialCmd.Data(
								credentialx.PublicID.String(),
								credentialx.Label,
							),
							"",
						),
					},
					{IsDivider: true},
					{
						LeadingIcon: "block",
						Label:       widget.T("Revoke"),
						HTMXAttrs: widget.HTMXAttrs{
							HxPost: qq.actions.RevokeMCPCredentialCmd.Endpoint(),
							HxVals: util.JSON(qq.actions.RevokeMCPCredentialCmd.Data(
								credentialx.PublicID.String(),
							)),
							HxConfirm: widget.T("Revoke this MCP credential?").String(ctx),
							HxSwap:    "none",
						},
					},
				},
			},
		}
	}

	return &widget.ListItem{
		Leading:        widget.NewIcon("vpn_key"),
		Headline:       widget.Tu(credentialx.Label),
		SupportingText: supportingText,
		Trailing:       trailing,
	}
}

func (qq *MCPCredentialListPartial) id() string {
	return "mcpCredentials"
}
