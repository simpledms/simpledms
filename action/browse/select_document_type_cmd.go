package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
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

func (qq *SelectDocumentTypeCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[SelectDocumentTypeCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	repos := qq.infra.SpaceFileRepoFactory().ForSpaceX(ctx)
	fileDTO := repos.Read.FileByPublicIDX(ctx, data.FileID)
	if fileDTO.DocumentTypeID == data.DocumentTypeID {
		err = repos.Write.SetFileDocumentTypeByIDX(ctx, fileDTO.ID, nil)
		if err != nil {
			return err
		}
		rw.AddRenderables(wx.NewSnackbarf("Document type deselected."))
	} else {
		err = repos.Write.SetFileDocumentTypeByIDX(ctx, fileDTO.ID, &data.DocumentTypeID)
		if err != nil {
			return err
		}
		rw.AddRenderables(wx.NewSnackbarf("Document type selected."))
	}

	return nil
}
