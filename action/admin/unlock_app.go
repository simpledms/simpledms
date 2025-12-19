package admin

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/encryptor"
	"github.com/simpledms/simpledms/model/common/mainrole"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type UnlockAppData struct {
	Passphrase string `validate:"required" form_attr_type:"password"`
}

// TODO correct package?
type UnlockApp struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[UnlockAppData]
}

func NewUnlockApp(infra *common.Infra, actions *Actions) *UnlockApp {
	config := actionx.NewConfig(actions.Route("unlock-app"), false)
	return &UnlockApp{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[UnlockAppData](infra, config, wx.T("Unlock app")),
	}
}

func (qq *UnlockApp) Data() *UnlockAppData {
	return &UnlockAppData{}
}

// TODO DDos protection
func (qq *UnlockApp) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	if !ctx.IsMainCtx() {
		return e.NewHTTPErrorf(http.StatusUnauthorized, "You must be logged in to unlock the app.")
	}
	if ctx.MainCtx().Account.Role != mainrole.Admin {
		return e.NewHTTPErrorf(http.StatusForbidden, "You must be an admin to unlock the app.")
	}

	data, err := autil.FormData[UnlockAppData](rw, req, ctx)
	if err != nil {
		return err
	}

	err = qq.infra.SystemConfig().Unlock(data.Passphrase)
	if err != nil {
		log.Println(err)
		return err
	}

	encryptor.NilableX25519MainIdentity = qq.infra.SystemConfig().NilableX25519Identity()

	rw.Header().Set("HX-Trigger", event.AppUnlocked.String())
	rw.AddRenderables(wx.NewSnackbarf("App unlocked."))

	return nil
}
