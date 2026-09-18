package widget

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestButtonHTMXOnEventsRenderAsAttributes(t *testing.T) {
	tests := []struct {
		name  string
		event string
		want  string
	}{
		{
			name:  "click",
			event: "click",
			want:  "hx-on:click",
		},
		{
			name:  "change",
			event: "change",
			want:  "hx-on:change",
		},
		{
			name:  "shorthand after request",
			event: ":after-request",
			want:  "hx-on::after-request",
		},
		{
			name:  "full after request",
			event: "htmx:after-request",
			want:  "hx-on:htmx:after-request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			document := renderButtonHTML(t, &Button{
				HTMXAttrs: HTMXAttrs{
					HxOn: &HxOn{
						Event:   tt.event,
						Handler: `handle(event)`,
					},
				},
			})
			button := buttonNode(t, document)

			if hasZgotmplZ(button) {
				t.Fatalf("event %q rendered as ZgotmplZ", tt.event)
			}
			if got := attribute(button, tt.want); got != "handle(event)" {
				t.Errorf("%s = %q, want %q", tt.want, got, "handle(event)")
			}
		})
	}
}

func TestButtonHTMXOnRejectsInvalidOrEmptyEventNames(t *testing.T) {
	for _, event := range []string{
		`click" onclick="injected`,
		"click something",
		"",
	} {
		t.Run(event, func(t *testing.T) {
			document := renderButtonHTML(t, &Button{
				HTMXAttrs: HTMXAttrs{
					HxOn: &HxOn{
						Event:   event,
						Handler: `injected()`,
					},
				},
			})
			button := buttonNode(t, document)

			if hasZgotmplZ(button) {
				t.Fatalf("invalid event %q rendered as ZgotmplZ", event)
			}
			for _, attr := range button.Attr {
				if strings.HasPrefix(attr.Key, "hx-on:") || attr.Key == "onclick" {
					t.Errorf("unexpected event attribute %q=%q for event %q", attr.Key, attr.Val, event)
				}
			}
		})
	}
}

func TestButtonHTMXOnHandlerRemainsOneEscapedAttribute(t *testing.T) {
	const handler = `if (value < 2 && quote === "x") { alert("<markup>"); }`
	document := renderButtonHTML(t, &Button{
		HTMXAttrs: HTMXAttrs{
			HxOn: &HxOn{
				Event:   "click",
				Handler: template.JS(handler),
			},
		},
	})
	button := buttonNode(t, document)

	if got := attribute(button, "hx-on:click"); got != handler {
		t.Errorf("handler = %q, want %q", got, handler)
	}
	if count := countAttributes(button, "hx-on:"); count != 1 {
		t.Errorf("rendered %d hx-on attributes, want one", count)
	}
}

func renderButtonHTML(t *testing.T, button *Button) *html.Node {
	t.Helper()
	tmpl, err := template.New("button").Funcs(template.FuncMap{
		"render": func(_ any, _ any) string { return "" },
	}).ParseFiles("button.gohtml", "htmx_attrs.gohtml")
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := tmpl.ExecuteTemplate(&output, "Button", button); err != nil {
		t.Fatal(err)
	}
	document, err := html.Parse(strings.NewReader(output.String()))
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func buttonNode(t *testing.T, document *html.Node) *html.Node {
	t.Helper()
	var button *html.Node
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "button" {
			button = node
			return
		}
		for child := node.FirstChild; child != nil && button == nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(document)
	if button == nil {
		t.Fatal("rendered document has no button")
	}
	return button
}

func attribute(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

func countAttributes(node *html.Node, prefix string) int {
	count := 0
	for _, attr := range node.Attr {
		if strings.HasPrefix(attr.Key, prefix) {
			count++
		}
	}
	return count
}

func hasZgotmplZ(node *html.Node) bool {
	for _, attr := range node.Attr {
		if strings.Contains(strings.ToLower(attr.Key), "zgotmplz") ||
			strings.Contains(strings.ToLower(attr.Val), "zgotmplz") {
			return true
		}
	}
	return false
}
