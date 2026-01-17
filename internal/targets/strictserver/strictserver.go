package strictserver

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kolah/eugene/internal/golang"
	"github.com/kolah/eugene/internal/model"
	"github.com/kolah/eugene/internal/templates"
)

type Framework interface {
	Name() string
	TypesTemplateName() string
	AdapterTemplateName() string
	ConvertPath(openAPIPath string) string
}

type Target struct {
	framework Framework
}

func New(frameworkName string) (*Target, error) {
	var fw Framework
	switch frameworkName {
	case "echo":
		fw = &EchoFramework{}
	case "chi":
		fw = &ChiFramework{}
	case "stdlib":
		fw = &StdlibFramework{}
	default:
		return nil, fmt.Errorf("unsupported server framework: %s", frameworkName)
	}
	return &Target{framework: fw}, nil
}

func (t *Target) Name() string {
	return "strict-server"
}

func (t *Target) FrameworkName() string {
	return t.framework.Name()
}

type templateData struct {
	Package       string
	Operations    []operationData
	Framework     string
	HasQueryParams bool
}

type operationData struct {
	ID           string
	Method       string
	Path         string
	FramePath    string
	Summary      string
	Description  string
	Tags         []string
	PathParams   []parameterData
	QueryParams  []parameterData
	HeaderParams []parameterData
	RequestBody  *requestBodyData
	Responses    []responseData
	IsStreaming  bool
}

type parameterData struct {
	Name     string
	GoName   string
	Type     string
	Required bool
}

type requestBodyData struct {
	Required bool
	Type     string
}

type responseData struct {
	StatusCode  string
	Description string
	Type        string
}

func (t *Target) GenerateTypes(engine templates.Engine, spec *model.Spec, pkg string) (string, error) {
	data := t.buildTemplateData(spec, pkg)
	return engine.Execute(t.framework.TypesTemplateName(), data)
}

func (t *Target) GenerateAdapter(engine templates.Engine, spec *model.Spec, pkg string) (string, error) {
	data := t.buildTemplateData(spec, pkg)
	return engine.Execute(t.framework.AdapterTemplateName(), data)
}

func (t *Target) buildTemplateData(spec *model.Spec, pkg string) templateData {
	var ops []operationData
	hasQueryParams := false

	for _, op := range spec.Operations {
		opData := operationData{
			ID:          golang.PascalCase(op.ID),
			Method:      string(op.Method),
			Path:        op.Path,
			FramePath:   t.framework.ConvertPath(op.Path),
			Summary:     op.Summary,
			Description: op.Description,
			Tags:        op.Tags,
			IsStreaming: op.Streaming != nil,
		}

		for _, p := range op.Parameters {
			pd := parameterData{
				Name:     p.Name,
				GoName:   golang.PascalCase(p.Name),
				Type:     schemaToGoType(p.Schema),
				Required: p.Required,
			}

			switch p.In {
			case model.LocationPath:
				opData.PathParams = append(opData.PathParams, pd)
			case model.LocationQuery:
				opData.QueryParams = append(opData.QueryParams, pd)
				hasQueryParams = true
			case model.LocationHeader:
				opData.HeaderParams = append(opData.HeaderParams, pd)
			}
		}

		if op.RequestBody != nil {
			rb := &requestBodyData{Required: op.RequestBody.Required}
			if len(op.RequestBody.Content) > 0 {
				rb.Type = schemaToGoType(op.RequestBody.Content[0].Schema)
			}
			opData.RequestBody = rb
		}

		for _, r := range op.Responses {
			rd := responseData{
				StatusCode:  r.StatusCode,
				Description: r.Description,
			}
			if len(r.Content) > 0 {
				rd.Type = schemaToGoType(r.Content[0].Schema)
			}
			opData.Responses = append(opData.Responses, rd)
		}

		ops = append(ops, opData)
	}

	return templateData{
		Package:       pkg,
		Operations:    ops,
		Framework:     t.framework.Name(),
		HasQueryParams: hasQueryParams,
	}
}


func schemaToGoType(s *model.Schema) string {
	if s == nil {
		return "any"
	}
	if s.Ref != "" {
		parts := splitRef(s.Ref)
		if len(parts) > 0 {
			return golang.PascalCase(parts[len(parts)-1])
		}
	}
	switch s.Type {
	case model.TypeString:
		return "string"
	case model.TypeInteger:
		if s.Format == "int64" {
			return "int64"
		}
		if s.Format == "int32" {
			return "int32"
		}
		return "int"
	case model.TypeNumber:
		return "float64"
	case model.TypeBoolean:
		return "bool"
	case model.TypeArray:
		return "[]" + schemaToGoType(s.Items)
	default:
		return "any"
	}
}

func splitRef(ref string) []string {
	var parts []string
	current := ""
	for _, c := range ref {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// statusCodeInt converts a status code string to int for template use
func StatusCodeInt(code string) int {
	if code == "default" {
		return 500
	}
	n, err := strconv.Atoi(code)
	if err != nil {
		return 500
	}
	return n
}

// Echo Framework
type EchoFramework struct{}

func (f *EchoFramework) Name() string                      { return "echo" }
func (f *EchoFramework) TypesTemplateName() string         { return "go/strict_types.tmpl" }
func (f *EchoFramework) AdapterTemplateName() string       { return "go/server/strict_echo.tmpl" }
func (f *EchoFramework) ConvertPath(path string) string {
	// Convert {id} to :id
	var result strings.Builder
	for _, c := range path {
		if c == '{' {
			result.WriteRune(':')
		} else if c == '}' {
			// skip closing brace
		} else {
			result.WriteRune(c)
		}
	}
	return result.String()
}

// Chi Framework
type ChiFramework struct{}

func (f *ChiFramework) Name() string                      { return "chi" }
func (f *ChiFramework) TypesTemplateName() string         { return "go/strict_types.tmpl" }
func (f *ChiFramework) AdapterTemplateName() string       { return "go/server/strict_chi.tmpl" }
func (f *ChiFramework) ConvertPath(path string) string    { return path } // Chi uses {id} syntax

// Stdlib Framework
type StdlibFramework struct{}

func (f *StdlibFramework) Name() string                      { return "stdlib" }
func (f *StdlibFramework) TypesTemplateName() string         { return "go/strict_types.tmpl" }
func (f *StdlibFramework) AdapterTemplateName() string       { return "go/server/strict_stdlib.tmpl" }
func (f *StdlibFramework) ConvertPath(path string) string    { return path } // stdlib uses {id} syntax
