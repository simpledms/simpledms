package managetenantusers

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/entmain/account"
	"github.com/simpledms/simpledms/app/simpledms/entx"
	"github.com/simpledms/simpledms/app/simpledms/event"
	"github.com/simpledms/simpledms/app/simpledms/model/common/language"
	"github.com/simpledms/simpledms/app/simpledms/model/common/mainrole"
	"github.com/simpledms/simpledms/app/simpledms/model/common/tenantrole"
	"github.com/simpledms/simpledms/app/simpledms/model/mailer"
	"github.com/simpledms/simpledms/app/simpledms/model/modelmain"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type CreateUserCmdData struct {
	Role      tenantrole.TenantRole `validate:"required"`
	Email     string                `validate:"required,email" form_attrs:"autofocus"`
	FirstName string                `validate:"required"`
	LastName  string                `validate:"required"`
	Language  language.Language     `validate:"required"` // `schema:"default:German"` // TODO default based on browser language
	// CustomMessage string            // TODO textarea
}

type CreateUserCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[CreateUserCmdData]
}

func NewCreateUserCmd(infra *common.Infra, actions *Actions) *CreateUserCmd {
	config := actionx.NewConfig(actions.Route("create-user-cmd"), false)
	return &CreateUserCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[CreateUserCmdData](infra, config, wx.T("Create user")),
	}
}

func (qq *CreateUserCmd) Data(
	role tenantrole.TenantRole,
	email, firstName, lastName string,
	language language.Language,
) *CreateUserCmdData {
	return &CreateUserCmdData{
		Role:      role,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Language:  language,
	}
}

func (qq *CreateUserCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CreateUserCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	if !ctx.IsTenantCtx() {
		return e.NewHTTPErrorf(http.StatusBadRequest, "You are not allowed to create users. No tenant selected.")
	}
	if ctx.TenantCtx().User.Role != tenantrole.Owner {
		return e.NewHTTPErrorf(http.StatusBadRequest, "You are not allowed to create users because you are not the owner.")
	}

	// TODO read-only tx?
	exists, err := ctx.TenantCtx().MainTx.Account.
		Query().
		Where(account.EmailEQ(entx.NewCIText(data.Email))).
		Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		// contact support so that adding a user to multiple tenants can be implemented...
		return e.NewHTTPErrorf(http.StatusConflict, "A user with this email address already exists, please contact support if you want to add this user anyway.")
	}

	accountQuery := ctx.TenantCtx().MainTx.Account.
		Create().
		SetFirstName(data.FirstName).
		SetLastName(data.LastName).
		SetLanguage(data.Language).
		SetEmail(entx.NewCIText(data.Email)).
		SetRole(mainrole.User)
	accountx := accountQuery.SaveX(ctx)

	_ = ctx.TenantCtx().MainTx.TenantAccountAssignment.
		Create().
		SetTenant(ctx.TenantCtx().Tenant).
		SetAccount(accountx).
		SetRole(data.Role).
		SetIsDefault(true).
		SaveX(ctx)

	ctx.TenantCtx().TTx.User.Create().
		SetAccountID(accountx.ID).
		SetRole(data.Role).
		SetEmail(accountx.Email).
		SetFirstName(accountx.FirstName).
		SetLastName(accountx.LastName).
		SaveX(ctx)

	accountm := modelmain.NewAccount(accountx)
	password, expiresAt, err := accountm.GenerateTemporaryPassword(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	mailer.NewMailer().CreateUser(ctx, accountx, password, expiresAt)

	if data.Role == tenantrole.Owner {
		rw.AddRenderables(wx.NewSnackbarf("Successfully created the new user. The passwort was sent to the user by mail. An owner can access all spaces without further configuration."))
	} else {
		rw.AddRenderables(wx.NewSnackbarf("Successfully created the new user. The passwort was sent to the user by mail. The next step is to permit the user to access a space."))
	}

	rw.Header().Set("HX-Trigger", event.UserCreated.String())

	return nil
}
