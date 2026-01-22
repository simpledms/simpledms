package admin

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/common/mainrole"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type ChangePassphraseCmdData struct {
	// FIXME show warning in form, that all data lost if passphrase gets lots
	//	     just add a description field to form? would allow to place descriptions anywhere in form...
	CurrentPassphrase    string `form_attr_type:"password"` // can be empty string
	NewPassphrase        string `validate:"required" form_attr_type:"password"`
	ConfirmNewPassphrase string `validate:"required" form_attr_type:"password"`
}

type ChangePassphraseCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[ChangePassphraseCmdData]
}

func NewChangePassphraseCmd(infra *common.Infra, actions *Actions) *ChangePassphraseCmd {
	config := actionx.NewConfig(actions.Route("change-passphrase-cmd"), false)
	return &ChangePassphraseCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[ChangePassphraseCmdData](infra, config, wx.T("Change passphrase")),
	}
}

func (qq *ChangePassphraseCmd) Data() *ChangePassphraseCmdData {
	return &ChangePassphraseCmdData{}
}

func (qq *ChangePassphraseCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	if !ctx.IsMainCtx() {
		return e.NewHTTPErrorf(http.StatusUnauthorized, "You must be logged in to unlock the app.")
	}
	if ctx.MainCtx().Account.Role != mainrole.Admin {
		return e.NewHTTPErrorf(http.StatusForbidden, "You must be an admin to unlock the app.")
	}

	data, err := autil.FormData[ChangePassphraseCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	err = qq.infra.SystemConfig().ChangePassphrase(
		ctx,
		data.CurrentPassphrase,
		data.NewPassphrase,
		data.ConfirmNewPassphrase,
	)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("HX-Trigger", event.AppPassphraseChanged.String())
	rw.AddRenderables(wx.NewSnackbarf("Passphrase changed."))

	return nil
}
