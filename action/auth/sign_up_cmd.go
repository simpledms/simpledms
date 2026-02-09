package auth

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/common/country"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/model/modelmain"
	"github.com/simpledms/simpledms/pluginx"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type SignUpCmdData struct {
	Email                 string            `validate:"required,email" form_attrs:"autofocus"`
	OrganizationName      string            // TODO or CompanyName?
	FirstName             string            `validate:"required"`
	LastName              string            `validate:"required"`
	Country               country.Country   `validate:"required"` // `schema:"default:Switzerland"` // TODO define default
	Language              language.Language `validate:"required"` // `schema:"default:German"` // TODO default based on browser language
	SubscribeToNewsletter bool
	// FIXME
	// AcceptTermsOfService  bool `validate:"required"` // TODO link // TODO or AGBs? or AGBs when purchasing?
	// AcceptPrivacyPolicy   bool `validate:"required"` // TODO link
}

type SignUpCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[SignUpCmdData]
}

func NewSignUpCmd(infra *common.Infra, actions *Actions) *SignUpCmd {
	config := actionx.NewConfig(
		actions.Route("sign-up-cmd"),
		false,
	)
	return &SignUpCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelperX[SignUpCmdData](
			infra,
			config,
			wx.T("Sign up [subject]"),
			wx.T("Sign up"),
		),
	}
}

func (qq *SignUpCmd) Data(
	email, firstName, lastName string,
	country country.Country,
	language language.Language,
	newsletter bool,
) *SignUpCmdData {
	return &SignUpCmdData{
		Email:                 email,
		FirstName:             firstName,
		LastName:              lastName,
		Country:               country,
		Language:              language,
		SubscribeToNewsletter: newsletter,
	}
}

func (qq *SignUpCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	// TODO validate input
	// TODO move to model?

	if !qq.infra.SystemConfig().IsSaaSModeEnabled() {
		// TODO or forbidden
		return e.NewHTTPErrorf(http.StatusBadRequest, "Sign up is disabled.")
	}

	data, err := autil.FormData[SignUpCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	tenantName := data.OrganizationName
	if data.OrganizationName == "" {
		tenantName = data.FirstName + " " + data.LastName
	}

	accountm, err := modelmain.NewSignUpService().SignUp(
		ctx,
		data.Email,
		tenantName,
		data.FirstName,
		data.LastName,
		data.Country,
		data.Language,
		data.SubscribeToNewsletter,
		false,
	)
	if err != nil {
		log.Println(err)
		return err
	}

	tenantx, err := accountm.Data.QueryTenants().Only(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	err = qq.infra.PluginRegistry().EmitSignUp(ctx, pluginx.SignUpEvent{
		AccountID:             accountm.Data.ID,
		AccountPublicID:       accountm.Data.PublicID.String(),
		AccountEmail:          accountm.Data.Email.String(),
		TenantID:              tenantx.ID,
		TenantPublicID:        tenantx.PublicID.String(),
		TenantName:            tenantx.Name,
		SubscribeToNewsletter: data.SubscribeToNewsletter,
	})
	if err != nil {
		log.Println(err)
		return err
	}

	rw.AddRenderables(wx.NewSnackbarf("Registration successful, please check your email for your password."))
	rw.Header().Set("HX-Trigger", event.TenantCreated.String())

	// TODO how to add tenant user? TTx not existing yet; on login or security risk?
	return nil
}
