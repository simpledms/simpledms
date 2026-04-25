package browse

import (
	"log"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type MakeDirCmdData struct {
	ParentDirID string `validate:"required" form_attr_type:"hidden"`
	DirName     string `validate:"required" form_attrs:"autofocus"`
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
		autil.NewFormHelper[MakeDirCmdData](
			infra,
			config,
			widget.T("Create directory"),
			// "#fileList",
		),
	}
}

func (qq *MakeDirCmd) Data(parentDirID, dirName string) *MakeDirCmdData {
	return &MakeDirCmdData{
		ParentDirID: parentDirID,
		DirName:     dirName,
	}
}

func (qq *MakeDirCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[MakeDirCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex, err := qq.infra.FileSystem().MakeDir(ctx, data.ParentDirID, data.DirName)
	if err != nil {
		log.Println(err)
		return err
	}

	// rw.Header().Set("HX-Push-Url", route.Browse(filex.ID))

	// TODO how to handle type of view? (table, list, cards)
	// TODO return list partial / may depend on context...
	qq.infra.Renderer().RenderX(rw, ctx,
		qq.actions.ListDirPartial.WidgetHandler(
			rw,
			req,
			ctx,
			data.ParentDirID,
			"",
		),
		widget.NewSnackbarf("«%s» created.", filex.Data.Name).WithAction(&widget.Link{
			Href:  route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String()),
			Child: widget.T("Open directory"), // TODO Go to, open, show?
		}),
	)
	return nil
}
