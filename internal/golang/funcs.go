package golang

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/kolah/eugene/internal/config"
	"github.com/kolah/eugene/internal/model"
)

// toSchemaPtr converts any schema value to a pointer.
// Handles both *model.Schema and model.Schema from templates.
func toSchemaPtr(s any) *model.Schema {
	if s == nil {
		return nil
	}
	switch v := s.(type) {
	case *model.Schema:
		return v
	case model.Schema:
		return &v
	default:
		return nil
	}
}

// TemplateFuncsWithResolver returns template functions with a resolver for context-aware type resolution.
func TemplateFuncsWithResolver(cfg *config.TypesConfig) template.FuncMap {
	resolver := NewTypeResolver(cfg)

	funcs := TemplateFuncs()
	funcs["resolveType"] = func(s any, parentName, fieldName string) string {
		return resolver.ResolveType(toSchemaPtr(s), parentName, fieldName)
	}
	funcs["nullableType"] = func(baseType string) string {
		return NullableType(cfg, baseType)
	}
	funcs["useNullable"] = func() bool {
		return cfg != nil && cfg.NullableStrategy == "nullable"
	}
	return funcs
}

func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"pascalCase":     PascalCase,
		"camelCase":      CamelCase,
		"snakeCase":      SnakeCase,
		"goType":         goTypeAny,
		"goName":         ToGoIdentifier,
		"goZeroValue":    GoZeroValue,
		"jsonTag":        JSONTag,
		"yamlTag":        YAMLTag,
		"structTag":      structTagAny,
		"structTagYAML":  structTagWithYAMLAny,
		"escapeKeyword":  EscapeKeyword,
		"goComment":      GoComment,
		"isRequired":     IsRequired,
		"needsPointer":   needsPointerAny,
		"isJSONIgnored":  isJSONIgnoredAny,
		"goNameExt":      goNameExtAny,
		"goTypeExt":      goTypeExtAny,
		"lower":          strings.ToLower,
		"upper":          strings.ToUpper,
		"join":           strings.Join,
		"hasPrefix":      strings.HasPrefix,
		"hasSuffix":      strings.HasSuffix,
		"trimPrefix":     strings.TrimPrefix,
		"trimSuffix":     strings.TrimSuffix,
		"refToTypeName":  RefToTypeName,
		"goBaseType":     goBaseTypeAny,
		"enumLiteral":    enumLiteralAny,
		"dict":           Dict,
		"statusCodeInt":  StatusCodeInt,
		"title":          Title,
		"isComposition":  isCompositionAny,
	}
}

// Template wrapper functions that handle both pointer and value schema types
func goTypeAny(s any) string                        { return GoType(toSchemaPtr(s)) }
func goBaseTypeAny(s any) string                    { return GoBaseType(toSchemaPtr(s)) }
func needsPointerAny(s any, required []string) bool { return NeedsPointer(toSchemaPtr(s), required) }
func structTagAny(s any, name string, required bool) string {
	return StructTag(toSchemaPtr(s), name, required)
}
func structTagWithYAMLAny(s any, name string, required bool, enableYAML bool) string {
	return StructTagWithOptions(toSchemaPtr(s), name, required, enableYAML)
}
func isJSONIgnoredAny(s any) bool            { return IsJSONIgnored(toSchemaPtr(s)) }
func goNameExtAny(s any, name string) string { return GoNameWithExtension(toSchemaPtr(s), name) }
func goTypeExtAny(s any) string              { return GoTypeWithExtension(toSchemaPtr(s)) }
func enumLiteralAny(s any, v any) string     { return EnumLiteral(toSchemaPtr(s), v) }

// RefToTypeName extracts the type name from a $ref string.
func RefToTypeName(ref string) string {
	return refToTypeName(ref)
}

// GoBaseType returns the base Go type for a schema (without considering enums).
func GoBaseType(s *model.Schema) string {
	if s == nil {
		return "any"
	}
	switch s.Type {
	case model.TypeString:
		return "string"
	case model.TypeInteger:
		return goIntegerType(s.Format)
	case model.TypeNumber:
		return goNumberType(s.Format)
	case model.TypeBoolean:
		return "bool"
	default:
		return "string"
	}
}

// EnumLiteral formats an enum value as a Go literal.
func EnumLiteral(s *model.Schema, v any) string {
	switch s.Type {
	case model.TypeString:
		return fmt.Sprintf("%q", v)
	case model.TypeInteger, model.TypeNumber:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%q", v)
	}
}

// Dict creates a map from key-value pairs for use in templates.
func Dict(values ...any) map[string]any {
	if len(values)%2 != 0 {
		return nil
	}
	dict := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			continue
		}
		dict[key] = values[i+1]
	}
	return dict
}

func GoComment(s string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	var result strings.Builder
	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString("// ")
		result.WriteString(strings.TrimSpace(line))
	}
	return result.String()
}

func IsRequired(name string, required []string) bool {
	return slices.Contains(required, name)
}

func NeedsPointer(s *model.Schema, required []string) bool {
	if s == nil {
		return false
	}
	if IsRequired(s.Name, required) {
		return false
	}
	if s.Nullable {
		return true
	}
	switch s.Type {
	case model.TypeString, model.TypeInteger, model.TypeNumber, model.TypeBoolean:
		return true
	default:
		return false
	}
}

