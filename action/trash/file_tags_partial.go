package trash

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/db/enttenant/tag"
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

func (qq *FileTagsPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

func (qq *FileTagsPartial) Widget(ctx ctxx.Context, data *FileTagsPartialData) *widget.ScrollableContent {
	filex := qq.infra.FileRepo.GetWithDeletedX(ctx, data.FileID)
	assignedTags := filex.Data.QueryTags().
		WithGroup().
		Order(tag.ByName()).
		AllX(ctx)

	if len(assignedTags) == 0 {
		return &widget.ScrollableContent{
			Widget: widget.Widget[widget.ScrollableContent]{
				ID: "trashFileTags",
			},
			Children: &widget.EmptyState{
				Icon:     widget.NewIcon("label"),
				Headline: widget.T("No tags assigned."),
			},
			MarginY: true,
		}
	}

	var items []*widget.ListItem
	for _, tagx := range assignedTags {
		supporting := ""
		if tagx.Edges.Group != nil {
			supporting = tagx.Edges.Group.Name
		}
		item := &widget.ListItem{
			Headline: widget.Tu(tagx.Name),
		}
		if supporting != "" {
			item.SupportingText = widget.Tu(supporting)
		}
		items = append(items, item)
	}

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: "trashFileTags",
		},
		GapY: true,
		Children: &widget.List{
			Children: items,
		},
		MarginY: true,
	}
}
