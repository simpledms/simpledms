package widget

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

func TestTextAreaLabelLayout(t *testing.T) {
	tmpl, err := template.New("textarea").Funcs(template.FuncMap{
		"render": func(_ any, _ *Text) string { return "Note" },
	}).ParseFiles("text_area.gohtml")
	if err != nil {
		t.Fatal(err)
	}
	for _, labeled := range []bool{false, true} {
		area := &TextArea{
			Name:         "Body",
			Value:        "Existing <note>",
			IsReadonly:   true,
			HasAutofocus: true,
		}
		if labeled {
			area.Label = Tu("Note")
		}
		var output bytes.Buffer
		if err := tmpl.ExecuteTemplate(&output, "TextArea", area); err != nil {
			t.Fatal(err)
		}
		html := output.String()
		for _, want := range []string{
			`name="Body"`, `rows="6"`, "readonly", "autofocus", "Existing &lt;note&gt;",
		} {
			if !strings.Contains(html, want) {
				t.Errorf("labeled=%t: missing %q", labeled, want)
			}
		}
		for _, marker := range []string{
			`<label`, `placeholder=" "`, `peer-placeholder-shown:top-0`, `peer-focus:-top-6`,
		} {
			if strings.Contains(html, marker) != labeled {
				t.Errorf("labeled=%t: unexpected presence of %q", labeled, marker)
			}
		}
		if labeled && strings.Index(html, "</textarea>") > strings.Index(html, "<span") {
			t.Error("floating label must follow the textarea for peer selectors")
		}
		if !labeled && !strings.Contains(html, area.GetClass()) {
			t.Error("unlabeled textarea styling changed")
		}
	}
}
