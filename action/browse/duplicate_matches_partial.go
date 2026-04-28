package browse

import (
	"log"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
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
) (*wx.ScrollableContent, error) {
	widget, _, _, err := qq.WidgetWithStatus(ctx, data)
	return widget, err
}

func (qq *DuplicateMatchesPartial) WidgetWithStatus(
	ctx ctxx.Context,
	data *DuplicateMatchesPartialData,
) (*wx.ScrollableContent, *wx.Text, bool, error) {
	content, statusMessage, hasDuplicates, err := qq.contentWithStatus(ctx, data)
	if err != nil {
		return nil, nil, false, err
	}

	return &wx.ScrollableContent{
		MarginY:  true,
		FlexCol:  true,
		Children: content,
	}, statusMessage, hasDuplicates, nil
}

func (qq *DuplicateMatchesPartial) contentWithStatus(
	ctx ctxx.Context,
	data *DuplicateMatchesPartialData,
) (renderable.Renderable, *wx.Text, bool, error) {
	result, err := qq.service.FindDuplicates(ctx, data.FileID)
	if err != nil {
		log.Println(err)
		return nil, nil, false, err
	}

	if !result.HasContentHash {
		return wx.NewBody(
			wx.BodyTypeMd,
			wx.T("Duplicate check is still being prepared for this file."),
		), wx.T("Duplicate check is still being prepared for this file."), false, nil
	}
	if len(result.Matches) == 0 {
		return wx.NewBody(wx.BodyTypeMd, wx.T("No duplicates found.")), nil, false, nil
	}

	listItems := make([]wx.IWidget, 0, len(result.Matches))
	for _, match := range result.Matches {
		listItems = append(listItems, qq.matchListItem(ctx, match))
	}

	return &wx.Column{
		GapYSize:   wx.Gap2,
		AutoHeight: true,
		Children: []wx.IWidget{
			&wx.Label{
				Text: wx.T("Duplicates found"),
				Type: wx.LabelTypeLg,
			},
			wx.NewBody(
				wx.BodyTypeMd,
				wx.Tf(
					"This file already exists in the following %d locations:",
					len(result.Matches),
				),
			),
			&wx.List{
				Children: listItems,
			},
		},
	}, nil, true, nil
}

func (qq *DuplicateMatchesPartial) matchListItem(
	ctx ctxx.Context,
	match *filemodel.DuplicateMatch,
) *wx.ListItem {
	href := route.BrowseFile(
		match.TenantPublicID,
		match.SpacePublicID,
		match.ParentDirPublicID,
		match.FilePublicID,
	)

	return &wx.ListItem{
		Href:           href,
		Headline:       wx.Tu(match.FileName),
		SupportingText: qq.matchSupportingText(ctx, match),
	}
}

func (qq *DuplicateMatchesPartial) matchSupportingText(
	ctx ctxx.Context,
	match *filemodel.DuplicateMatch,
) *wx.Text {
	parts := []string{
		wx.Tf("Space: %s", wx.Tu(match.SpaceName)).String(ctx),
	}
	if !match.ParentDirIsRoot {
		parts = append(parts, wx.Tf("Folder: %s", wx.Tu(match.ParentDirName)).String(ctx))
	}
	parts = append(
		parts,
		qq.versionLabel(ctx, match),
		wx.Tf(
			"Uploaded %s",
			timex.NewDateTime(match.UploadedAt).String(ctx.MainCtx().LanguageBCP47),
		).String(ctx),
		fileutil.FormatSize(match.Size),
	)

	return wx.Tu(strings.Join(parts, " - "))
}

func (qq *DuplicateMatchesPartial) versionLabel(ctx ctxx.Context, match *filemodel.DuplicateMatch) string {
	if match.IsCurrentVersion {
		return wx.Tf("Current version %d", match.VersionNumber).String(ctx)
	}
	return wx.Tf("Version %d", match.VersionNumber).String(ctx)
}
