package managetenantusers

import (
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/model/common/tenantrole"
	"github.com/simpledms/simpledms/model/modelmain"
	"github.com/simpledms/simpledms/ui/uix/event"
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

	err = modelmain.NewTenantUserService().Create(
		ctx,
		data.Role,
		data.Email,
		data.FirstName,
		data.LastName,
		data.Language,
		qq.infra.SystemConfig().AbsoluteURL("/"),
	)
	if err != nil {
		return err
	}

	if data.Role == tenantrole.Owner {
		rw.AddRenderables(wx.NewSnackbarf("Successfully created the new user. The passwort was sent to the user by mail. An owner can access all spaces without further configuration."))
	} else {
		rw.AddRenderables(wx.NewSnackbarf("Successfully created the new user. The passwort was sent to the user by mail. The next step is to permit the user to access a space."))
	}

	rw.Header().Set("HX-Trigger", event.UserCreated.String())

	return nil
}
