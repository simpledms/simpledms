package browse

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type MakeDirCmdData struct {
	ParentDirID string `validate:"required" form_attr_type:"hidden"`
	FolderName  string `validate:"required" form_attrs:"autofocus"`
}

// TODO or CreateDir?
type MakeDirCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[MakeDirCmdData]
}

func NewMakeDirCmd(
	infra *common.Infra,
	actions *Actions,
) *MakeDirCmd {
	config := actionx.NewConfig(
		actions.Route("make-dir-cmd"),
		false,
	)
	return &MakeDirCmd{
		infra,
		actions,
		config,
		autil.NewFormHelperX[MakeDirCmdData](
			infra,
			config,
			widget.T("Create folder"),
			widget.T("Create"),
		),
	}
}

func (qq *MakeDirCmd) Data(parentDirID, dirName string) *MakeDirCmdData {
	return &MakeDirCmdData{
		ParentDirID: parentDirID,
		FolderName:  dirName,
	}
}

func (qq *MakeDirCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[MakeDirCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex, err := qq.infra.FileSystem().MakeDir(ctx, data.ParentDirID, data.FolderName)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("HX-Trigger", event.DirectoryCreated.String())
	rw.AddRenderables(
		widget.NewSnackbarf("«%s» created.", filex.Data.Name).WithAction(&widget.Link{
			Href:  route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String()),
			Child: widget.T("Open folder"),
		}),
	)
	return nil
}
