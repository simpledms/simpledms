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

type InitAppData struct {
	Passphrase        string `validate:"required" form_attr_type:"password"`
	ConfirmPassphrase string `validate:"required" form_attr_type:"password"`
	// TODO option to provide identity?

	modelmain.S3Config
	modelmain.TLSConfig
	modelmain.MailerConfig
	modelmain.OCRConfig
}

type InitApp struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[InitAppData]
}

func NewInitApp(infra *common.Infra, actions *Actions) *InitApp {
	config := actionx.NewConfig(actions.Route("init-app"), false)
	return &InitApp{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[InitAppData](infra, config, wx.T("Init app")),
	}
}

func (qq *InitApp) Data() *InitAppData {
	return &InitAppData{}
}

func (qq *InitApp) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[InitAppData](rw, req, ctx)
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
