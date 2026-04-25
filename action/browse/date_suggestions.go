package browse

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/simpledms/simpledms/core/ui/widget"
	timex2 "github.com/simpledms/simpledms/core/util/timex"
	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
)

type DateSuggestionsWidget struct {
	filex      *filemodel.File
	fieldID    string
	propertyID int64
}

func filePropertyFieldID(propertyID int64) string {
	return fmt.Sprintf("file-property-%d", propertyID)
}

func NewDateSuggestionsWidget(filex *filemodel.File, fieldID string, propertyID int64) *DateSuggestionsWidget {
	return &DateSuggestionsWidget{
		filex:      filex,
		fieldID:    fieldID,
		propertyID: propertyID,
	}
}

func (qq *DateSuggestionsWidget) suggestionsID() string {
	return fmt.Sprintf("file-property-date-suggestions-%d", qq.propertyID)
}

func (qq *DateSuggestionsWidget) suggestionsFromFile() []timex2.Date {
	content := strings.TrimSpace(qq.filex.Data.Name)
	if qq.filex.Data.OcrContent != "" {
		if content == "" {
			content = qq.filex.Data.OcrContent
		} else {
			content = content + "\n" + qq.filex.Data.OcrContent
		}
	}

	return timex2.SuggestDatesFromText(content)
}

func (qq *DateSuggestionsWidget) suggestionChips(ctx ctxx.Context, suggestions []timex2.Date) []widget.IWidget {
	chips := make([]widget.IWidget, 0, len(suggestions))
	for _, suggestion := range suggestions {
		label := suggestion.String(ctx.MainCtx().LanguageBCP47)
		chips = append(chips, &widget.AssistChip{
			Label:       widget.Tu(label),
			LeadingIcon: "event",
			HTMXAttrs: widget.HTMXAttrs{
				HxOn: &widget.HxOn{
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

func (qq *DateSuggestionsWidget) Widget(ctx ctxx.Context, showSuggestions bool, swapOOB string) *widget.Container {
	var child widget.IWidget
	if showSuggestions {
		suggestions := qq.suggestionsFromFile()
		if len(suggestions) > 0 {
			child = qq.suggestionChips(ctx, suggestions)
		}
	}

	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: qq.suggestionsID(),
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxSwapOOB: swapOOB,
		},
		Gap:   true,
		Child: child,
	}
}
