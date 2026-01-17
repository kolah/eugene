package server

type StdlibFramework struct{}

func (f *StdlibFramework) Name() string {
	return "stdlib"
}

func (f *StdlibFramework) TemplateName() string {
	return "go/server/stdlib.tmpl"
}

func (f *StdlibFramework) ConvertPath(openAPIPath string) string {
	// Go 1.22+ net/http uses same syntax as OpenAPI: /pets/{petId}
	return openAPIPath
}
