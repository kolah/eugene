package server

import (
	"fmt"

	"github.com/kolah/eugene/internal/model"
	"github.com/kolah/eugene/internal/templates"
)

type Framework interface {
	Name() string
	TemplateName() string
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
	return "server"
}

func (t *Target) FrameworkName() string {
	return t.framework.Name()
}

type serverFeatures struct {
	HasStreaming      bool // any operation uses SSE
	HasQueryString    bool // any operation uses querystring param (OpenAPI 3.2)
	HasCallbacks      bool // any operation defines callbacks
	HasMultipart      bool // any operation uses multipart/form-data
	HasFormUrlEncoded bool // any operation uses application/x-www-form-urlencoded
}

type templateData struct {
	Package    string
	Operations []operationData
	Framework  string
	Tags       []tagData // OpenAPI 3.2: hierarchical tags
	Features   serverFeatures
	Callbacks  []callbackData
}

type callbackData struct {
	Name       string
	GoName     string // PascalCase
	Operations []callbackOperationData
}

type callbackOperationData struct {
	Method      string
	RequestBody *requestBodyData
	Responses   []responseData
}

type tagData struct {
	Name        string
	Description string
	Parent      string // OpenAPI 3.2: parent tag for hierarchy
	Kind        string // OpenAPI 3.2: tag classification
	Children    []string
}

type operationData struct {
	ID               string
	Method           string
	Path             string
	FramePath        string
	Summary          string
	Description      string
	Tags             []string
	Parameters       []parameterData
	QueryString      *querystringData // OpenAPI 3.2: in: querystring
	RequestBody      *requestBodyData
	Responses        []responseData
	Streaming        *streamingData // SSE/streaming
	HasBody          bool
	HasQueryString   bool
	IsStreaming      bool
	IsMultipart      bool
	IsFormUrlEncoded bool
}

type streamingData struct {
	MediaType string
	EventType string
}

type parameterData struct {
	Name        string
	GoName      string
	In          string
	Description string
	Required    bool
	Type        string
}

type querystringData struct {
	Name   string
	GoName string
	Type   string
}

type requestBodyData struct {
	Required        bool
	MediaType       string
	Type            string
	IsMultipart     bool
	IsFormUrlEncoded bool
	MultipartFields []multipartFieldData
}

type multipartFieldData struct {
	Name     string
	GoName   string
	Type     string // "*multipart.FileHeader", "string", "[]string"
	IsFile   bool
	IsArray  bool
	Required bool
}

type responseData struct {
	StatusCode  string
	Description string
	MediaType   string
	Type        string
}

func (t *Target) Generate(engine templates.Engine, spec *model.Spec, pkg string) (string, error) {
	data := templateData{
		Package:   pkg,
		Framework: t.framework.Name(),
	}

	for _, op := range spec.Operations {
		opData := operationData{
			ID:          op.ID,
			Method:      string(op.Method),
			Path:        op.Path,
			FramePath:   t.framework.ConvertPath(op.Path),
			Summary:     op.Summary,
			Description: op.Description,
			Tags:        op.Tags,
			HasBody:     op.RequestBody != nil,
			IsStreaming: op.Streaming != nil,
		}

		if op.Streaming != nil {
			opData.Streaming = &streamingData{
				MediaType: op.Streaming.MediaType,
				EventType: op.Streaming.EventType,
			}
		}

		for _, p := range op.Parameters {
			if p.In == model.LocationQueryString {
				// OpenAPI 3.2: querystring parameter
				opData.QueryString = &querystringData{
					Name:   p.Name,
					GoName: toGoParamName(p.Name),
					Type:   schemaToGoType(p.Schema),
				}
				opData.HasQueryString = true
			} else {
				opData.Parameters = append(opData.Parameters, parameterData{
					Name:        p.Name,
					GoName:      toGoParamName(p.Name),
					In:          string(p.In),
					Description: p.Description,
					Required:    p.Required,
					Type:        schemaToGoType(p.Schema),
				})
			}
		}

		if op.RequestBody != nil {
			rb := &requestBodyData{Required: op.RequestBody.Required}
			if len(op.RequestBody.Content) > 0 {
				content := op.RequestBody.Content[0]
				rb.MediaType = content.MediaType
				rb.Type = schemaToGoType(content.Schema)

				if content.MediaType == "multipart/form-data" {
					rb.IsMultipart = true
					opData.IsMultipart = true
					data.Features.HasMultipart = true
					rb.MultipartFields = extractMultipartFields(content.Schema, op.RequestBody.Required)
				} else if content.MediaType == "application/x-www-form-urlencoded" {
					rb.IsFormUrlEncoded = true
					opData.IsFormUrlEncoded = true
					data.Features.HasFormUrlEncoded = true
					rb.MultipartFields = extractFormUrlEncodedFields(content.Schema, op.RequestBody.Required)
				}
			}
			opData.RequestBody = rb
		}

		for _, r := range op.Responses {
			rd := responseData{
				StatusCode:  r.StatusCode,
				Description: r.Description,
			}
			if len(r.Content) > 0 {
				rd.MediaType = r.Content[0].MediaType
				rd.Type = schemaToGoType(r.Content[0].Schema)
			}
			opData.Responses = append(opData.Responses, rd)
		}

		data.Operations = append(data.Operations, opData)

		// Compute features from operation flags
		if opData.IsStreaming {
			data.Features.HasStreaming = true
		}
		if opData.HasQueryString {
			data.Features.HasQueryString = true
		}

		// Collect callbacks from this operation
		for _, cb := range op.Callbacks {
			cbData := callbackData{
				Name:   cb.Name,
				GoName: toGoParamName(cb.Name),
			}
			for _, cbOp := range cb.Operations {
				cbOpData := callbackOperationData{
					Method: string(cbOp.Method),
				}
				if cbOp.RequestBody != nil && len(cbOp.RequestBody.Content) > 0 {
					cbOpData.RequestBody = &requestBodyData{
						Required:  cbOp.RequestBody.Required,
						MediaType: cbOp.RequestBody.Content[0].MediaType,
						Type:      schemaToGoType(cbOp.RequestBody.Content[0].Schema),
					}
				}
				for _, r := range cbOp.Responses {
					rd := responseData{
						StatusCode:  r.StatusCode,
						Description: r.Description,
					}
					if len(r.Content) > 0 {
						rd.MediaType = r.Content[0].MediaType
						rd.Type = schemaToGoType(r.Content[0].Schema)
					}
					cbOpData.Responses = append(cbOpData.Responses, rd)
				}
				cbData.Operations = append(cbData.Operations, cbOpData)
			}
			data.Callbacks = append(data.Callbacks, cbData)
			data.Features.HasCallbacks = true
		}
	}

	// Build hierarchical tag data
	data.Tags = buildTagData(spec.Tags)

	return engine.Execute(t.framework.TemplateName(), data)
}

