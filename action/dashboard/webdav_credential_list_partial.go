package dashboard

import (
	"html/template"
	"log"
	"strings"
	"time"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/webdavcredential"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/timex"
)

type WebDAVCredentialListPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewWebDAVCredentialListPartial(
	infra *common.Infra,
	actions *Actions,
) *WebDAVCredentialListPartial {
	return &WebDAVCredentialListPartial{
		infra:   infra,
		actions: actions,
		Config:  actionx.NewConfig(actions.Route("webdav-credential-list-partial"), true),
	}
}

func (qq *WebDAVCredentialListPartial) Data(
	destination string,
	credentialStatusValues ...string,
) *WebDAVCredentialListPartialData {
	return &WebDAVCredentialListPartialData{
		Destination:            destination,
		CredentialStatusValues: credentialStatusValues,
	}
}

func (qq *WebDAVCredentialListPartial) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[WebDAVCredentialListPartialData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[WebDAVCredentialListPartialData](rw, req)
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
	return qq.infra.Renderer().Render(rw, ctx, &widget.View{
		Children: []widget.IWidget{
			overview,
			// keeps the filter indicator in the app bar in sync with the applied filter
			qq.actions.WebDAVCredentialsPage.filterButton(data, true),
		},
	})
}

func (qq *WebDAVCredentialListPartial) Widget(
	ctx ctxx.Context,
	req *httpx.Request,
	data *WebDAVCredentialListPartialData,
) (*widget.Container, error) {
	showActive, showRevoked, err := data.statusFilter()
	if err != nil {
		return nil, err
	}
	query := ctx.MainCtx().MainTx.WebDAVCredential.Query().
		Where(webdavcredential.AccountID(ctx.MainCtx().Account.ID)).
		Order(webdavcredential.ByCreatedAt())
	if showActive && !showRevoked {
		query.Where(webdavcredential.RevokedAtIsNil())
	}
	if showRevoked && !showActive {
		query.Where(webdavcredential.RevokedAtNotNil())
	}
	credentials, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	destinations, err := webDAVCredentialDestinations(ctx)
	if err != nil {
		return nil, err
	}

	var content widget.IWidget = &widget.EmptyState{
		Icon:        widget.NewIcon("vpn_key"),
		Headline:    widget.T("No WebDAV credentials"),
		Description: widget.T("Create a device credential to upload files to an Inbox over WebDAV."),
		Actions: []widget.IWidget{
			&widget.Button{
				Icon:  widget.NewIcon("add"),
				Label: widget.T("Create WebDAV credential"),
				HTMXAttrs: qq.actions.CreateWebDAVCredentialCmd.ModalLinkAttrs(
					qq.actions.CreateWebDAVCredentialCmd.Data("", ""),
				),
			},
		},
	}
	if len(credentials) > 0 {
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
				event.WebDAVCredentialChanged,
				event.WebDAVCredentialFilterChanged,
			),
			HxPost: qq.Endpoint(),
			// hx-vals can also be evaluated by nested form submissions; guard the event.
			HxVals: template.JS("js:{..." + string(util.JSON(data)) +
				",CreatedDestination:(event && event.type === 'webDAVCredentialChanged' && " +
				"event.detail) ? (event.detail.destination || '') : ''}"),
			HxTarget: "#" + qq.id(),
			HxSwap:   "outerHTML",
		},
	}, nil
}

func (qq *WebDAVCredentialListPartial) tabbedCredentials(
	ctx ctxx.Context,
	req *httpx.Request,
	data *WebDAVCredentialListPartialData,
	credentials []*entmain.WebDAVCredential,
	destinations []*webDAVCredentialDestination,
) *widget.TabBar {
	credentialsByDestination := make(map[string][]*entmain.WebDAVCredential)
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
		qq.destinationContent(
			ctx,
			req,
			credentialsByDestination[activeKey],
			destination,
		),
	)
}

func (qq *WebDAVCredentialListPartial) destinationContent(
	ctx ctxx.Context,
	req *httpx.Request,
	credentials []*entmain.WebDAVCredential,
	destination *webDAVCredentialDestination,
) *widget.ScrollableContent {
	var toolbar *widget.Toolbar
	if destination != nil {
		url := destination.url(qq.infra.SystemConfig(), req)
		toolbar = widget.NewToolbar(
			widget.T("WebDAV URL").String(ctx),
			&widget.Row{
				// Same icon-to-text gap as the app bar, so the URL aligns with the page title.
				GapXSize: widget.Gap4,
				Children: []widget.IWidget{
					widget.NewIcon("link"),
					&widget.Link{
						Href:              "#",
						Child:             widget.Tu(url).SetWrap(),
						CopyValue:         url,
						CopyTooltip:       widget.T("Copy WebDAV URL"),
						CopiedMessage:     widget.T("WebDAV URL copied to clipboard."),
						CopyFailedMessage: widget.T("Could not copy WebDAV URL."),
					},
				},
			},
		)
	}
	items := make([]*widget.ListItem, 0, len(credentials)+1)
	if destination != nil {
		items = append(items, &widget.ListItem{
			Leading:  widget.NewIcon("add"),
			Headline: widget.T("Create WebDAV credential"),
			Type:     widget.ListItemTypeHelper,
			HTMXAttrs: qq.actions.CreateWebDAVCredentialCmd.ModalLinkAttrs(
				qq.actions.CreateWebDAVCredentialCmd.Data(destination.value(), ""),
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

func (qq *WebDAVCredentialListPartial) credentialListItem(
	ctx ctxx.Context,
	credentialx *entmain.WebDAVCredential,
) *widget.ListItem {
	supportingText := widget.Tf(
		"Username: %s · Created: %s",
		credentialx.Username,
		formatCredentialTime(ctx, credentialx.CreatedAt),
	)
	if credentialx.LastUsedAt != nil {
		supportingText = widget.Tf(
			"Username: %s · Last used: %s",
			credentialx.Username,
			formatCredentialTime(ctx, *credentialx.LastUsedAt),
		)
	}
	if credentialx.RevokedAt != nil {
		supportingText = widget.Tf(
			"Username: %s · Revoked: %s",
			credentialx.Username,
			formatCredentialTime(ctx, *credentialx.RevokedAt),
		)
	}

	var trailing widget.IWidget
	if credentialx.RevokedAt == nil {
		trailing = &widget.IconButton{
			Icon:    "more_vert",
			Tooltip: widget.T("Edit"),
			Children: &widget.Menu{
				Items: []*widget.MenuItem{
					{
						LeadingIcon: "edit",
						Label:       widget.T("Edit"),
						HTMXAttrs: qq.actions.EditWebDAVCredentialCmd.ModalLinkAttrs(
							qq.actions.EditWebDAVCredentialCmd.Data(
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
							HxPost: qq.actions.RevokeWebDAVCredentialCmd.Endpoint(),
							HxVals: util.JSON(qq.actions.RevokeWebDAVCredentialCmd.Data(
								credentialx.PublicID.String(),
								0,
							)),
							HxConfirm: widget.T("Revoke this WebDAV credential?").String(ctx),
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

func (qq *WebDAVCredentialListPartial) id() string {
	return "webDAVCredentialOverview"
}

func webDAVCredentialTabID(label string) string {
	return strings.ReplaceAll(strings.ToLower(label), " ", "-")
}

func formatCredentialTime(ctx ctxx.Context, value time.Time) string {
	return timex.NewDateTime(value).String(ctx.MainCtx().LanguageBCP47)
}
