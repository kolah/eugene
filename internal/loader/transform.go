package loader

import (
	"strings"

	"github.com/kolah/eugene/internal/model"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

type transformer struct {
	componentSchemas map[*base.Schema]string
}

func Transform(result *Result) (*model.Spec, error) {
	doc := result.Document.Model

	t := &transformer{
		componentSchemas: make(map[*base.Schema]string),
	}

	if doc.Components != nil && doc.Components.Schemas != nil {
		for name, schemaProxy := range doc.Components.Schemas.FromOldest() {
			t.componentSchemas[schemaProxy.Schema()] = "#/components/schemas/" + name
		}
	}

	spec := &model.Spec{
		Info:    transformInfo(doc.Info),
		Servers: transformServers(doc.Servers),
		Tags:    transformTags(doc.Tags),
	}

	if doc.Components != nil && doc.Components.Schemas != nil {
		for name, schemaProxy := range doc.Components.Schemas.FromOldest() {
			schema := t.transformSchema(name, schemaProxy.Schema())
			spec.Schemas = append(spec.Schemas, *schema)
		}
	}

	if doc.Components != nil && doc.Components.Responses != nil {
		for name, resp := range doc.Components.Responses.FromOldest() {
			schema := t.extractResponseSchema(name, resp)
			if schema != nil && !schemaExists(spec.Schemas, schema.Name) {
				spec.Schemas = append(spec.Schemas, *schema)
			}
		}
	}

	if doc.Paths != nil {
		for pathStr, pathItem := range doc.Paths.PathItems.FromOldest() {
			path, ops := t.transformPath(pathStr, pathItem)
			spec.Paths = append(spec.Paths, path)
			spec.Operations = append(spec.Operations, ops...)
		}
	}

	if doc.Components != nil && doc.Components.SecuritySchemes != nil {
		for name, scheme := range doc.Components.SecuritySchemes.FromOldest() {
			spec.Security = append(spec.Security, transformSecurityScheme(name, scheme))
		}
	}

	return spec, nil
}

func transformInfo(info *base.Info) model.Info {
	if info == nil {
		return model.Info{}
	}
	return model.Info{
		Title:       info.Title,
		Description: info.Description,
		Version:     info.Version,
	}
}

func transformServers(servers []*v3.Server) []model.Server {
	var result []model.Server
	for _, s := range servers {
		result = append(result, model.Server{
			URL:         s.URL,
			Description: s.Description,
		})
	}
	return result
}

func transformTags(tags []*base.Tag) []model.Tag {
	var result []model.Tag
	for _, t := range tags {
		result = append(result, model.Tag{
			Name:        t.Name,
			Summary:     t.Summary,
			Description: t.Description,
			Parent:      t.Parent,
			Kind:        t.Kind,
		})
	}
	return result
}

func (t *transformer) transformPath(pathStr string, pathItem *v3.PathItem) (model.Path, []model.Operation) {
	path := model.Path{Path: pathStr}
	var ops []model.Operation

	// Use a slice for deterministic ordering
	methods := []struct {
		method model.Method
		op     *v3.Operation
	}{
		{model.MethodGet, pathItem.Get},
		{model.MethodPost, pathItem.Post},
		{model.MethodPut, pathItem.Put},
		{model.MethodDelete, pathItem.Delete},
		{model.MethodPatch, pathItem.Patch},
		{model.MethodHead, pathItem.Head},
		{model.MethodOptions, pathItem.Options},
		{model.MethodTrace, pathItem.Trace},
		{model.MethodQuery, pathItem.Query}, // OpenAPI 3.2
	}

	for _, m := range methods {
		if m.op == nil {
			continue
		}
		operation := t.transformOperation(m.method, pathStr, m.op)
		ops = append(ops, operation)
		path.Operations = append(path.Operations, operation)
	}

	return path, ops
}

func (t *transformer) transformOperation(method model.Method, path string, op *v3.Operation) model.Operation {
	operation := model.Operation{
		ID:          op.OperationId,
		Method:      method,
		Path:        path,
		Summary:     op.Summary,
		Description: op.Description,
		Tags:        op.Tags,
		Deprecated:  boolPtr(op.Deprecated),
	}

	for _, p := range op.Parameters {
		operation.Parameters = append(operation.Parameters, t.transformParameter(p))
	}

	if op.RequestBody != nil {
		operation.RequestBody = t.transformRequestBody(op.RequestBody)
	}

	if op.Responses != nil && op.Responses.Codes != nil {
		for code, resp := range op.Responses.Codes.FromOldest() {
			response := t.transformResponse(code, resp)
			operation.Responses = append(operation.Responses, response)

			// Detect SSE/streaming responses
			if strings.HasPrefix(code, "2") && operation.Streaming == nil {
				for _, content := range response.Content {
					if content.MediaType == "text/event-stream" {
						operation.Streaming = &model.StreamingConfig{
							MediaType:   content.MediaType,
							EventSchema: content.Schema,
						}
						if content.Schema != nil && content.Schema.Ref != "" {
							parts := strings.Split(content.Schema.Ref, "/")
							if len(parts) > 0 {
								operation.Streaming.EventType = parts[len(parts)-1]
							}
						}
						break
					}
				}
			}
		}
	}

	for _, secReq := range op.Security {
		for name, scopes := range secReq.Requirements.FromOldest() {
			operation.Security = append(operation.Security, model.SecurityRequirement{
				Name:   name,
				Scopes: scopes,
			})
		}
	}

	operation.Callbacks = t.transformCallbacks(op.Callbacks)

	return operation
}

func (t *transformer) transformCallbacks(callbacks *orderedmap.Map[string, *v3.Callback]) []model.Callback {
	if callbacks == nil {
		return nil
	}
	var result []model.Callback
	for name, cb := range callbacks.FromOldest() {
		callback := model.Callback{Name: name}
		for expr, pathItem := range cb.Expression.FromOldest() {
			callback.Expression = expr
			callback.Operations = append(callback.Operations, t.transformCallbackOperations(pathItem)...)
		}
		result = append(result, callback)
	}
	return result
}

func (t *transformer) transformCallbackOperations(pathItem *v3.PathItem) []model.CallbackOperation {
	var ops []model.CallbackOperation
	methods := []struct {
		method model.Method
		op     *v3.Operation
	}{
		{model.MethodGet, pathItem.Get},
		{model.MethodPost, pathItem.Post},
		{model.MethodPut, pathItem.Put},
		{model.MethodDelete, pathItem.Delete},
		{model.MethodPatch, pathItem.Patch},
		{model.MethodHead, pathItem.Head},
		{model.MethodOptions, pathItem.Options},
		{model.MethodTrace, pathItem.Trace},
	}
	for _, m := range methods {
		if m.op == nil {
			continue
		}
		cbOp := model.CallbackOperation{Method: m.method}
		if m.op.RequestBody != nil {
			cbOp.RequestBody = t.transformRequestBody(m.op.RequestBody)
		}
		if m.op.Responses != nil && m.op.Responses.Codes != nil {
			for code, resp := range m.op.Responses.Codes.FromOldest() {
				cbOp.Responses = append(cbOp.Responses, t.transformResponse(code, resp))
			}
		}
		ops = append(ops, cbOp)
	}
	return ops
}

func (t *transformer) transformParameter(p *v3.Parameter) model.Parameter {
	param := model.Parameter{
		Name:        p.Name,
		In:          model.ParameterLocation(strings.ToLower(p.In)),
		Description: p.Description,
		Required:    boolPtr(p.Required),
		Deprecated:  p.Deprecated,
	}

	if p.Schema != nil {
		param.Schema = t.transformSchemaProxy(p.Schema)
	} else if p.Content != nil {
		// OpenAPI 3.2: querystring parameters use content instead of schema
		for _, content := range p.Content.FromOldest() {
			if content.Schema != nil {
				param.Schema = t.transformSchemaProxy(content.Schema)
				break
			}
		}
	}

	return param
}

func (t *transformer) transformRequestBody(rb *v3.RequestBody) *model.RequestBody {
	body := &model.RequestBody{
		Description: rb.Description,
		Required:    boolPtr(rb.Required),
	}

	if rb.Content != nil {
		for mediaType, content := range rb.Content.FromOldest() {
			mtc := model.MediaTypeContent{MediaType: mediaType}
			if content.Schema != nil {
				mtc.Schema = t.transformSchemaProxy(content.Schema)
			}
			body.Content = append(body.Content, mtc)
		}
	}

	return body
}

func (t *transformer) transformResponse(code string, resp *v3.Response) model.Response {
	response := model.Response{
		StatusCode:  code,
		Description: resp.Description,
	}

	if resp.Content != nil {
		for mediaType, content := range resp.Content.FromOldest() {
			mtc := model.MediaTypeContent{MediaType: mediaType}
			if content.Schema != nil {
				mtc.Schema = t.transformSchemaProxy(content.Schema)
			}
			response.Content = append(response.Content, mtc)
		}
	}

	if resp.Headers != nil {
		for name, header := range resp.Headers.FromOldest() {
			h := model.Header{
				Name:        name,
				Description: header.Description,
				Required:    header.Required,
			}
			if header.Schema != nil {
				h.Schema = t.transformSchemaProxy(header.Schema)
			}
			response.Headers = append(response.Headers, h)
		}
	}

	return response
}

func (t *transformer) transformSchemaProxy(proxy *base.SchemaProxy) *model.Schema {
	if proxy == nil {
		return nil
	}

	ref := proxy.GetReference()
	if ref == "" {
		if resolved, ok := t.componentSchemas[proxy.Schema()]; ok {
			return &model.Schema{Ref: resolved}
		}
	}

	schema := t.transformSchema("", proxy.Schema())
	if schema != nil && ref != "" {
		schema.Ref = ref
	}
	return schema
}

func (t *transformer) transformSchema(name string, s *base.Schema) *model.Schema {
	if s == nil {
		return nil
	}

	schema := &model.Schema{
		Name:        name,
		Description: s.Description,
		Format:      s.Format,
		Nullable:    boolPtr(s.Nullable),
		Deprecated:  boolPtr(s.Deprecated),
		Default:     s.Default,
		Example:     s.Example,
		Pattern:     s.Pattern,
		UniqueItems: boolPtr(s.UniqueItems),
	}

	if len(s.Type) > 0 {
		schema.Type = model.SchemaType(s.Type[0])
	}

	if s.Enum != nil {
		for _, e := range s.Enum {
			schema.Enum = append(schema.Enum, e.Value)
		}
	}

	if s.Properties != nil {
		for propName, propProxy := range s.Properties.FromOldest() {
			propSchema := t.transformSchemaProxy(propProxy)
			if propSchema != nil && propSchema.Name == "" {
				propSchema.Name = propName
			}
			prop := model.Property{
				Name:   propName,
				Schema: propSchema,
			}
			schema.Properties = append(schema.Properties, prop)
		}
	}

	schema.Required = s.Required

	if s.Items != nil && s.Items.A != nil {
		schema.Items = t.transformSchemaProxy(s.Items.A)
	}

	if s.AdditionalProperties != nil && s.AdditionalProperties.A != nil {
		schema.AdditionalProperties = t.transformSchemaProxy(s.AdditionalProperties.A)
	}

	for _, proxy := range s.AllOf {
		schema.AllOf = append(schema.AllOf, t.transformSchemaProxy(proxy))
	}
	for _, proxy := range s.OneOf {
		schema.OneOf = append(schema.OneOf, t.transformSchemaProxy(proxy))
	}
	for _, proxy := range s.AnyOf {
		schema.AnyOf = append(schema.AnyOf, t.transformSchemaProxy(proxy))
	}

	if s.Discriminator != nil {
		schema.Discriminator = &model.Discriminator{
			PropertyName: s.Discriminator.PropertyName,
			Mapping:      make(map[string]string),
		}
		if s.Discriminator.Mapping != nil {
			for k, v := range s.Discriminator.Mapping.FromOldest() {
				schema.Discriminator.Mapping[k] = v
			}
		}
	}

	if s.Minimum != nil {
		v := float64(*s.Minimum)
		schema.Minimum = &v
	}
	if s.Maximum != nil {
		v := float64(*s.Maximum)
		schema.Maximum = &v
	}
	if s.MinLength != nil {
		v := int64(*s.MinLength)
		schema.MinLength = &v
	}
	if s.MaxLength != nil {
		v := int64(*s.MaxLength)
		schema.MaxLength = &v
	}
	if s.MinItems != nil {
		v := int64(*s.MinItems)
		schema.MinItems = &v
	}
	if s.MaxItems != nil {
		v := int64(*s.MaxItems)
		schema.MaxItems = &v
	}
	if s.MinProperties != nil {
		v := int64(*s.MinProperties)
		schema.MinProperties = &v
	}
	if s.MaxProperties != nil {
		v := int64(*s.MaxProperties)
		schema.MaxProperties = &v
	}

	if s.ExclusiveMinimum != nil && s.ExclusiveMinimum.IsA() {
		schema.ExclusiveMinimum = s.ExclusiveMinimum.A
	}
	if s.ExclusiveMaximum != nil && s.ExclusiveMaximum.IsA() {
		schema.ExclusiveMaximum = s.ExclusiveMaximum.A
	}

	// Parse x-oink-* extensions
	schema.Extensions = parseExtensions(s.Extensions)

	return schema
}

func parseExtensions(extensions *orderedmap.Map[string, *yaml.Node]) *model.SchemaExtensions {
	if extensions == nil {
		return nil
	}

	var ext *model.SchemaExtensions

	for pair := extensions.First(); pair != nil; pair = pair.Next() {
		key := pair.Key()
		node := pair.Value()

		if !strings.HasPrefix(key, "x-oink-") {
			continue
		}

		if ext == nil {
			ext = &model.SchemaExtensions{}
		}

		switch key {
		case "x-oink-go-type":
			if node.Kind == yaml.ScalarNode {
				ext.GoType = node.Value
			}
		case "x-oink-go-type-import":
			ext.GoTypeImport = parseGoTypeImport(node)
		case "x-oink-go-name":
			if node.Kind == yaml.ScalarNode {
				ext.GoName = node.Value
			}
		case "x-oink-extra-tags":
			ext.ExtraTags = parseExtraTags(node)
		case "x-oink-omitempty":
			if node.Kind == yaml.ScalarNode {
				v := node.Value == "true"
				ext.OmitEmpty = &v
			}
		case "x-oink-omitzero":
			if node.Kind == yaml.ScalarNode {
				v := node.Value == "true"
				ext.OmitZero = &v
			}
		case "x-oink-json-ignore":
			if node.Kind == yaml.ScalarNode {
				ext.JSONIgnore = node.Value == "true"
			}
		}
	}

	return ext
}

func parseGoTypeImport(node *yaml.Node) *model.GoTypeImport {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}

	imp := &model.GoTypeImport{}
	for i := 0; i < len(node.Content)-1; i += 2 {
		key := node.Content[i].Value
		value := node.Content[i+1].Value
		switch key {
		case "path":
			imp.Path = value
		case "alias":
			imp.Alias = value
		}
	}
	if imp.Path == "" {
		return nil
	}
	return imp
}

