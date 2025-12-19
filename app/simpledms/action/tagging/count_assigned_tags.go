package tagging

import (
	"fmt"

	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/event"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type CountAssignedTagsData struct {
	FileID string
	Layout string // TODO enum
}

type CountAssignedTags struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewCountAssignedTags(infra *common.Infra, actions *Actions) *CountAssignedTags {
	return &CountAssignedTags{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("count-assigned-tags"),
			true,
		),
	}
}

func (qq *CountAssignedTags) Data(fileID string) *CountAssignedTagsData {
	return &CountAssignedTagsData{
		FileID: fileID,
	}
}

func (qq *CountAssignedTags) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CountAssignedTagsData](rw, req, ctx)
	if err != nil {
		return err
	}

	// TODO handle Layout
	qq.infra.Renderer().RenderX(rw, ctx, qq.Badge(ctx, data.FileID))
	return nil
}

func (qq *CountAssignedTags) Badge(ctx ctxx.Context, fileID string) *wx.Badge {
	// soft delete filter is not applied via TagAssignment
	filex := qq.infra.FileRepo.GetX(ctx, fileID)
	tagsCount := filex.Data.QueryTags().CountX(ctx)

	id := autil.GenerateID(fmt.Sprintf("tagsCount-%d", fileID))
	return &wx.Badge{
		Widget: wx.Widget[wx.Badge]{
			ID: id,
		},
		Value:    tagsCount,
		IsInline: true,
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:    qq.Endpoint(),
			HxTrigger: fmt.Sprintf("%s from:body", event.TagUpdated.String()),
			HxVals:    util.JSON(qq.Data(fileID)),
			HxTarget:  "#" + id,
			HxSwap:    "outerHTML",
		},
	}
}
