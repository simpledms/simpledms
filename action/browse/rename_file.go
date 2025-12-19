package browse

// package action

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type RenameFileData struct {
	FileID      string `validate:"required" form_attr_type:"hidden"`
	NewFilename string `validate:"required" form_attrs:"autofocus"`
}

type RenameFile struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[RenameFileData]
}

func NewRenameFile(infra *common.Infra, actions *Actions) *RenameFile {
	config := actionx.NewConfig(
		actions.Route("rename-file"),
		false,
	)
	return &RenameFile{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[RenameFileData](infra, config, wx.T("Rename file")),
	}
}

func (qq *RenameFile) Data(fileID, newFilename string) *RenameFileData {
	return &RenameFileData{
		FileID:      fileID,
		NewFilename: newFilename,
	}
}

func (qq *RenameFile) FormHandler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[RenameFileData](rw, req, ctx)
	if err != nil {
		return err
	}

	wrapper := req.URL.Query().Get("wrapper")
	hxTarget := req.URL.Query().Get("hx-target")

	form := &wx.Form{
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   "outerHTML",
		},
		Children: []wx.IWidget{
			wx.NewFormFields(ctx, data),
		},
	}
	container := &wx.Container{
		GapY: true,
		Child: []wx.IWidget{
			&wx.Container{
				Child: []wx.IWidget{
					wx.NewLabel(wx.LabelTypeMd, wx.T("Original filename")),
					wx.NewBody(wx.BodyTypeSm, wx.Tu(data.NewFilename)),
				},
			},
			form,
		},
	}

	qq.infra.Renderer().RenderX(rw, ctx,
		autil.WrapWidgetWithID(
			wx.T("Rename file"),
			wx.T("Save"),
			container,
			actionx.ResponseWrapper(wrapper),
			wx.DialogLayoutDefault,
			"",
			form.GetID(),
		),
	)
	return nil
}

func (qq *RenameFile) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[RenameFileData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)
	filex, err = qq.infra.FileSystem().Rename(ctx, filex, data.NewFilename)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.AddRenderables(wx.NewSnackbarf("Renamed to «%s»", filex.Data.Name))
	rw.Header().Add("HX-Trigger", event.FileUpdated.String())

	return nil
}

/*
func (qq *RenameFile) Widget(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context, filex *model.File) *wx.ListDetailLayout {
	parent, err := filex.Parent(ctx)
	if err != nil {
		log.Println(err)
		panic(err)
	}
	// complete list because order can change
	// TODO selected file?
	return qq.actions.ListDir.WidgetHandler(
		rw,
		req,
		ctx,
		parent.Data.PublicID.String(),
		"",
	)
}
*/