// NullableType returns the appropriate type wrapper based on the nullable strategy.
// With "nullable" strategy: returns "nullable.Nullable[baseType]"
// With "pointer" strategy (default): returns "*baseType"
func NullableType(cfg *config.TypesConfig, baseType string) string {
	if cfg != nil && cfg.NullableStrategy == "nullable" {
		return fmt.Sprintf("nullable.Nullable[%s]", baseType)
	}
	return "*" + baseType
}

// StatusCodeInt converts an HTTP status code string to int.
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

// Title returns the string in title case (first letter uppercase, rest lowercase).
func Title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// StructTag generates the full struct tag string with extensions support.
// It handles json tag, extra tags from x-oink-extra-tags, and omitempty/omitzero/json-ignore.
func StructTag(s *model.Schema, name string, required bool) string {
	return StructTagWithOptions(s, name, required, false)
}

// StructTagWithOptions generates struct tags with optional YAML tag support.
func StructTagWithOptions(s *model.Schema, name string, required bool, enableYAML bool) string {
	if s == nil {
		tag := JSONTag(name, required)
		if enableYAML {
			tag = tag[:len(tag)-1] + " " + YAMLTag(name, required) + "`"
		}
		return tag
	}

	ext := s.Extensions
	if ext != nil && ext.JSONIgnore {
		if enableYAML {
			return "`json:\"-\" yaml:\"-\"`"
		}
		return "`json:\"-\"`"
	}

	// Build JSON tag parts
	var jsonParts []string
	jsonParts = append(jsonParts, name)

	// Determine omitempty
	omitEmpty := false
	if ext != nil && ext.OmitEmpty != nil {
		omitEmpty = *ext.OmitEmpty
	} else {
		omitEmpty = !required
	}
	if omitEmpty {
		jsonParts = append(jsonParts, "omitempty")
	}

	// Determine omitzero
	if ext != nil && ext.OmitZero != nil && *ext.OmitZero {
		jsonParts = append(jsonParts, "omitzero")
	}

	jsonTag := fmt.Sprintf("json:\"%s\"", strings.Join(jsonParts, ","))

	// Collect all tags
	var tags []string
	tags = append(tags, jsonTag)

	// Add YAML tag if enabled
	if enableYAML {
		var yamlParts []string
		yamlParts = append(yamlParts, name)
		if omitEmpty {
			yamlParts = append(yamlParts, "omitempty")
		}
		tags = append(tags, fmt.Sprintf("yaml:\"%s\"", strings.Join(yamlParts, ",")))
	}

	// Add extra tags from extensions
	if ext != nil && ext.ExtraTags != nil {
		for tagName, tagValue := range ext.ExtraTags {
			tags = append(tags, fmt.Sprintf("%s:\"%s\"", tagName, tagValue))
		}
	}

	return "`" + strings.Join(tags, " ") + "`"
}

// YAMLTag generates a yaml struct tag.
func YAMLTag(name string, required bool) string {
	if required {
		return fmt.Sprintf("yaml:\"%s\"", name)
	}
	return fmt.Sprintf("yaml:\"%s,omitempty\"", name)
}

// IsJSONIgnored returns true if the schema has x-oink-json-ignore: true.
func IsJSONIgnored(s *model.Schema) bool {
	if s == nil || s.Extensions == nil {
		return false
	}
	return s.Extensions.JSONIgnore
}

// GoNameWithExtension returns the field name, using x-oink-go-name if specified.
func GoNameWithExtension(s *model.Schema, name string) string {
	if s != nil && s.Extensions != nil && s.Extensions.GoName != "" {
		return s.Extensions.GoName
	}
	return PascalCase(name)
}

// GoTypeWithExtension returns the custom Go type from x-oink-go-type extension.
// Returns empty string if no extension is specified (caller should fall back to default type).
func GoTypeWithExtension(s *model.Schema) string {
	if s != nil && s.Extensions != nil && s.Extensions.GoType != "" {
		return s.Extensions.GoType
	}
	return ""
}

func isCompositionAny(s any) bool {
	schema := toSchemaPtr(s)
	if schema == nil {
		return false
	}
	return len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 || len(schema.AllOf) > 0
}

// CollectExtensionImports collects custom imports from x-oink-go-type-import extensions.
func CollectExtensionImports(schemas []model.Schema) []model.GoTypeImport {
	var imports []model.GoTypeImport
	seen := make(map[string]bool)

	var collectFromSchema func(s *model.Schema)
	collectFromSchema = func(s *model.Schema) {
		if s == nil {
			return
		}
		if s.Extensions != nil && s.Extensions.GoTypeImport != nil {
			imp := s.Extensions.GoTypeImport
			if !seen[imp.Path] {
				seen[imp.Path] = true
				imports = append(imports, *imp)
			}
		}
		for _, prop := range s.Properties {
			collectFromSchema(prop.Schema)
		}
		collectFromSchema(s.Items)
		collectFromSchema(s.AdditionalProperties)
	}

	for i := range schemas {
		collectFromSchema(&schemas[i])
	}
	return imports
}
