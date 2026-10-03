package browse

// package action

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type RenameFileCmdData struct {
	FileID      string `validate:"required" form_attr_type:"hidden"`
	NewFilename string `validate:"required" form_attrs:"autofocus"`
}

type RenameFileCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[RenameFileCmdData]
}

func NewRenameFileCmd(infra *common.Infra, actions *Actions) *RenameFileCmd {
	config := actionx.NewConfig(
		actions.Route("rename-file-cmd"),
		false,
	)
	return &RenameFileCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelperX[RenameFileCmdData](
			infra,
			config,
			widget.T("Rename file"),
			widget.T("Rename"),
		),
	}
}

func (qq *RenameFileCmd) Data(fileID, newFilename string) *RenameFileCmdData {
	return &RenameFileCmdData{
		FileID:      fileID,
		NewFilename: newFilename,
	}
}

func (qq *RenameFileCmd) FormHandler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[RenameFileCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	wrapper := req.URL.Query().Get("wrapper")

	form := &widget.Form{
		HTMXAttrs: widget.HTMXAttrs{
			HxPost: qq.Endpoint(),
			HxSwap: "none",
		},
		Children: []widget.IWidget{
			widget.NewFormFields(ctx, data),
		},
	}
	container := &widget.Container{
		GapY: true,
		Child: []widget.IWidget{
			&widget.Container{
				Child: []widget.IWidget{
					widget.NewLabel(widget.LabelTypeMd, widget.T("Original filename")),
					widget.NewBody(widget.BodyTypeSm, widget.Tu(data.NewFilename)),
				},
			},
			form,
		},
	}

	qq.infra.Renderer().RenderX(rw, ctx,
		autil.WrapWidgetWithID(
			widget.T("Rename file"),
			widget.T("Rename"),
			container,
			actionx.ResponseWrapper(wrapper),
			widget.DialogLayoutDefault,
			"",
			form.GetID(),
		),
	)
	return nil
}

func (qq *RenameFileCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[RenameFileCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)
	filex, err = qq.infra.FileSystem().Rename(ctx, filex, data.NewFilename)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.AddRenderables(widget.NewSnackbarf("Renamed to «%s».", filex.Data.Name))
	rw.Header().Add("HX-Trigger", event.FileUpdated.String())

	return nil
}
