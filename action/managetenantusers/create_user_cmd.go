package managetenantusers

import (
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/common/language"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	tenantusermodel "github.com/simpledms/simpledms/model/main/tenantuser"
	"github.com/simpledms/simpledms/ui/uix/event"
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
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelperX[CreateUserCmdData](
			infra,
			config,
			widget.T("Create user"),
			widget.T("Create"),
		),
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
		return e.NewHTTPErrorf(http.StatusBadRequest, "You are not allowed to create users. No organization selected.")
	}
	if ctx.TenantCtx().User.Role != tenantrole.Owner {
		return e.NewHTTPErrorf(http.StatusBadRequest, "You are not allowed to create users because you are not the owner.")
	}

	err = tenantusermodel.Create(
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
		rw.AddRenderables(widget.NewSnackbarf(
			"User created. The password was sent by email. Owners can access all Spaces without further setup.",
		))
	} else {
		rw.AddRenderables(widget.NewSnackbarf(
			"User created. The password was sent by email. Next, assign the user to a Space.",
		))
	}

	rw.Header().Set("HX-Trigger", event.UserCreated.String())

	return nil
}
