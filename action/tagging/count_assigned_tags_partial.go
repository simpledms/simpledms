package tagging

import (
	"fmt"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/ui/uix/events"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type CountAssignedTagsPartialData struct {
	FileID string
	Layout string // TODO enum
}

type CountAssignedTagsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewCountAssignedTagsPartial(infra *common.Infra, actions *Actions) *CountAssignedTagsPartial {
	return &CountAssignedTagsPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("count-assigned-tags-partial"),
			true,
		),
	}
}

func (qq *CountAssignedTagsPartial) Data(fileID string) *CountAssignedTagsPartialData {
	return &CountAssignedTagsPartialData{
		FileID: fileID,
	}
}

func (qq *CountAssignedTagsPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CountAssignedTagsPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	// TODO handle Layout
	qq.infra.Renderer().RenderX(rw, ctx, qq.Badge(ctx, data.FileID))
	return nil
}

func (qq *CountAssignedTagsPartial) Badge(ctx ctxx.Context, fileID string) *widget.Badge {
	// soft delete filter is not applied via TagAssignment
	filex := qq.infra.FileRepo.GetX(ctx, fileID)
	tagsCount := filex.Data.QueryTags().CountX(ctx)

	id := autil.GenerateID(fmt.Sprintf("tagsCount-%s", fileID))
	return &widget.Badge{
		Widget: widget.Widget[widget.Badge]{
			ID: id,
		},
		Value:    tagsCount,
		IsInline: true,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:    qq.Endpoint(),
			HxTrigger: events.HxTrigger(event.TagUpdated),
			HxVals:    util.JSON(qq.Data(fileID)),
			HxTarget:  "#" + id,
			HxSwap:    "outerHTML",
		},
	}
}
