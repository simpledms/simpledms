package dashboard

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/model/main/webdavcredential"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type CreateWebDAVCredentialCmd struct {
	infra       *common.Infra
	tenantDBs   *tenantdbs.TenantDBs
	actions     *Actions
	credentialx *webdavcredential.CredentialService
	*actionx.Config
}

func NewCreateWebDAVCredentialCmd(
	infra *common.Infra,
	tenantDBs *tenantdbs.TenantDBs,
	actions *Actions,
) *CreateWebDAVCredentialCmd {
	config := actionx.NewConfig(actions.Route("create-webdav-credential-cmd"), false).
		SetUsesSeparatedCmd(true)
	return &CreateWebDAVCredentialCmd{
		infra:       infra,
		tenantDBs:   tenantDBs,
		actions:     actions,
		credentialx: webdavcredential.NewCredentialService(),
		Config:      config,
	}
}

func (qq *CreateWebDAVCredentialCmd) Data(
	destination string,
	label string,
) *CreateWebDAVCredentialCmdData {
	return &CreateWebDAVCredentialCmdData{
		Destination: destination,
		Label:       label,
	}
}

func (qq *CreateWebDAVCredentialCmd) ModalLinkAttrs(data *CreateWebDAVCredentialCmdData) widget.HTMXAttrs {
	return widget.HTMXAttrs{
		HxPost:        qq.FormEndpointWithParams(actionx.ResponseWrapperDialog, "closest dialog"),
		HxVals:        util.JSON(data),
		LoadInPopover: true,
	}
}

func (qq *CreateWebDAVCredentialCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[CreateWebDAVCredentialCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	tenantPublicID, spacePublicID, ok := strings.Cut(data.Destination, ":")
	if !ok || tenantPublicID == "" || spacePublicID == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Form validation failed.")
	}

	spaceCtx, err := qq.spaceContext(ctx, tenantPublicID, spacePublicID)
	if err != nil {
		return err
	}
	defer func() {
		if err := spaceCtx.TTx.Rollback(); err != nil {
			log.Println(err)
		}
	}()

	endpointURL := webDAVCredentialURL(
		qq.infra.SystemConfig(),
		req,
		tenantPublicID,
		spacePublicID,
	)
	secretLength := 0
	if data.SecretLength != nil {
		secretLength = *data.SecretLength
	}
	result, err := qq.credentialx.CreateOwnerCredential(
		spaceCtx,
		data.Label,
		endpointURL,
		secretLength,
		data.CompatibilityMode,
	)
	if err != nil {
		return err
	}

	rw.Header().Set("Cache-Control", "no-store")
	rw.Header().Set("HX-Trigger", event.AccountUpdated.String())
	rw.AddRenderables(widget.NewSnackbarf("WebDAV credential created."))
	return qq.infra.Renderer().Render(rw, ctx, qq.secretDialog(ctx, result))
}

