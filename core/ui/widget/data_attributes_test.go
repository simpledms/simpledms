package widget

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

// Each value renders as a data attribute that scripts read through dataset.
func TestDataAttributesRenderEachValueAsDataAttribute(t *testing.T) {
	tmpl, err := template.New("data").Funcs(template.FuncMap{
		"render": func(_ any, _ *Text) string { return `Copied "codes" & more` },
	}).ParseFiles("data_attributes.gohtml")
	if err != nil {
		t.Fatal(err)
	}
	attributes := &DataAttributes{
		Widget: Widget[DataAttributes]{ID: "messages"},
		Values: map[string]*Text{"copied-message": Tu("ignored")},
	}
	var output bytes.Buffer
	if err := tmpl.ExecuteTemplate(&output, "DataAttributes", attributes); err != nil {
		t.Fatal(err)
	}

	want := `data-copied-message="Copied &#34;codes&#34; &amp; more"`
	if !strings.Contains(output.String(), want) {
		t.Fatalf("rendered data attributes lack %s:\n%s", want, output.String())
	}
}

// Only valid data attribute names render; anything else could break out of the tag.
func TestDataAttributesRejectInvalidNames(t *testing.T) {
	for _, name := range []string{"", "Title", "a b", `x" onclick="alert(1)`, "1st", "a_b"} {
		attributes := &DataAttributes{Values: map[string]*Text{name: Tu("v")}}
		if _, err := attributes.Attributes(); err == nil {
			t.Fatalf("data attribute name %q was accepted", name)
		}
	}
}
