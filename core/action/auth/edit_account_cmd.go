package auth

import (
	"net/http"
	"time"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entx"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/model/common/language"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type EditAccountCmdData struct {
	// TODO change email flow (needs confirmation)
	AccountID string `validate:"required" form_attr_type:"hidden"`
	// don't allow name for the moment because it would also require changing all users...
	// FirstName             string            `validate:"required"`
	// LastName              string            `validate:"required"`
	Language              language.Language `validate:"required"`
	SubscribeToNewsletter bool
}

type EditAccountCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[EditAccountCmdData]
}

func NewEditAccountCmd(infra *common.Infra, actions *Actions) *EditAccountCmd {
	config := actionx.NewConfig(actions.Route("edit-account-cmd"), false)
	return &EditAccountCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[EditAccountCmdData](infra, config, widget.T("Edit account")),
	}
}

func (qq *EditAccountCmd) Data(accountID string, language language.Language, subscribeToNewsletter bool) *EditAccountCmdData {
	return &EditAccountCmdData{
		AccountID: accountID,
		// FirstName:             firstName,
		// LastName:              lastName,
		Language:              language,
		SubscribeToNewsletter: subscribeToNewsletter,
	}
}

func (qq *EditAccountCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[EditAccountCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	// querying instead of using MainCtx().Account is more robust against future changes
	accountx, err := ctx.MainCtx().MainTx.Account.Query().Where(account.PublicID(entx.NewCIText(data.AccountID))).Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusForbidden, "You cannot edit another account.")
		}

		return err
	}

	query := accountx.Update().SetLanguage(data.Language)
	if accountx.SubscribedToNewsletterAt == nil && data.SubscribeToNewsletter {
		query.SetSubscribedToNewsletterAt(time.Now())
	} else if accountx.SubscribedToNewsletterAt != nil && !data.SubscribeToNewsletter {
		query.ClearSubscribedToNewsletterAt()
	}
	accountx = query.SaveX(ctx)

	rw.AddRenderables(widget.NewSnackbarf("Account updated."))
	rw.Header().Set("HX-Trigger", events.AccountUpdated.String())

	return nil
}
