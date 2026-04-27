package trash

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileTagsPartialData struct {
	FileID string
}

type FileTagsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileTagsPartial(infra *common.Infra, actions *Actions) *FileTagsPartial {
	return &FileTagsPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("file-tags-partial"),
			true,
		),
	}
}

func (qq *FileTagsPartial) Data(fileID string) *FileTagsPartialData {
	return &FileTagsPartialData{FileID: fileID}
}

func (qq *FileTagsPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileTagsPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *FileTagsPartial) Widget(ctx ctxx.Context, data *FileTagsPartialData) *wx.ScrollableContent {
	filex := qq.infra.FileRepo.GetWithDeletedX(ctx, data.FileID)
	assignedTags := filex.Data.QueryTags().
		WithGroup().
		Order(tag.ByName()).
		AllX(ctx)

	if len(assignedTags) == 0 {
		return &wx.ScrollableContent{
			Widget: wx.Widget[wx.ScrollableContent]{
				ID: "trashFileTags",
			},
			Children: &wx.EmptyState{
				Icon:     wx.NewIcon("label"),
				Headline: wx.T("No tags assigned."),
			},
			MarginY: true,
		}
	}

	var items []*wx.ListItem
	for _, tagx := range assignedTags {
		supporting := ""
		if tagx.Edges.Group != nil {
			supporting = tagx.Edges.Group.Name
		}
		item := &wx.ListItem{
			Headline: wx.Tu(tagx.Name),
		}
		if supporting != "" {
			item.SupportingText = wx.Tu(supporting)
		}
		items = append(items, item)
	}

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: "trashFileTags",
		},
		GapY: true,
		Children: &wx.List{
			Children: items,
		},
		MarginY: true,
	}
}
