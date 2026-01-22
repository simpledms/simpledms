package browse

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
		actions.Route("make-dir"),
		false,
	)
	return &MakeDirCmd{
		infra,
		actions,
		config,
		autil.NewFormHelper[MakeDirCmdData](
			infra,
			config,
			wx.T("Create directory"),
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

func (qq *MakeDirCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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
		wx.NewSnackbarf("«%s» created.", filex.Data.Name).WithAction(&wx.Link{
			Href:  route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String()),
			Child: wx.T("Open directory"), // TODO Go to, open, show?
		}),
	)
	return nil
}
