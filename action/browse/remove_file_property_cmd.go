package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type RemoveFilePropertyCmdData struct {
	FileID     string
	PropertyID int64
}

type RemoveFilePropertyCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewRemoveFilePropertyCmd(infra *common.Infra, actions *Actions) *RemoveFilePropertyCmd {
	config := actionx.NewConfig(
		actions.Route("remove-file-property-cmd"),
		false,
	)
	return &RemoveFilePropertyCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *RemoveFilePropertyCmd) Data(fileID string, propertyID int64) *RemoveFilePropertyCmdData {
	return &RemoveFilePropertyCmdData{
		FileID:     fileID,
		PropertyID: propertyID,
	}
}

func (qq *RemoveFilePropertyCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[RemoveFilePropertyCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)
	propertyx := ctx.SpaceCtx().Space.QueryProperties().Where(property.ID(data.PropertyID)).OnlyX(ctx)

	ctx.SpaceCtx().TTx.FilePropertyAssignment.Delete().
		Where(
			filepropertyassignment.FileID(filex.Data.ID),
			filepropertyassignment.PropertyID(data.PropertyID),
		).ExecX(ctx)

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.FilePropertyUpdated.String())
	rw.AddRenderables(wx.NewSnackbarf("«%s» removed.", propertyx.Name))

	return nil
}
