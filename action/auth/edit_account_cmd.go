package auth

import (
	"net/http"
	"time"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/event"
	"github.com/simpledms/simpledms/model/common/language"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
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
		FormHelper: autil.NewFormHelper[EditAccountCmdData](infra, config, wx.T("Edit account")),
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

func (qq *EditAccountCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[EditAccountCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	if ctx.MainCtx().Account.PublicID.String() != data.AccountID {
		return e.NewHTTPErrorf(http.StatusForbidden, "You cannot edit another account.")
	}

	// querying instead of using MainCtx().Account is more robust against future changes
	accountx := ctx.MainCtx().MainTx.Account.Query().Where(account.PublicID(entx.NewCIText(data.AccountID))).OnlyX(ctx)

	query := accountx.Update().SetLanguage(data.Language)
	if accountx.SubscribedToNewsletterAt == nil && data.SubscribeToNewsletter {
		query.SetSubscribedToNewsletterAt(time.Now())
	} else if accountx.SubscribedToNewsletterAt != nil && !data.SubscribeToNewsletter {
		query.ClearSubscribedToNewsletterAt()
	}
	accountx = query.SaveX(ctx)

	rw.AddRenderables(wx.NewSnackbarf("Account updated."))
	rw.Header().Set("HX-Trigger", event.AccountUpdated.String())

	return nil
}
