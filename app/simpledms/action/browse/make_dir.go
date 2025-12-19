package browse

import (
	"log"

	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/ui/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type MakeDirData struct {
	ParentDirID string `validate:"required" form_attr_type:"hidden"`
	DirName     string `validate:"required" form_attrs:"autofocus"`
}

// TODO or CreateDir?
type MakeDir struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[MakeDirData]
}

func NewMakeDir(
	infra *common.Infra,
	actions *Actions,
) *MakeDir {
	config := actionx.NewConfig(
		actions.Route("make-dir"),
		false,
	)
	return &MakeDir{
		infra,
		actions,
		config,
		autil.NewFormHelper[MakeDirData](
			infra,
			config,
			wx.T("Create directory"),
			// "#fileList",
		),
	}
}

func (qq *MakeDir) Data(parentDirID, dirName string) *MakeDirData {
	return &MakeDirData{
		ParentDirID: parentDirID,
		DirName:     dirName,
	}
}

func (qq *MakeDir) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[MakeDirData](rw, req, ctx)
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
		qq.actions.ListDir.WidgetHandler(
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
