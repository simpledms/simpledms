package admin

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/model/common/mainrole"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
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
		FormHelper: autil.NewFormHelper[ChangePassphraseCmdData](infra, config, widget.T("Change passphrase")),
	}
}

func (qq *ChangePassphraseCmd) Data() *ChangePassphraseCmdData {
	return &ChangePassphraseCmdData{}
}

func (qq *ChangePassphraseCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

	rw.Header().Set("HX-Trigger", events.AppPassphraseChanged.String())
	rw.AddRenderables(widget.NewSnackbarf("Passphrase changed."))

	return nil
}
