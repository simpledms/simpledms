package tagging

import (
	"log"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/db/enttenant"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type CreateTagCmdData struct {
	GroupTagID int64           `form_attr_type:"hidden"`
	Name       string          `validate:"required" form_attrs:"autofocus"`
	Type       tagtype.TagType `validate:"required"` // `schema:"default:Base"`
	// IsGroup     bool
	// IsComposed  bool
	// Color    string
	// Icon     string
	// Weight   int
	// AttributeName   string
}

// TODO or AddNewTag NewTag
type CreateTagCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[CreateTagCmdData]
}

func NewCreateTagCmd(
	infra *common.Infra,
	actions *Actions,
) *CreateTagCmd {
	config := actionx.NewConfig(
		actions.Route("create-tag-cmd"),
		false,
	)
	return &CreateTagCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelper[CreateTagCmdData](
			infra,
			config,
			widget.T("Create tag"),
			// "TODO", // TODO
		),
	}
}

func (qq *CreateTagCmd) Data(groupTagID int64) *CreateTagCmdData {
	return &CreateTagCmdData{
		GroupTagID: groupTagID,
	}
}

func (qq *CreateTagCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CreateTagCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	tagx, err := qq.execute(ctx, data)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.TagCreated.String())
	rw.AddRenderables(widget.NewSnackbarf("Tag «%s» created.", tagx.Name))

	return nil
}

func (qq *CreateTagCmd) execute(ctx ctxx.Context, data *CreateTagCmdData) (*enttenant.Tag, error) {
	return taggingmodel.NewTagService().Create(
		ctx,
		ctx.SpaceCtx().Space.ID,
		data.GroupTagID,
		data.Name,
		data.Type,
	)
}
