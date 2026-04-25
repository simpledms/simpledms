package documenttype

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type ImportFromLibraryCmdData struct {
}

type ImportFromLibraryCmdFormData struct {
	ImportFromLibraryCmdData `structs:",flatten"`
	TemplateKeys             []string `validate:"required" form:"library_template_keys"`
}

type ImportFromLibraryCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewImportFromLibraryCmd(infra *common.Infra, actions *Actions) *ImportFromLibraryCmd {
	config := actionx.NewConfig(
		actions.Route("import-document-types-from-library-cmd"),
		false,
	)
	return &ImportFromLibraryCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *ImportFromLibraryCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ImportFromLibraryCmdFormData](rw, req, ctx)
	if err != nil {
		return err
	}

	if err := documenttypemodel.ImportFromLibrary(ctx, data.TemplateKeys); err != nil {
		return err
	}

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.DocumentTypeCreated.String())

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		wx.NewSnackbarf("Document types imported."),
	)
}