func toGoParamName(name string) string {
	result := ""
	upper := true
	for _, c := range name {
		if c == '_' || c == '-' {
			upper = true
			continue
		}
		if upper {
			result += string(toUpper(c))
			upper = false
		} else {
			result += string(c)
		}
	}
	return result
}

func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32
	}
	return r
}

func schemaToGoType(s *model.Schema) string {
	if s == nil {
		return "any"
	}
	if s.Ref != "" {
		parts := splitRef(s.Ref)
		if len(parts) > 0 {
			return parts[len(parts)-1]
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

func buildTagData(tags []model.Tag) []tagData {
	// First pass: create tag data
	tagMap := make(map[string]*tagData)
	var result []tagData

	for _, t := range tags {
		td := tagData{
			Name:        t.Name,
			Description: t.Description,
			Parent:      t.Parent,
			Kind:        t.Kind,
		}
		tagMap[t.Name] = &td
		result = append(result, td)
	}

	// Second pass: populate children
	for i := range result {
		if result[i].Parent != "" {
			if parent, ok := tagMap[result[i].Parent]; ok {
				parent.Children = append(parent.Children, result[i].Name)
			}
		}
	}

	// Update result with children data
	for i := range result {
		if td, ok := tagMap[result[i].Name]; ok {
			result[i].Children = td.Children
		}
	}

	return result
}

func extractMultipartFields(schema *model.Schema, bodyRequired bool) []multipartFieldData {
	if schema == nil {
		return nil
	}

	requiredSet := make(map[string]bool)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	var fields []multipartFieldData
	for _, prop := range schema.Properties {
		field := multipartFieldData{
			Name:     prop.Name,
			GoName:   toGoParamName(prop.Name),
			Required: requiredSet[prop.Name] && bodyRequired,
		}

		if prop.Schema != nil {
			if prop.Schema.Format == "binary" {
				field.IsFile = true
				field.Type = "*multipart.FileHeader"
			} else if prop.Schema.Type == model.TypeArray {
				field.IsArray = true
				if prop.Schema.Items != nil && prop.Schema.Items.Format == "binary" {
					field.IsFile = true
					field.Type = "[]*multipart.FileHeader"
				} else {
					field.Type = "[]string"
				}
			} else {
				field.Type = "string"
			}
		} else {
			field.Type = "string"
		}

		fields = append(fields, field)
	}

	return fields
}

func extractFormUrlEncodedFields(schema *model.Schema, bodyRequired bool) []multipartFieldData {
	if schema == nil {
		return nil
	}

	requiredSet := make(map[string]bool)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	var fields []multipartFieldData
	for _, prop := range schema.Properties {
		field := multipartFieldData{
			Name:     prop.Name,
			GoName:   toGoParamName(prop.Name),
			Required: requiredSet[prop.Name] && bodyRequired,
		}

		if prop.Schema != nil && prop.Schema.Type == model.TypeArray {
			field.IsArray = true
			field.Type = "[]string"
		} else {
			field.Type = "string"
		}

		fields = append(fields, field)
	}

	return fields
}
