package golang

import (
	"golang.org/x/tools/imports"
)

func Format(src []byte) ([]byte, error) {
	return imports.Process("", src, &imports.Options{
		Comments:   true,
		TabIndent:  true,
		TabWidth:   8,
		FormatOnly: false,
	})
}
