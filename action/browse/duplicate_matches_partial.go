package browse

import (
	"log"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/fileutil"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/timex"
)

type DuplicateMatchesPartialData struct {
	FileID string
}

type DuplicateMatchesPartial struct {
	infra   *common.Infra
	service *filemodel.DuplicateDetectionService
	*actionx.Config
}

func NewDuplicateMatchesPartial(infra *common.Infra, actions *Actions) *DuplicateMatchesPartial {
	return &DuplicateMatchesPartial{
		infra:   infra,
		service: filemodel.NewDuplicateDetectionService(),
		Config: actionx.NewConfig(
			actions.Route("duplicate-matches-partial"),
			true,
		),
	}
}

func (qq *DuplicateMatchesPartial) Data(fileID string) *DuplicateMatchesPartialData {
	return &DuplicateMatchesPartialData{
		FileID: fileID,
	}
}

func (qq *DuplicateMatchesPartial) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[DuplicateMatchesPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	widget, _, _, err := qq.WidgetWithStatus(ctx, data)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(rw, ctx, widget)
}

func (qq *DuplicateMatchesPartial) Widget(
	ctx ctxx.Context,
	data *DuplicateMatchesPartialData,
) (*widget.ScrollableContent, error) {
	widget, _, _, err := qq.WidgetWithStatus(ctx, data)
	return widget, err
}

func (qq *DuplicateMatchesPartial) WidgetWithStatus(
	ctx ctxx.Context,
	data *DuplicateMatchesPartialData,
) (*widget.ScrollableContent, *widget.Text, bool, error) {
	content, statusMessage, hasDuplicates, err := qq.contentWithStatus(ctx, data)
	if err != nil {
		return nil, nil, false, err
	}

	return &widget.ScrollableContent{
		MarginY:  true,
		FlexCol:  true,
		Children: content,
	}, statusMessage, hasDuplicates, nil
}

func (qq *DuplicateMatchesPartial) contentWithStatus(
	ctx ctxx.Context,
	data *DuplicateMatchesPartialData,
) (renderable.Renderable, *widget.Text, bool, error) {
	result, err := qq.service.FindDuplicates(ctx, data.FileID)
	if err != nil {
		log.Println(err)
		return nil, nil, false, err
	}

	if !result.HasContentHash {
		return widget.NewBody(
			widget.BodyTypeMd,
			widget.T("Duplicate check is still being prepared for this file."),
		), widget.T("Duplicate check is still being prepared for this file."), false, nil
	}
	if len(result.Matches) == 0 {
		return widget.NewBody(widget.BodyTypeMd, widget.T("No duplicates found.")), nil, false, nil
	}

	listItems := make([]widget.IWidget, 0, len(result.Matches))
	for _, match := range result.Matches {
		listItems = append(listItems, qq.matchListItem(ctx, match))
	}

	return &widget.Column{
		GapYSize:   widget.Gap2,
		AutoHeight: true,
		Children: []widget.IWidget{
			&widget.Label{
				Text: widget.T("Duplicates found"),
				Type: widget.LabelTypeLg,
			},
			widget.NewBody(
				widget.BodyTypeMd,
				widget.Tf(
					"This file already exists in the following %d locations:",
					len(result.Matches),
				),
			),
			&widget.List{
				Children: listItems,
			},
		},
	}, nil, true, nil
}

func (qq *DuplicateMatchesPartial) matchListItem(
	ctx ctxx.Context,
	match *filemodel.DuplicateMatch,
) *widget.ListItem {
	return &widget.ListItem{
		Headline:       widget.Tu(match.FileName),
		SupportingText: qq.matchSupportingText(ctx, match),
		HTMXAttrs: widget.HTMXAttrs{
			HxGet: route.BrowseFile(
				match.TenantPublicID,
				match.SpacePublicID,
				match.ParentDirPublicID,
				match.FilePublicID,
			),
		},
	}
}

func (qq *DuplicateMatchesPartial) matchSupportingText(
	ctx ctxx.Context,
	match *filemodel.DuplicateMatch,
) *widget.Text {
	parts := []string{
		widget.Tf("Space: %s", widget.Tu(match.SpaceName)).String(ctx),
	}
	if !match.ParentDirIsRoot {
		parts = append(parts, widget.Tf("Folder: %s", widget.Tu(match.ParentDirName)).String(ctx))
	}
	parts = append(
		parts,
		qq.versionLabel(ctx, match),
		widget.Tf(
			"Uploaded %s",
			timex.NewDateTime(match.UploadedAt).String(ctx.MainCtx().LanguageBCP47),
		).String(ctx),
		fileutil.FormatSize(match.Size),
	)

	return widget.Tu(strings.Join(parts, " - "))
}

func (qq *DuplicateMatchesPartial) versionLabel(ctx ctxx.Context, match *filemodel.DuplicateMatch) string {
	if match.IsCurrentVersion {
		return widget.Tf("Current version %d", match.VersionNumber).String(ctx)
	}
	return widget.Tf("Version %d", match.VersionNumber).String(ctx)
}
