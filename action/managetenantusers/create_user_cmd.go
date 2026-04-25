package managetenantusers

import (
	"net/http"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/model/common/language"
	"github.com/marcobeierer/go-core/model/common/tenantrole"
	tenantusermodel "github.com/marcobeierer/go-core/model/tenantuser"
	"github.com/marcobeierer/go-core/ui/uix/events"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	"github.com/marcobeierer/go-core/util/e"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
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
		FormHelper: autil.NewFormHelper[CreateUserCmdData](infra, config, widget.T("Create user")),
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

func (qq *CreateUserCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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
		rw.AddRenderables(widget.NewSnackbarf("Successfully created the new user. The passwort was sent to the user by mail. An owner can access all spaces without further configuration."))
	} else {
		rw.AddRenderables(widget.NewSnackbarf("Successfully created the new user. The passwort was sent to the user by mail. The next step is to permit the user to access a space."))
	}

	rw.Header().Set("HX-Trigger", events.UserCreated.String())

	return nil
}
