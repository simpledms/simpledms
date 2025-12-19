package admin

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/common/mainrole"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type RemovePassphraseData struct {
	CurrentPassphrase string `validate:"required" form_attr_type:"password"`
}

type RemovePassphrase struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[RemovePassphraseData]
}

func NewRemovePassphrase(infra *common.Infra, actions *Actions) *RemovePassphrase {
	config := actionx.NewConfig(actions.Route("remove-passphrase"), false)
	return &RemovePassphrase{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[RemovePassphraseData](infra, config, wx.T("Remove passphrase")),
	}
}

func (qq *RemovePassphrase) Data() *RemovePassphraseData {
	return &RemovePassphraseData{}
}

func (qq *RemovePassphrase) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	if !ctx.IsMainCtx() {
		return e.NewHTTPErrorf(http.StatusUnauthorized, "You must be logged in to unlock the app.")
	}
	if ctx.MainCtx().Account.Role != mainrole.Admin {
		return e.NewHTTPErrorf(http.StatusForbidden, "You must be an admin to unlock the app.")
	}

	data, err := autil.FormData[RemovePassphraseData](rw, req, ctx)
	if err != nil {
		return err
	}

	err = qq.infra.SystemConfig().RemovePassphrase(
		ctx,
		data.CurrentPassphrase,
	)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("HX-Trigger", event.AppPassphraseChanged.String())
	rw.AddRenderables(wx.NewSnackbarf("Passphrase removed."))

	return nil
}