func parseExtraTags(node *yaml.Node) map[string]string {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}

	tags := make(map[string]string)
	for i := 0; i < len(node.Content)-1; i += 2 {
		key := node.Content[i].Value
		value := node.Content[i+1].Value
		tags[key] = value
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}

func transformSecurityScheme(name string, scheme *v3.SecurityScheme) model.SecurityScheme {
	ss := model.SecurityScheme{
		Name:         name,
		Type:         model.SecuritySchemeType(scheme.Type),
		Description:  scheme.Description,
		In:           scheme.In,
		Scheme:       scheme.Scheme,
		BearerFormat: scheme.BearerFormat,
	}

	if scheme.Flows != nil {
		ss.Flows = &model.OAuthFlows{}
		if scheme.Flows.Implicit != nil {
			ss.Flows.Implicit = transformOAuthFlow(scheme.Flows.Implicit)
		}
		if scheme.Flows.Password != nil {
			ss.Flows.Password = transformOAuthFlow(scheme.Flows.Password)
		}
		if scheme.Flows.ClientCredentials != nil {
			ss.Flows.ClientCredentials = transformOAuthFlow(scheme.Flows.ClientCredentials)
		}
		if scheme.Flows.AuthorizationCode != nil {
			ss.Flows.AuthorizationCode = transformOAuthFlow(scheme.Flows.AuthorizationCode)
		}
	}

	return ss
}

func transformOAuthFlow(flow *v3.OAuthFlow) *model.OAuthFlow {
	f := &model.OAuthFlow{
		AuthorizationURL: flow.AuthorizationUrl,
		TokenURL:         flow.TokenUrl,
		RefreshURL:       flow.RefreshUrl,
		Scopes:           make(map[string]string),
	}

	if flow.Scopes != nil {
		for scope, desc := range flow.Scopes.FromOldest() {
			f.Scopes[scope] = desc
		}
	}

	return f
}

func (t *transformer) extractResponseSchema(name string, resp *v3.Response) *model.Schema {
	if resp == nil || resp.Content == nil {
		return nil
	}

	for _, content := range resp.Content.FromOldest() {
		if content.Schema == nil {
			continue
		}

		ref := content.Schema.GetReference()
		if ref != "" {
			return &model.Schema{
				Name: name,
				Ref:  ref,
			}
		}

		return t.transformSchema(name, content.Schema.Schema())
	}
	return nil
}

func schemaExists(schemas []model.Schema, name string) bool {
	for _, s := range schemas {
		if s.Name == name {
			return true
		}
	}
	return false
}

func boolPtr(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
