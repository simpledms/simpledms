package openfile

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"unicode"

	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	temporaryfilemodel "github.com/simpledms/simpledms/model/main/temporaryfile"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type UploadFromURLPageState struct {
	URL            string `url:"url"`
	Source         string `url:"source"`
	CallbackOrigin string `url:"callback_origin"`
	PermissionID   string `url:"permission_id"`
	FilePath       string `url:"file_path"`
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
	source := strings.TrimSpace(state.Source)
	normalizedURL, err := qq.uploadFromURLService.ValidateURLForSource(rawURL, source)
	if err != nil {
		return err
	}
	callbackOrigin := ""
	permissionID := ""
	fileName := ""
	filePath := ""
	if source == temporaryfilemodel.OpenCloudURLSource {
		callbackOrigin, err = qq.uploadFromURLService.ValidateOpenCloudCallbackOrigin(state.CallbackOrigin)
		if err != nil {
			return err
		}
		permissionID = strings.TrimSpace(state.PermissionID)
		if permissionID == "" || len(permissionID) > 1024 {
			return e.NewHTTPErrorf(http.StatusBadRequest, "Invalid OpenCloud permission ID.")
		}

		parsedURL, err := url.Parse(normalizedURL)
		if err != nil {
			return e.NewHTTPErrorf(http.StatusBadRequest, "Invalid OpenCloud public link.")
		}
		fileName = path.Base(parsedURL.Path)
		filePath = strings.TrimSpace(state.FilePath)
		if len(filePath) > 4096 || strings.IndexFunc(filePath, unicode.IsControl) >= 0 ||
			path.Base(strings.ReplaceAll(filePath, "\\", "/")) != fileName {
			filePath = ""
		}
	}

	if source == temporaryfilemodel.OpenCloudURLSource {
		return qq.Render(
			rw,
			req,
			ctx,
			qq.infra,
			"Import file",
			qq.Widget(ctx, normalizedURL, source, callbackOrigin, permissionID, fileName, filePath),
		)
	}

	return qq.Render(
		rw,
		req,
		ctx,
		qq.infra,
		"Import URL",
		qq.Widget(ctx, normalizedURL, source, callbackOrigin, permissionID, fileName, filePath),
	)
}

func (qq *UploadFromURLPage) Widget(
	ctx ctxx.Context,
	rawURL string,
	source string,
	callbackOrigin string,
	permissionID string,
	fileName string,
	filePath string,
) renderable.Renderable {
	vals, err := json.Marshal(map[string]string{
		"url":             rawURL,
		"source":          source,
		"callback_origin": callbackOrigin,
		"permission_id":   permissionID,
	})
	if err != nil {
		log.Println(err)
		vals = []byte("{}")
	}

	description := widget.Tuf("URL: %s", rawURL)
	headline := widget.T("Import file from URL")
	var afterRequest *widget.HxOn
	if source == temporaryfilemodel.OpenCloudURLSource {
		headline = widget.Tuf("%s", fileName)
		description = widget.T("A file shared from OpenCloud is ready to import.")
		if filePath != "" {
			description = widget.Tuf("%s", filePath)
		}
		message, _ := json.Marshal(map[string]string{
			"type":         "simpledms:opencloud-import-staged",
			"permissionId": permissionID,
		})
		targetOrigin, _ := json.Marshal(callbackOrigin)
		afterRequest = &widget.HxOn{
			Event: ":after-request",
			// HX-Location is handled before HTMX sets event.detail.successful.
			// Values are JSON encoded before being placed in this trusted handler.
			Handler: template.JS(fmt.Sprintf( // #nosec G203
				`const message = %s;
				const targetOrigin = %s;
				if (event.detail.xhr.status === 200 &&
					event.detail.xhr.getResponseHeader('HX-Location') && window.opener) {
					try {
						window.opener.postMessage(message, targetOrigin);
					} catch (error) {
						console.error('[OpenCloud import] callback post failed', {
							permissionId: message.permissionId,
							errorName: error?.name || typeof error
						});
						throw error;
					}
				}`,
				message,
				targetOrigin,
			)),
		}
	}

	return &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "upload", nil),
		Content: &widget.ListDetailLayout{
			AppBar: qq.appBar(ctx, source),
			List: []widget.IWidget{
				&widget.EmptyState{
					Icon:                    widget.NewIcon("upload"),
					Headline:                headline,
					Description:             description,
					WrapDescriptionAnywhere: true,
					Actions: []widget.IWidget{
						&widget.Button{
							Label:     widget.T("Download and continue"),
							StyleType: widget.ButtonStyleTypeTonal,
							HTMXAttrs: widget.HTMXAttrs{
								HxPost: qq.actions.UploadFromURLCmd.Endpoint(),
								HxVals: template.JS(vals),
								HxOn:   afterRequest,
							},
						},
					},
				},
			},
		},
	}
}

func (qq *UploadFromURLPage) appBar(ctx ctxx.Context, source string) *widget.AppBar {
	title := widget.T("Import URL")
	if source == temporaryfilemodel.OpenCloudURLSource {
		title = widget.T("Import file")
	}

	return &widget.AppBar{
		Leading: &widget.Icon{
			Name: "upload",
		},
		LeadingAltMobile: partial2.NewNavigationRailToggle(),
		Title: &widget.AppBarTitle{
			Text: title,
		},
		Actions: []widget.IWidget{},
	}
}
