package openfile

import (
	"encoding/json"
	"html/template"
	"log"
	"strings"

	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	temporaryfilemodel "github.com/simpledms/simpledms/model/main/temporaryfile"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
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

	return &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "upload", nil),
		Content: &widget.ListDetailLayout{
			AppBar: qq.appBar(ctx),
			List: []widget.IWidget{
				&widget.EmptyState{
					Icon:        widget.NewIcon("upload"),
					Headline:    widget.T("Import file from URL"),
					Description: widget.Tuf("URL: %s", rawURL),
					Actions: []widget.IWidget{
						&widget.Button{
							Label:     widget.T("Download and continue"),
							StyleType: widget.ButtonStyleTypeTonal,
							HTMXAttrs: widget.HTMXAttrs{
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

func (qq *UploadFromURLPage) appBar(ctx ctxx.Context) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.Icon{
			Name: "upload",
		},
		LeadingAltMobile: partial2.NewNavigationRailToggle(),
		Title: &widget.AppBarTitle{
			Text: widget.T("Import URL"),
		},
		Actions: []widget.IWidget{},
	}
}
