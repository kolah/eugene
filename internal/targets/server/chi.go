package server

type ChiFramework struct{}

func (f *ChiFramework) Name() string {
	return "chi"
}

func (f *ChiFramework) TemplateName() string {
	return "go/server/chi.tmpl"
}

func (f *ChiFramework) ConvertPath(openAPIPath string) string {
	// Chi uses same syntax as OpenAPI: /pets/{petId}
	return openAPIPath
}
