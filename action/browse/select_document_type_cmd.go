package browse

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
)

type SelectDocumentTypeCmdData struct {
	FileID         string
	DocumentTypeID int64
}

type SelectDocumentTypeCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewSelectDocumentTypeCmd(infra *common.Infra, actions *Actions) *SelectDocumentTypeCmd {
	config := actionx.NewConfig(
		actions.Route("select-document-type-cmd"),
		false,
	)
	return &SelectDocumentTypeCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *SelectDocumentTypeCmd) Data(fileID string, documentTypeID int64) *SelectDocumentTypeCmdData {
	return &SelectDocumentTypeCmdData{
		FileID:         fileID,
		DocumentTypeID: documentTypeID,
	}
}

func (qq *SelectDocumentTypeCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[SelectDocumentTypeCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)
	if filex.Data.DocumentTypeID == data.DocumentTypeID {
		filex.Data.Update().ClearDocumentTypeID().SaveX(ctx)
		rw.AddRenderables(wx.NewSnackbarf("Document type deselected."))
	} else {
		filex.Data.Update().SetDocumentTypeID(data.DocumentTypeID).SaveX(ctx)
		rw.AddRenderables(wx.NewSnackbarf("Document type selected."))
	}

	return nil
}
