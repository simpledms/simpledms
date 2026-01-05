package auth

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/common/country"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/model/modelmain"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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

	_, err = modelmain.NewSignUpService().SignUp(
		ctx,
		data.Email,
		tenantName,
		data.FirstName,
		data.LastName,
		data.Country,
		data.Language,
		data.SubscribeToNewsletter,
	)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.AddRenderables(wx.NewSnackbarf("Registration successful, please check your email for your password."))

	// TODO how to add tenant user? TTx not existing yet; on login or security risk?
	return nil
}
