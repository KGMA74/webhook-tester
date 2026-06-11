package web

import "embed"

// TemplateFS holds the embedded HTML templates, compiled into the binary.
//
//go:embed template
var TemplateFS embed.FS
