package inbox

// package action

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type AssignFileData struct {
	DestDirID string `form_attr_type:"hidden"`
	FileID    string `form_attr_type:"hidden"`
	Filename  string `form_attrs:"autofocus"`
}

type AssignFile struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[AssignFileData]
}

func NewAssignFile(infra *common.Infra, actions *Actions) *AssignFile {
	config := actionx.NewConfig(
		actions.Route("assign-file"),
		false,
	)
	formHelper := autil.NewFormHelper[AssignFileData](infra, config, wx.T("Assign file"))
	return &AssignFile{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: formHelper,
	}
}

func (qq *AssignFile) Data(destDirID, fileID, filename string) *AssignFileData {
	return &AssignFileData{
		DestDirID: destDirID,
		FileID:    fileID,
		Filename:  filename,
	}
}

func (qq *AssignFile) FormHandler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AssignFileData](rw, req, ctx)
	if err != nil {
		return err
	}

	wrapper := req.URL.Query().Get("wrapper")
	hxTarget := req.URL.Query().Get("hx-target")

	container := &wx.Container{
		GapY: true,
		Child: []wx.IWidget{
			&wx.Container{
				Child: []wx.IWidget{
					wx.NewLabel(wx.LabelTypeMd, wx.T("Original filename")),
					wx.NewBody(wx.BodyTypeSm, wx.Tu(data.Filename)),
				},
			},
			&wx.Form{
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:   qq.Endpoint(),
					HxTarget: hxTarget,
					HxSwap:   "outerHTML",
				},
				Children: []wx.IWidget{
					wx.NewFormFields(ctx, data),
				},
			},
		},
	}

	qq.infra.Renderer().RenderX(rw, ctx,
		autil.WrapWidget(
			wx.T("Assign file"),
			wx.T("Save"),
			container,
			actionx.ResponseWrapper(wrapper),
			wx.DialogLayoutDefault,
		),
	)
	return nil
}

func (qq *AssignFile) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AssignFileData](rw, req, ctx)
	if err != nil {
		return err
	}

	destDir := qq.infra.FileRepo.GetX(ctx, data.DestDirID)
	filex := qq.infra.FileRepo.GetWithParentX(ctx, data.FileID)

	// FIXME see comment in MoveFile
	filex.File, err = qq.infra.FileSystem().Move(ctx, destDir, filex.File, data.Filename, "")
	if err != nil {
		log.Println(err)
		return err
	}

	// TODO snackbar not shown; modal not closed
	// rw.Header().Set("HX-Location", route.InboxRoot())

	action := &wx.Link{
		Href:  route.BrowseFile(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Parent(ctx).Data.PublicID.String(), filex.Data.PublicID.String()),
		Child: wx.T("Open file"),
	}
	return qq.infra.Renderer().Render(rw, ctx,
		wx.NewSnackbarf("Moved to «%s».", destDir.Data.Name).WithAction(action),
	)
}
