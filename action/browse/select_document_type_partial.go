package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type SelectDocumentTypePartialData struct {
	FileID         string
	DocumentTypeID int64
}

type SelectDocumentTypePartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewSelectDocumentTypePartial(infra *common.Infra, actions *Actions) *SelectDocumentTypePartial {
	config := actionx.NewConfig(
		actions.Route("select-document-type"),
		false,
	)
	return &SelectDocumentTypePartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *SelectDocumentTypePartial) Data(fileID string, documentTypeID int64) *SelectDocumentTypePartialData {
	return &SelectDocumentTypePartialData{
		FileID:         fileID,
		DocumentTypeID: documentTypeID,
	}
}

func (qq *SelectDocumentTypePartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[SelectDocumentTypePartialData](rw, req, ctx)
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
