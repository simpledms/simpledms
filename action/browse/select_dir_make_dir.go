package browse

import (
	"log"

	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type SelectDirMakeDirData struct {
	*acommon.MoveFileData `structs:",flatten"` // TODO use SelectDirData once implemented
	NewDirName            string               `form_attrs:"autofocus"`
}

// Name only makes sense once SelectDir is factored out from MoveFile and can
// be used independently
type SelectDirMakeDir struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[SelectDirMakeDirData]
}

func NewSelectDirMakeDir(
	infra *common.Infra,
	actions *Actions,
) *SelectDirMakeDir {
	config := actionx.NewConfig(
		actions.Route("select-dir/make-dir"), // TODO suffix should be handled by actions (embedding)
		false,
	)
	return &SelectDirMakeDir{
		infra,
		actions,
		config,
		autil.NewFormHelper[SelectDirMakeDirData](
			infra,
			config,
			wx.T("Create directory"),
			// "#fileList",
		),
	}
}

func (qq *SelectDirMakeDir) Data(moveFileData *acommon.MoveFileData, dirName string) *SelectDirMakeDirData {
	return &SelectDirMakeDirData{
		MoveFileData: moveFileData,
		NewDirName:   dirName,
	}
}

func (qq *SelectDirMakeDir) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[SelectDirMakeDirData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex, err := qq.infra.FileSystem().MakeDir(ctx, data.CurrentDirID, data.NewDirName)
	if err != nil {
		log.Println(err)
		return err
	}

	// switch dir
	data.MoveFileData.CurrentDirID = filex.Data.PublicID.String()

	// hxTarget := req.URL.Query().Get("hx-target")

	// TODO how to handle type of view? (table, list, cards)
	// TODO return list partial / may depend on context...
	return qq.infra.Renderer().Render(rw, ctx,
		// qq.actions.MoveFile.Form(ctx, data.MoveFileData, actionx.ResponseWrapperDialog, "#fileList"),
		wx.NewSnackbarf("«%s» created.", filex.Data.Name),
	)
}
