package widget

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

func TestEmptyStateDescriptionWrapping(t *testing.T) {
	tmpl, err := template.New("empty-state").Funcs(template.FuncMap{
		"render": func(_ any, _ any) string { return "Description" },
	}).ParseFiles("empty_state.gohtml")
	if err != nil {
		t.Fatal(err)
	}

	for _, wrapAnywhere := range []bool{false, true} {
		emptyState := &EmptyState{
			Description:             Tu("Description"),
			WrapDescriptionAnywhere: wrapAnywhere,
		}
		var output bytes.Buffer
		if err := tmpl.ExecuteTemplate(&output, "EmptyState", emptyState); err != nil {
			t.Fatal(err)
		}

		hasWrappingStyle := strings.Contains(output.String(), "overflow-wrap: anywhere")
		if hasWrappingStyle != wrapAnywhere {
			t.Errorf("wrapAnywhere=%t: unexpected wrapping style presence", wrapAnywhere)
		}
	}
}
