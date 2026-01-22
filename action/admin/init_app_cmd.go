package admin

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/modelmain"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type InitAppCmdData struct {
	Passphrase        string `validate:"required" form_attr_type:"password"`
	ConfirmPassphrase string `validate:"required" form_attr_type:"password"`
	// TODO option to provide identity?

	modelmain.S3Config
	modelmain.TLSConfig
	modelmain.MailerConfig
	modelmain.OCRConfig
}

type InitAppCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[InitAppCmdData]
}

func NewInitAppCmd(infra *common.Infra, actions *Actions) *InitAppCmd {
	config := actionx.NewConfig(actions.Route("init-app-cmd"), false)
	return &InitAppCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[InitAppCmdData](infra, config, wx.T("Init app")),
	}
}

func (qq *InitAppCmd) Data() *InitAppCmdData {
	return &InitAppCmdData{}
}

func (qq *InitAppCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[InitAppCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	if data.Passphrase != data.ConfirmPassphrase {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Passphrases do not match.")
	}

	err = modelmain.InitApp(
		ctx,
		data.Passphrase,
		false,
		data.S3Config,
		data.TLSConfig,
		data.MailerConfig,
		data.OCRConfig,
	)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("HX-Trigger", event.AppInitialized.String())
	rw.AddRenderables(wx.NewSnackbarf("App initialized."))

	return nil
}
