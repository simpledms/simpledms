package browse

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/timex"
)

type DateSuggestionsWidget struct {
	fileName   string
	ocrContent string
	fieldID    string
	propertyID int64
}

func filePropertyFieldID(propertyID int64) string {
	return fmt.Sprintf("file-property-%d", propertyID)
}

func NewDateSuggestionsWidget(
	fileName string,
	ocrContent string,
	fieldID string,
	propertyID int64,
) *DateSuggestionsWidget {
	return &DateSuggestionsWidget{
		fileName:   fileName,
		ocrContent: ocrContent,
		fieldID:    fieldID,
		propertyID: propertyID,
	}
}

func (qq *DateSuggestionsWidget) suggestionsID() string {
	return fmt.Sprintf("file-property-date-suggestions-%d", qq.propertyID)
}

func (qq *DateSuggestionsWidget) suggestionsFromFile() []timex.Date {
	content := strings.TrimSpace(qq.fileName)
	if qq.ocrContent != "" {
		if content == "" {
			content = qq.ocrContent
		} else {
			content = content + "\n" + qq.ocrContent
		}
	}

	return timex.SuggestDatesFromText(content)
}

func (qq *DateSuggestionsWidget) suggestionChips(ctx ctxx.Context, suggestions []timex.Date) []wx.IWidget {
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
						// safe, not user input:
						qq.fieldID,
						// safe, strictly formatted user input:
						suggestion.Format("2006-01-02"),
					)),
				},
			},
		})
	}

	return chips
}

func (qq *DateSuggestionsWidget) Widget(ctx ctxx.Context, showSuggestions bool, swapOOB string) *wx.Container {
	var child wx.IWidget
	if showSuggestions {
		suggestions := qq.suggestionsFromFile()
		if len(suggestions) > 0 {
			child = qq.suggestionChips(ctx, suggestions)
		}
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: qq.suggestionsID(),
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxSwapOOB: swapOOB,
		},
		Gap:   true,
		Child: child,
	}
}
