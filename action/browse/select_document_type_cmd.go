package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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
	).EnableCommittedResponse()
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

func (qq *SelectDocumentTypeCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[SelectDocumentTypeCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)
	isSelected, _, err := documenttypemodel.NewAssignmentService().Toggle(
		ctx, filex.Data.ID, data.DocumentTypeID,
	)
	if err != nil {
		return err
	}
	if !isSelected {
		rw.AddRenderables(wx.NewSnackbarf("Document type deselected."))
	} else {
		rw.AddRenderables(wx.NewSnackbarf("Document type selected."))
	}

	return nil
}
