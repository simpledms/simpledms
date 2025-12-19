package auth

import (
	"log"
	"time"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/entx"
	"github.com/simpledms/simpledms/model/common/country"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/model/common/mainrole"
	"github.com/simpledms/simpledms/model/common/tenantrole"
	"github.com/simpledms/simpledms/model/mailer"
	"github.com/simpledms/simpledms/model/modelmain"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/timex"
)

type SignUpData struct {
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

type SignUp struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[SignUpData]
}

func NewSignUp(infra *common.Infra, actions *Actions) *SignUp {
	config := actionx.NewConfig(
		actions.Route("sign-up"),
		false,
	)
	return &SignUp{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelperX[SignUpData](
			infra,
			config,
			wx.T("Sign up [subject]"),
			wx.T("Sign up"),
		),
	}
}

func (qq *SignUp) Data(
	email, firstName, lastName string,
	country country.Country,
	language language.Language,
	newsletter bool,
) *SignUpData {
	return &SignUpData{
		Email:                 email,
		FirstName:             firstName,
		LastName:              lastName,
		Country:               country,
		Language:              language,
		SubscribeToNewsletter: newsletter,
	}
}

func (qq *SignUp) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	// TODO validate input
	// TODO move to model?

	data, err := autil.FormData[SignUpData](rw, req, ctx)
	if err != nil {
		return err
	}

	tenantName := data.OrganizationName
	if data.OrganizationName == "" {
		tenantName = data.FirstName + " " + data.LastName
	}

	tenantxQuery := ctx.VisitorCtx().MainTx.Tenant.
		Create().
		SetName(tenantName).
		SetCountry(data.Country)
	/* FIXME
	if data.AcceptPrivacyPolicy {
		tenantxQuery = tenantxQuery.SetPrivacyPolicyAccepted(time.Now())
	}
	if data.AcceptTermsOfService {
		tenantxQuery = tenantxQuery.SetTermsOfServiceAccepted(time.Now())
	}
	*/
	tenantxQuery = tenantxQuery.SetPrivacyPolicyAccepted(timex.NewDateTimeZero().Time)
	tenantxQuery = tenantxQuery.SetTermsOfServiceAccepted(timex.NewDateTimeZero().Time)

	tenantx := tenantxQuery.SaveX(ctx)

	accountQuery := ctx.VisitorCtx().MainTx.Account.
		Create().
		SetFirstName(data.FirstName).
		SetLastName(data.LastName).
		SetLanguage(data.Language).
		SetEmail(entx.NewCIText(data.Email)).
		SetRole(mainrole.User)
	if data.SubscribeToNewsletter {
		accountQuery.SetSubscribedToNewsletterAt(time.Now())
	}
	accountx := accountQuery.SaveX(ctx)

	// tenantx = tenantx.Update().AddUsers(userx).SaveX(ctx)
	_ = ctx.VisitorCtx().MainTx.TenantAccountAssignment.
		Create().
		SetTenant(tenantx).
		SetAccount(accountx).
		SetIsContactPerson(true).
		SetRole(tenantrole.Owner).
		SetIsDefault(true).
		SaveX(ctx)

	// Tenant.User is created in tenant initialization

	accountm := modelmain.NewAccount(accountx)
	password, expiresAt, err := accountm.GenerateTemporaryPassword(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	mailer.NewMailer().SignUp(ctx, accountx, password, expiresAt)

	rw.AddRenderables(wx.NewSnackbarf("Registration successful, please check your email for your password."))

	// TODO how to add tenant user? TTx not existing yet; on login or security risk?
	return nil
}
