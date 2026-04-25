package openfile

import (
	"encoding/json"
	"html/template"
	"log"
	"strings"

	autil "github.com/simpledms/simpledms/core/action/util"

	acommon "github.com/simpledms/simpledms/core/action/common"
	"github.com/simpledms/simpledms/core/common"
	temporaryfilemodel "github.com/simpledms/simpledms/core/model/temporaryfile"
	"github.com/simpledms/simpledms/core/ui/renderable"
	"github.com/simpledms/simpledms/core/ui/widget"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
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

func (qq *UploadFromURLPage) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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
		LeadingAltMobile: partial2.NewMainMenu(ctx, qq.infra),
		Title: &widget.AppBarTitle{
			Text: widget.T("Import URL"),
		},
		Actions: []widget.IWidget{},
	}
}
