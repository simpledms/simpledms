package inbox

// package action

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	filingmodel "github.com/simpledms/simpledms/model/tenant/filing"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
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
		actions.Route("assign-file-cmd"),
		false,
	).EnableCommittedResponse()
	formHelper := autil.NewFormHelperX[AssignFileCmdData](
		infra,
		config,
		widget.T("Assign file"),
		widget.T("Assign"),
	)
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
					HxPost: qq.Endpoint(),
					HxSwap: "none",
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
			widget.T("Assign"),
			container,
			actionx.ResponseWrapper(wrapper),
			widget.DialogLayoutDefault,
			"",
			formID,
		),
	)
	return nil
}

func (qq *AssignFileCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AssignFileCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	fileData, err := filingmodel.NewFilingService(qq.infra.FileSystem()).FileInboxDocument(
		ctx, data.FileID, data.DestDirID, data.Filename, "",
	)
	if err != nil {
		log.Println(err)
		return err
	}
	filex := qq.infra.FileRepo.GetWithParentX(ctx, fileData.PublicID.String())
	destDir := qq.infra.FileRepo.GetX(ctx, data.DestDirID)

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

	return nil
}
