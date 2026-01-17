package server

import "strings"

type EchoFramework struct{}

func (f *EchoFramework) Name() string {
	return "echo"
}

func (f *EchoFramework) TemplateName() string {
	return "go/server/echo.tmpl"
}

func (f *EchoFramework) ConvertPath(openAPIPath string) string {
	// OpenAPI: /pets/{petId} -> Echo: /pets/:petId
	result := openAPIPath
	result = strings.ReplaceAll(result, "{", ":")
	result = strings.ReplaceAll(result, "}", "")
	return result
}
