package staticpage

import "embed"

//go:embed content/*.md
var contentFS embed.FS
