package widget

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"strings"
)

// Deprecated: use ui.Renderer instead
func name(qq any) string {
	name := fmt.Sprintf("%T", qq)
	name = strings.TrimPrefix(name, "*widget.")
	return name
}

// Deprecated: use ui.Renderer instead
func renderX(writer io.Writer, templates *template.Template, data any) {
	err := templates.ExecuteTemplate(writer, name(data), data)
	if err != nil {
		// TODO write HTTP error
		log.Println(err)
		panic(err)
	}
}

// Deprecated: use ui.Renderer instead
func render(writer io.Writer, templates *template.Template, data any) error {
	err := templates.ExecuteTemplate(writer, name(data), data)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
