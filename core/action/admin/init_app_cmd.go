package admin

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	appmodel "github.com/simpledms/simpledms/core/model/app"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type InitAppCmdData struct {
	Passphrase        string `validate:"required" form_attr_type:"password"`
	ConfirmPassphrase string `validate:"required" form_attr_type:"password"`
	// TODO option to provide identity?

	appmodel.S3Config
	appmodel.TLSConfig
	appmodel.MailerConfig
	appmodel.OCRConfig
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
		FormHelper: autil.NewFormHelper[InitAppCmdData](infra, config, widget.T("Init app")),
	}
}

func (qq *InitAppCmd) Data() *InitAppCmdData {
	return &InitAppCmdData{}
}

func (qq *InitAppCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[InitAppCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	if data.Passphrase != data.ConfirmPassphrase {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Passphrases do not match.")
	}

	err = appmodel.InitApp(
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

	rw.Header().Set("HX-Trigger", events.AppInitialized.String())
	rw.AddRenderables(widget.NewSnackbarf("App initialized."))

	return nil
}
