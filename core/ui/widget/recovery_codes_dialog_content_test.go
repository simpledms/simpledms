package widget

import (
	"bytes"
	"html/template"
	"regexp"
	"strings"
	"testing"
)

// htmxHeadPattern is the raw-text pattern htmx strips from swapped responses before parsing.
var htmxHeadPattern = regexp.MustCompile(`(?i)<head(\s[^>]*)?>[\s\S]*?</head>`)

// The backup codes dialog is swapped in by htmx, which cuts anything resembling a <head> out of
// the response text, scripts included. The print, copy, and download script must survive intact.
func TestRecoveryCodesDialogSurvivesHTMXHeadExtraction(t *testing.T) {
	tmpl, err := template.New("dialog").Funcs(template.FuncMap{
		"render": func(_ any, _ any) string { return "label" },
	}).ParseFiles("recovery_codes_dialog_content.gohtml")
	if err != nil {
		t.Fatal(err)
	}
	content := &RecoveryCodesDialogContent{
		Widget: Widget[RecoveryCodesDialogContent]{ID: "codes"}, CodesText: "AAAA-BBBB",
	}
	var output bytes.Buffer
	if err := tmpl.ExecuteTemplate(&output, "RecoveryCodesDialogContent", content); err != nil {
		t.Fatal(err)
	}

	if head := htmxHeadPattern.FindString(output.String()); head != "" {
		t.Fatalf("htmx would strip this from the dialog response:\n%s", head)
	}
}

// The script takes its translated labels from the rendered data attributes only.
func TestRecoveryCodesDialogRendersEveryLabelItsScriptReads(t *testing.T) {
	var tmpl *template.Template
	tmpl = template.Must(template.New("dialog").Funcs(template.FuncMap{
		"render": func(_ any, child any) (template.HTML, error) {
			attributes, isDataAttributes := child.(*DataAttributes)
			if !isDataAttributes {
				return "label", nil
			}
			var output bytes.Buffer
			err := tmpl.ExecuteTemplate(&output, "DataAttributes", attributes)
			return template.HTML(output.String()), err
		},
	}).ParseFiles("recovery_codes_dialog_content.gohtml", "data_attributes.gohtml"))
	content := &RecoveryCodesDialogContent{
		Widget: Widget[RecoveryCodesDialogContent]{ID: "codes"}, CodesText: "AAAA-BBBB",
	}
	var output bytes.Buffer
	if err := tmpl.ExecuteTemplate(&output, "RecoveryCodesDialogContent", content); err != nil {
		t.Fatal(err)
	}
	html := output.String()

	reads := regexp.MustCompile(`messages\.dataset\.([A-Za-z]+)`).FindAllStringSubmatch(html, -1)
	if len(reads) == 0 {
		t.Fatalf("script reads no labels:\n%s", html)
	}
	upper := regexp.MustCompile(`[A-Z]`)
	for _, read := range reads {
		name := "data-" + upper.ReplaceAllStringFunc(read[1], func(s string) string {
			return "-" + strings.ToLower(s)
		})
		if !strings.Contains(html, name+`="label"`) {
			t.Fatalf("script reads dataset.%s, but %s is not rendered:\n%s", read[1], name, html)
		}
	}
}
