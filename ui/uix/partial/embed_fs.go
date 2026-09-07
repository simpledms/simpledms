package partial

import "embed"

// TemplateFS contains the application partial templates.
//
//go:embed *.gohtml
var TemplateFS embed.FS
