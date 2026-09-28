package dashboard

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/mcpcredential"
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

func NewMCPCredentialListPartial(infra *common.Infra, actions *Actions) *MCPCredentialListPartial {
	return &MCPCredentialListPartial{
		infra:   infra,
		actions: actions,
		Config:  actionx.NewConfig(actions.Route("mcp-credential-list-partial"), true),
	}
}

func (qq *MCPCredentialListPartial) Handler(
	rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
) error {
	data, err := autil.FormData[MCPCredentialListData](rw, req, ctx)
	if err != nil {
		return err
	}
	view, err := qq.Widget(ctx, data.Offset)
	if err != nil {
		return err
	}
	rw.Header().Set("Cache-Control", "no-store")
	return qq.infra.Renderer().Render(rw, ctx, view)
}

func (qq *MCPCredentialListPartial) Widget(ctx ctxx.Context, offset int) (*widget.Container, error) {
	if ctx.VisitorCtx().IsTemporarySession {
		return nil, e.NewHTTPErrorf(http.StatusForbidden, "A full Session is required.")
	}
	credentials, err := ctx.MainCtx().MainTx.MCPCredential.Query().Where(
		mcpcredential.AccountID(ctx.MainCtx().Account.ID),
	).Order(entmain.Desc(mcpcredential.FieldID)).Offset(offset).Limit(51).All(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	hasMore := len(credentials) > 50
	if hasMore {
		credentials = credentials[:50]
	}
	destinations, err := webDAVCredentialDestinations(ctx)
	if err != nil {
		return nil, err
	}
	var items []*widget.ListItem
	for _, credential := range credentials {
		scopeName := credential.SpacePublicID.String()
		for _, destination := range destinations {
			if destination.tenantID == credential.TenantID &&
				destination.spacePublicID == credential.SpacePublicID.String() {
				scopeName = destination.label
				break
			}
		}
		mode := widget.T("Read-only")
		if !credential.IsReadOnly {
			mode = widget.T("Read/write")
		}
		var trailing widget.IWidget = widget.T("Revoked")
		if credential.RevokedAt == nil {
			trailing = &widget.Button{
				Label: widget.T("Revoke"),
				HTMXAttrs: widget.HTMXAttrs{
					Role:   "button",
					HxPost: qq.actions.RevokeMCPCredentialCmd.Endpoint(),
					HxVals: util.JSON(&RevokeMCPCredentialCmdData{
						CredentialID: credential.PublicID.String(),
					}),
					HxSwap: "none",
				},
			}
		}
		items = append(items, &widget.ListItem{
			Headline: widget.Tu(credential.Label),
			SupportingText: widget.Tuf("%s · %s · %s", mode.String(ctx),
				scopeName, credential.CreatedAt.Format("2006-01-02")),
			Trailing: trailing,
		})
	}
	children := []widget.IWidget{&widget.List{Children: items}}
	if len(items) == 0 {
		children = []widget.IWidget{&widget.EmptyState{Headline: widget.T("No MCP credentials")}}
	}
	if offset > 0 {
		children = append(children, qq.pageButton("Previous", offset-50))
	}
	if hasMore {
		children = append(children, qq.pageButton("Next", offset+50))
	}
	return &widget.Container{
		Widget: widget.Widget[widget.Container]{ID: "mcpCredentials"},
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:    qq.Endpoint(),
			HxTrigger: "mcpCredentialsChanged from:body",
			HxTarget:  "#mcpCredentials",
			HxSwap:    "outerHTML",
		},
		GapY:  true,
		Child: children,
	}, nil
}

func (qq *MCPCredentialListPartial) pageButton(label string, offset int) *widget.Button {
	return &widget.Button{
		Label: widget.T(label),
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(&MCPCredentialListData{Offset: offset}),
			Role:     "button",
			HxTarget: "#mcpCredentials",
			HxSwap:   "outerHTML",
		},
	}
}
