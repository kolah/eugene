package spec

import (
	"encoding/base64"

	"github.com/kolah/eugene/internal/templates"
)

type Target struct{}

func New() *Target {
	return &Target{}
}

type templateData struct {
	Package  string
	SpecData string
}

func (t *Target) Generate(engine templates.Engine, specData []byte, pkg string) (string, error) {
	data := templateData{
		Package:  pkg,
		SpecData: base64.StdEncoding.EncodeToString(specData),
	}

	return engine.Execute("go/spec.tmpl", data)
}
