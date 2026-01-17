package client

import (
	"strings"

	"github.com/kolah/eugene/internal/golang"
	"github.com/kolah/eugene/internal/model"
	"github.com/kolah/eugene/internal/templates"
)

type Target struct{}

func New() *Target {
	return &Target{}
}

func (t *Target) Name() string {
	return "client"
}

type clientFeatures struct {
	HasStreaming      bool // any operation uses SSE
	HasQueryParams    bool // any operation uses standard query params
	HasQueryString    bool // any operation uses querystring param (OpenAPI 3.2)
	HasMultipart      bool // any operation uses multipart/form-data
	HasFormUrlEncoded bool // any operation uses application/x-www-form-urlencoded
}

type templateData struct {
	Package    string
	Operations []operationData
	Tags       []tagData // OpenAPI 3.2: hierarchical tags
	Features   clientFeatures
}

type tagData struct {
	Name        string
	Description string
	Parent      string
	Kind        string
	Children    []string
}

type operationData struct {
	ID               string
	Method           string
	Path             string
	Summary          string
	Description      string
	PathParams       []parameterData
	QueryParams      []parameterData
	HeaderParams     []parameterData
	QueryStringParam *parameterData // OpenAPI 3.2: in: querystring
	RequestBody      *requestBodyData
	SuccessResponse  *responseData   // kept for backward compat
	Responses        []responseData  // all responses with status codes
	Streaming        *streamingData  // SSE streaming
	HasPathParams    bool
	HasQueryParams   bool
	HasHeaderParams  bool
	HasQueryString   bool // OpenAPI 3.2: in: querystring
	HasBody          bool
	IsStreaming      bool
	IsMultipart      bool
	IsFormUrlEncoded bool
}

type streamingData struct {
	EventType string
}

type parameterData struct {
	Name     string
	GoName   string
	Type     string
	Required bool
}

type requestBodyData struct {
	Required         bool
	MediaType        string
	Type             string
	IsMultipart      bool
	IsFormUrlEncoded bool
	MultipartFields  []multipartFieldData
}

type multipartFieldData struct {
	Name     string
	GoName   string
	Type     string // "io.Reader", "string", "[]string"
	IsFile   bool
	IsArray  bool
	Required bool
}

type responseData struct {
	StatusCode string
	MediaType  string
	Type       string
}

func (t *Target) Generate(engine templates.Engine, spec *model.Spec, pkg string) (string, error) {
	data := templateData{Package: pkg}

	for _, op := range spec.Operations {
		opData := operationData{
			ID:          op.ID,
			Method:      string(op.Method),
			Path:        op.Path,
			Summary:     op.Summary,
			Description: op.Description,
			IsStreaming: op.Streaming != nil,
		}

		if op.Streaming != nil {
			opData.Streaming = &streamingData{
				EventType: op.Streaming.EventType,
			}
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
				opData.HasPathParams = true
			case model.LocationQuery:
				opData.QueryParams = append(opData.QueryParams, pd)
				opData.HasQueryParams = true
			case model.LocationHeader:
				opData.HeaderParams = append(opData.HeaderParams, pd)
				opData.HasHeaderParams = true
			case model.LocationQueryString:
				// OpenAPI 3.2: querystring parameter - entire query as single object
				opData.QueryStringParam = &pd
				opData.HasQueryString = true
			}
		}

		if op.RequestBody != nil {
			opData.HasBody = true
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
			rd := responseData{StatusCode: r.StatusCode}
			if len(r.Content) > 0 {
				rd.MediaType = r.Content[0].MediaType
				rd.Type = schemaToGoType(r.Content[0].Schema)
			}
			opData.Responses = append(opData.Responses, rd)

			// Keep SuccessResponse for backward compat
			if strings.HasPrefix(r.StatusCode, "2") && opData.SuccessResponse == nil {
				opData.SuccessResponse = &rd
			}
		}

		data.Operations = append(data.Operations, opData)

		// Compute features from operation flags
		if opData.IsStreaming {
			data.Features.HasStreaming = true
		}
		if opData.HasQueryParams {
			data.Features.HasQueryParams = true
		}
		if opData.HasQueryString {
			data.Features.HasQueryString = true
		}
	}

	// Build hierarchical tag data
	data.Tags = buildTagData(spec.Tags)

	return engine.Execute("go/client.tmpl", data)
}

func buildTagData(tags []model.Tag) []tagData {
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

	for i := range result {
		if result[i].Parent != "" {
			if parent, ok := tagMap[result[i].Parent]; ok {
				parent.Children = append(parent.Children, result[i].Name)
			}
		}
	}

	for i := range result {
		if td, ok := tagMap[result[i].Name]; ok {
			result[i].Children = td.Children
		}
	}

	return result
}

func schemaToGoType(s *model.Schema) string {
	if s == nil {
		return "any"
	}
	if s.Ref != "" {
		parts := strings.Split(s.Ref, "/")
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
		if s.Items != nil && s.Items.Ref != "" {
			parts := strings.Split(s.Items.Ref, "/")
			if len(parts) > 0 {
				return "[]" + golang.PascalCase(parts[len(parts)-1])
			}
		}
		return "[]" + schemaToGoType(s.Items)
	default:
		return "any"
	}
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
			GoName:   golang.PascalCase(prop.Name),
			Required: requiredSet[prop.Name] && bodyRequired,
		}

		if prop.Schema != nil {
			if prop.Schema.Format == "binary" {
				field.IsFile = true
				field.Type = "*FileUpload"
			} else if prop.Schema.Type == model.TypeArray {
				field.IsArray = true
				if prop.Schema.Items != nil && prop.Schema.Items.Format == "binary" {
					field.IsFile = true
					field.Type = "[]*FileUpload"
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
			GoName:   golang.PascalCase(prop.Name),
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
