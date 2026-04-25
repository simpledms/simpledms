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

type RemovePassphraseCmdData struct {
	CurrentPassphrase string `validate:"required" form_attr_type:"password"`
}

type RemovePassphraseCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[RemovePassphraseCmdData]
}

func NewRemovePassphraseCmd(infra *common.Infra, actions *Actions) *RemovePassphraseCmd {
	config := actionx.NewConfig(actions.Route("remove-passphrase-cmd"), false)
	return &RemovePassphraseCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[RemovePassphraseCmdData](infra, config, widget.T("Remove passphrase")),
	}
}

func (qq *RemovePassphraseCmd) Data() *RemovePassphraseCmdData {
	return &RemovePassphraseCmdData{}
}

func (qq *RemovePassphraseCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	if !ctx.IsMainCtx() {
		return e.NewHTTPErrorf(http.StatusUnauthorized, "You must be logged in to unlock the app.")
	}
	if ctx.MainCtx().Account.Role != mainrole.Admin {
		return e.NewHTTPErrorf(http.StatusForbidden, "You must be an admin to unlock the app.")
	}

	data, err := autil.FormData[RemovePassphraseCmdData](rw, req, ctx)
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

	rw.Header().Set("HX-Trigger", events.AppPassphraseChanged.String())
	rw.AddRenderables(widget.NewSnackbarf("Passphrase removed."))

	return nil
}
