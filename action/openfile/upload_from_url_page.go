package openfile

import (
	"encoding/json"
	"html/template"
	"log"
	"strings"

	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	temporaryfilemodel "github.com/simpledms/simpledms/model/main/temporaryfile"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/httpx"
)

type UploadFromURLPageState struct {
	URL string `url:"url"`
}

type UploadFromURLPage struct {
	acommon.Page
	infra                *common.Infra
	actions              *Actions
	uploadFromURLService *temporaryfilemodel.UploadFromURLService
}

func NewUploadFromURLPage(
	infra *common.Infra,
	actions *Actions,
	uploadFromURLService *temporaryfilemodel.UploadFromURLService,
) *UploadFromURLPage {
	return &UploadFromURLPage{
		infra:                infra,
		actions:              actions,
		uploadFromURLService: uploadFromURLService,
	}
}

func (qq *UploadFromURLPage) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	state := autil.StateX[UploadFromURLPageState](rw, req)

	rawURL := strings.TrimSpace(state.URL)
	normalizedURL, err := qq.uploadFromURLService.ValidateURL(rawURL)
	if err != nil {
		return err
	}

	return qq.Render(rw, req, ctx, qq.infra, "Import URL", qq.Widget(ctx, normalizedURL))
}

func (qq *UploadFromURLPage) Widget(ctx ctxx.Context, rawURL string) renderable.Renderable {
	vals, err := json.Marshal(map[string]string{
		"url": rawURL,
	})
	if err != nil {
		log.Println(err)
		vals = []byte("{}")
	}

	return &wx.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "upload", nil),
		Content: &wx.ListDetailLayout{
			AppBar: qq.appBar(ctx),
			List: []wx.IWidget{
				&wx.EmptyState{
					Icon:        wx.NewIcon("upload"),
					Headline:    wx.T("Import file from URL"),
					Description: wx.Tuf("URL: %s", rawURL),
					Actions: []wx.IWidget{
						&wx.Button{
							Label:     wx.T("Download and continue"),
							StyleType: wx.ButtonStyleTypeTonal,
							HTMXAttrs: wx.HTMXAttrs{
								HxPost: qq.actions.UploadFromURLCmd.Endpoint(),
								HxVals: template.JS(vals),
							},
						},
					},
				},
			},
		},
	}
}

func (qq *UploadFromURLPage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "upload",
		},
		LeadingAltMobile: partial2.NewMainMenu(ctx, qq.infra),
		Title: &wx.AppBarTitle{
			Text: wx.T("Import URL"),
		},
		Actions: []wx.IWidget{},
	}
}
