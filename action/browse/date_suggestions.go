package browse

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/timex"
)

func filePropertyFieldID(propertyID int64) string {
	return fmt.Sprintf("file-property-%d", propertyID)
}

func filePropertyDateSuggestionsID(propertyID int64) string {
	return fmt.Sprintf("file-property-date-suggestions-%d", propertyID)
}

func dateSuggestionsFromFile(filex *model.File) []timex.Date {
	content := strings.TrimSpace(filex.Data.Name)
	if filex.Data.OcrContent != "" {
		if content == "" {
			content = filex.Data.OcrContent
		} else {
			content = content + "\n" + filex.Data.OcrContent
		}
	}

	return timex.SuggestDatesFromText(content)
}

func dateSuggestionChips(ctx ctxx.Context, fieldID string, suggestions []timex.Date) []wx.IWidget {
	chips := make([]wx.IWidget, 0, len(suggestions))
	for _, suggestion := range suggestions {
		label := suggestion.String(ctx.MainCtx().LanguageBCP47)
		chips = append(chips, &wx.AssistChip{
			Label:       wx.Tu(label),
			LeadingIcon: "event",
			HTMXAttrs: wx.HTMXAttrs{
				HxOn: &wx.HxOn{
					Event: "click",
					Handler: template.JS(fmt.Sprintf(
						"const el = document.getElementById('%s'); if (el) { el.value='%s'; el.dispatchEvent(new Event('change', { bubbles:true })); }",
						fieldID,
						suggestion.Format("2006-01-02"),
					)),
				},
			},
		})
	}

	return chips
}

func dateSuggestionsContainer(
	ctx ctxx.Context,
	filex *model.File,
	fieldID string,
	propertyID int64,
	showSuggestions bool,
	swapOOB string,
) *wx.Container {
	var child wx.IWidget
	if showSuggestions {
		suggestions := dateSuggestionsFromFile(filex)
		if len(suggestions) > 0 {
			child = dateSuggestionChips(ctx, fieldID, suggestions)
		}
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: filePropertyDateSuggestionsID(propertyID),
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxSwapOOB: swapOOB,
		},
		Gap:   true,
		Child: child,
	}
}
