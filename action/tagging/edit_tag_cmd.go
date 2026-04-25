package tagging

import (
	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type EditTagCmdData struct {
	// necessary for response (isChecked)
	TagID int64  `validate:"required" form_attr_type:"hidden"`
	Name  string `validate:"required" form_attrs:"autofocus"`
}

type EditTagCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[EditTagCmdData]
}

func NewEditTagCmd(
	infra *common.Infra,
	actions *Actions,
) *EditTagCmd {
	config := actionx.NewConfig(actions.Route("edit-tag-cmd"), false)
	return &EditTagCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelper[EditTagCmdData](
			infra,
			config,
			widget.T("Edit tag"),
			// "",
		),
	}
}

func (qq *EditTagCmd) Data(tagID int64, name string) *EditTagCmdData {
	return &EditTagCmdData{
		TagID: tagID,
		Name:  name,
	}
}

func (qq *EditTagCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := qq.MapFormData(rw, req, ctx)
	if err != nil {
		return err
	}

	tagx, err := taggingmodel.NewTagService().Edit(ctx, data.TagID, data.Name)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.TagUpdated.String())
	rw.Header().Set("HX-Reswap", "none")
	rw.AddRenderables(widget.NewSnackbarf("«%s» updated.", tagx.Name))

	return nil
}
