package templates

import "embed"

//go:embed go/*.tmpl go/server/*.tmpl
var FS embed.FS