func (qq *CreateWebDAVCredentialCmd) FormHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormDataX[CreateWebDAVCredentialCmdData](rw, req, ctx, true)
	if err != nil {
		return err
	}

	destinations, err := webDAVCredentialDestinations(ctx)
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
	secretLength := webdavcredential.DefaultSecretLength
	if data.SecretLength != nil {
		secretLength = *data.SecretLength
	}

	form := &widget.Form{
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxTarget: "closest dialog",
			HxSwap:   "outerHTML",
		},
		Children: []widget.IWidget{
			&widget.TextField{
				Label:        widget.T("Device label"),
				Name:         "Label",
				Type:         "text",
				IsRequired:   true,
				HasAutofocus: true,
				DefaultValue: data.Label,
			},
			&widget.TextField{
				Label:          widget.T("Secret length"),
				SupportingText: widget.T("Reduce the secret length only if your device limits the maximum password length."),
				Name:           "SecretLength",
				Type:           "number",
				Step:           "1",
				Min:            strconv.Itoa(webdavcredential.MinimumSecretLength),
				Max:            strconv.Itoa(webdavcredential.DefaultSecretLength),
				IsRequired:     true,
				DefaultValue:   strconv.Itoa(secretLength),
			},
			&widget.Switch{
				Label:          widget.T("Compatibility mode"),
				SupportingText: widget.T("Uses only letters, numbers, hyphens, and underscores for devices with limited support for special characters."),
				Name:           "CompatibilityMode",
				Value:          "true",
				IsChecked:      data.CompatibilityMode,
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
	return qq.infra.Renderer().Render(rw, ctx, autil.WrapWidget(
		widget.T("Create WebDAV credential"),
		submitLabel,
		form,
		actionx.ResponseWrapper(req.URL.Query().Get("wrapper")),
		widget.DialogLayoutStable,
	))
}

func (qq *CreateWebDAVCredentialCmd) secretDialog(
	ctx ctxx.Context,
	result *webdavcredential.CreateResult,
) *widget.Dialog {
	return &widget.Dialog{
		Headline:     widget.T("WebDAV credential created"),
		SubmitLabel:  nil,
		IsOpenOnLoad: true,
		Layout:       widget.DialogLayoutDefault,
		Child: &widget.Column{
			GapYSize:         widget.Gap3,
			NoOverflowHidden: true,
			AutoHeight:       true,
			Children: []widget.IWidget{
				widget.T("Copy the secret now. It will not be shown again.").SetWrap(),
				credentialValue(ctx, "WebDAV URL", result.URL),
				credentialValue(ctx, "Username", result.Username),
				credentialValue(ctx, "Secret", result.Secret),
			},
		},
	}
}

func (qq *CreateWebDAVCredentialCmd) spaceContext(
	ctx ctxx.Context,
	tenantPublicID string,
	spacePublicID string,
) (*ctxx.SpaceContext, error) {
	mainCtx := ctx.MainCtx()
	spacesByTenant, err := mainCtx.ReadOnlyAccountSpacesByTenant()
	if err != nil {
		return nil, err
	}
	tenantx, spacex, err := findWebDAVCredentialDestination(
		spacesByTenant,
		tenantPublicID,
		spacePublicID,
	)
	if err != nil {
		return nil, err
	}
	tenantDB, ok := qq.tenantDBs.Load(tenantx.ID)
	if !ok {
		return nil, e.NewHTTPErrorf(http.StatusNotFound, "Destination unavailable.")
	}
	tenantTx, err := tenantDB.ReadOnlyConn.Tx(ctx)
	if err != nil {
		return nil, err
	}
	userx, err := tenantTx.User.Query().
		Where(user.AccountID(mainCtx.Account.ID)).
		Only(mainCtx)
	if err != nil {
		_ = tenantTx.Rollback()
		return nil, err
	}
	tenantCtx := ctxx.NewTenantContextWithUser(mainCtx, tenantTx, tenantx, userx, true)
	currentSpace, err := tenantTx.Space.Query().
		Where(space.ID(spacex.ID)).
		Only(tenantCtx)
	if err != nil {
		_ = tenantTx.Rollback()
		return nil, err
	}
	return ctxx.NewSpaceContext(tenantCtx, currentSpace), nil
}

func findWebDAVCredentialDestination(
	spacesByTenant map[*entmain.Tenant][]*enttenant.Space,
	tenantPublicID string,
	spacePublicID string,
) (*entmain.Tenant, *enttenant.Space, error) {
	for tenantx, spaces := range spacesByTenant {
		if tenantx.PublicID.String() != tenantPublicID {
			continue
		}
		for _, spacex := range spaces {
			if spacex.PublicID.String() != spacePublicID {
				continue
			}
			return tenantx, spacex, nil
		}
	}
	return nil, nil, e.NewHTTPErrorf(
		http.StatusForbidden,
		"You cannot create a credential for this Space.",
	)
}

func credentialValue(ctx ctxx.Context, label string, value string) *widget.Column {
	translatedLabel := widget.T(label).String(ctx)
	return &widget.Column{
		GapYSize:         widget.Gap1,
		AutoHeight:       true,
		NoOverflowHidden: true,
		Children: []widget.IWidget{
			&widget.Label{Text: widget.T(label), Type: widget.LabelTypeLg},
			&widget.Link{
				Href:              "#",
				Classes:           "w-full min-w-0 max-w-full",
				Child:             widget.Tu(value).SetWrap(),
				WrapAnywhere:      true,
				CopyValue:         value,
				CopyTooltip:       widget.Tf("Copy %s", translatedLabel),
				CopiedMessage:     widget.Tf("%s copied to clipboard.", translatedLabel),
				CopyFailedMessage: widget.Tf("Could not copy %s.", translatedLabel),
			},
		},
	}
}
