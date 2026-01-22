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

type AssignFileCmdData struct {
	DestDirID string `form_attr_type:"hidden"`
	FileID    string `form_attr_type:"hidden"`
	Filename  string `form_attrs:"autofocus"`
}

type AssignFileCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[AssignFileCmdData]
}

func NewAssignFileCmd(infra *common.Infra, actions *Actions) *AssignFileCmd {
	config := actionx.NewConfig(
		actions.Route("assign-file"),
		false,
	)
	formHelper := autil.NewFormHelper[AssignFileCmdData](infra, config, wx.T("Assign file"))
	return &AssignFileCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: formHelper,
	}
}

func (qq *AssignFileCmd) Data(destDirID, fileID, filename string) *AssignFileCmdData {
	return &AssignFileCmdData{
		DestDirID: destDirID,
		FileID:    fileID,
		Filename:  filename,
	}
}

func (qq *AssignFileCmd) FormHandler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AssignFileCmdData](rw, req, ctx)
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

func (qq *AssignFileCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AssignFileCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	destDir := qq.infra.FileRepo.GetX(ctx, data.DestDirID)
	filex := qq.infra.FileRepo.GetWithParentX(ctx, data.FileID)

	// FIXME see comment in MoveFileCmd
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
