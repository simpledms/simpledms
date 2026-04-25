package inbox

// package action

import (
	"log"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	actionx2 "github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type AssignFileCmdData struct {
	DestDirID string `form_attr_type:"hidden"`
	FileID    string `form_attr_type:"hidden"`
	Filename  string `form_attrs:"autofocus"`
}

type AssignFileCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx2.Config
	*autil.FormHelper[AssignFileCmdData]
}

func NewAssignFileCmd(infra *common.Infra, actions *Actions) *AssignFileCmd {
	config := actionx2.NewConfig(
		actions.Route("assign-file-cmd"),
		false,
	)
	formHelper := autil.NewFormHelper[AssignFileCmdData](infra, config, widget.T("Assign file"))
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

func (qq *AssignFileCmd) FormHandler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AssignFileCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	wrapper := req.URL.Query().Get("wrapper")
	hxTarget := req.URL.Query().Get("hx-target")

	formID := "assignFileForm"
	container := &widget.Container{
		GapY: true,
		Child: []widget.IWidget{
			&widget.Container{
				Child: []widget.IWidget{
					widget.NewLabel(widget.LabelTypeMd, widget.T("Original filename")),
					widget.NewBody(widget.BodyTypeSm, widget.Tu(data.Filename)),
				},
			},
			&widget.Form{
				Widget: widget.Widget[widget.Form]{
					ID: formID,
				},
				HTMXAttrs: widget.HTMXAttrs{
					HxPost:   qq.Endpoint(),
					HxTarget: hxTarget,
					HxSwap:   "outerHTML",
				},
				Children: []widget.IWidget{
					widget.NewFormFields(ctx, data),
				},
			},
		},
	}

	qq.infra.Renderer().RenderX(rw, ctx,
		autil.WrapWidgetWithID(
			widget.T("Assign file"),
			widget.T("Save"),
			container,
			actionx2.ResponseWrapper(wrapper),
			widget.DialogLayoutDefault,
			"",
			formID,
		),
	)
	return nil
}

func (qq *AssignFileCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

	filex.Data.Update().SetIsInInbox(false).SaveX(ctx)

	// TODO snackbar not shown; modal not closed
	// rw.Header().Set("HX-Location", route.InboxRoot())

	action := &widget.Link{
		Href: route.BrowseFile(
			ctx.TenantCtx().TenantID,
			ctx.SpaceCtx().SpaceID,
			filex.Parent(ctx).Data.PublicID.String(),
			filex.Data.PublicID.String(),
		),
		Child: widget.T("Open file"),
	}

	rw.AddRenderables(
		widget.NewSnackbarf("Moved to «%s».", destDir.Data.Name).WithAction(action),
	)
	rw.Header().Set("HX-Trigger", event.FileMoved.String())
	// TODO not nice because logic to reload list and close details is implemented by handling FileMoved event
	// TODO select next file to process instead
	rw.Header().Set("HX-Replace-Url", route.InboxRoot(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID))

	return nil
}
