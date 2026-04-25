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
	"github.com/simpledms/simpledms/encryptor"
)

type UnlockAppCmdData struct {
	Passphrase string `validate:"required" form_attr_type:"password"`
}

// TODO correct package?
type UnlockAppCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[UnlockAppCmdData]
}

func NewUnlockAppCmd(infra *common.Infra, actions *Actions) *UnlockAppCmd {
	config := actionx.NewConfig(actions.Route("unlock-app-cmd"), false)
	return &UnlockAppCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[UnlockAppCmdData](infra, config, widget.T("Unlock app")),
	}
}

func (qq *UnlockAppCmd) Data() *UnlockAppCmdData {
	return &UnlockAppCmdData{}
}

// TODO DDos protection
func (qq *UnlockAppCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	if !ctx.IsMainCtx() {
		return e.NewHTTPErrorf(http.StatusUnauthorized, "You must be logged in to unlock the app.")
	}
	if ctx.MainCtx().Account.Role != mainrole.Admin {
		return e.NewHTTPErrorf(http.StatusForbidden, "You must be an admin to unlock the app.")
	}

	data, err := autil.FormData[UnlockAppCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	err = qq.infra.SystemConfig().Unlock(data.Passphrase)
	if err != nil {
		log.Println(err)
		return err
	}

	encryptor.NilableX25519MainIdentity = qq.infra.SystemConfig().NilableX25519Identity()

	rw.Header().Set("HX-Trigger", events.AppUnlocked.String())
	rw.AddRenderables(widget.NewSnackbarf("App unlocked."))

	return nil
}
