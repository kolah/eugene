package golang

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kolah/eugene/internal/config"
	"github.com/kolah/eugene/internal/model"
)

func GoType(s *model.Schema) string {
	if s == nil {
		return "any"
	}

	if s.Ref != "" {
		return refToTypeName(s.Ref)
	}

	if len(s.AllOf) > 0 || len(s.OneOf) > 0 || len(s.AnyOf) > 0 {
		return "any"
	}

	switch s.Type {
	case model.TypeString:
		return goStringType(s.Format)
	case model.TypeInteger:
		return goIntegerType(s.Format)
	case model.TypeNumber:
		return goNumberType(s.Format)
	case model.TypeBoolean:
		return "bool"
	case model.TypeArray:
		itemType := GoType(s.Items)
		return "[]" + itemType
	case model.TypeObject:
		if s.AdditionalProperties != nil {
			valueType := GoType(s.AdditionalProperties)
			return "map[string]" + valueType
		}
		if len(s.Properties) == 0 {
			return "map[string]any"
		}
		return "any"
	default:
		return "any"
	}
}

func goStringType(format string) string {
	switch format {
	case "date-time":
		return "time.Time"
	case "date":
		return "time.Time"
	case "uuid":
		return "string"
	case "uri":
		return "string"
	case "byte":
		return "[]byte"
	case "binary":
		return "[]byte"
	default:
		return "string"
	}
}

func goIntegerType(format string) string {
	switch format {
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	default:
		return "int"
	}
}

func goNumberType(format string) string {
	switch format {
	case "float":
		return "float32"
	case "double":
		return "float64"
	default:
		return "float64"
	}
}

func refToTypeName(ref string) string {
	if len(ref) > 0 && ref[0] == '#' {
		parts := splitRef(ref)
		if len(parts) > 0 {
			return PascalCase(parts[len(parts)-1])
		}
	}
	return "any"
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

func NeedsTimeImport(s *model.Schema) bool {
	if s == nil {
		return false
	}
	if s.Type == model.TypeString && (s.Format == "date-time" || s.Format == "date") {
		return true
	}
	if s.Items != nil && NeedsTimeImport(s.Items) {
		return true
	}
	for _, p := range s.Properties {
		if NeedsTimeImport(p.Schema) {
			return true
		}
	}
	return false
}

func GoZeroValue(s *model.Schema) string {
	if s == nil {
		return "nil"
	}
	switch s.Type {
	case model.TypeString:
		return `""`
	case model.TypeInteger, model.TypeNumber:
		return "0"
	case model.TypeBoolean:
		return "false"
	case model.TypeArray:
		return "nil"
	case model.TypeObject:
		return "nil"
	default:
		return "nil"
	}
}

func JSONTag(name string, required bool) string {
	if required {
		return fmt.Sprintf("`json:\"%s\"`", name)
	}
	return fmt.Sprintf("`json:\"%s,omitempty\"`", name)
}

// TypeResolver resolves OpenAPI schemas to Go types with context awareness.
// It collects nested types that need to be generated separately.
type TypeResolver struct {
	cfg           *config.TypesConfig
	importMapping map[string]string
	nestedTypes   []ResolvedType
	seen          map[string]bool
	enumValues    map[string]string // enum values hash → canonical type name
	mappedImports map[string]bool
	registry      *EnumRegistry                   // shared registry for stable enum naming
	schemaLookup  func(ref string) *model.Schema // lookup schemas by $ref
}

// ResolvedType represents a type that needs to be generated.
type ResolvedType struct {
	Name          string
	Schema        *model.Schema
	IsUnion       bool
	IsAllOf       bool
	IsEnum        bool
	Discriminator *model.Discriminator
	Variants      []UnionVariant
}

// UnionVariant represents a variant in a oneOf/anyOf union.
type UnionVariant struct {
	Name      string
	TypeName  string
	DiscValue string
	Schema    *model.Schema
}

// NewTypeResolver creates a new TypeResolver with the given configuration.
func NewTypeResolver(cfg *config.TypesConfig) *TypeResolver {
	return NewTypeResolverWithImportMapping(cfg, nil)
}

// NewTypeResolverWithImportMapping creates a TypeResolver with import mapping support.
func NewTypeResolverWithImportMapping(cfg *config.TypesConfig, importMapping map[string]string) *TypeResolver {
	return &TypeResolver{
		cfg:           cfg,
		importMapping: importMapping,
		seen:          make(map[string]bool),
		enumValues:    make(map[string]string),
		mappedImports: make(map[string]bool),
	}
}

// NewTypeResolverWithRegistry creates a TypeResolver with a shared EnumRegistry.
// The registry enables stable, context-aware enum naming across targets.
func NewTypeResolverWithRegistry(cfg *config.TypesConfig, importMapping map[string]string, registry *EnumRegistry) *TypeResolver {
	return &TypeResolver{
		cfg:           cfg,
		importMapping: importMapping,
		seen:          make(map[string]bool),
		enumValues:    make(map[string]string),
		mappedImports: make(map[string]bool),
		registry:      registry,
	}
}

// NewTypeResolverWithSchemaLookup creates a TypeResolver with schema lookup capability.
// This enables the flatten strategy for allOf compositions by resolving $ref schemas.
func NewTypeResolverWithSchemaLookup(cfg *config.TypesConfig, importMapping map[string]string, registry *EnumRegistry, schemaLookup func(ref string) *model.Schema) *TypeResolver {
	return &TypeResolver{
		cfg:           cfg,
		importMapping: importMapping,
		seen:          make(map[string]bool),
		enumValues:    make(map[string]string),
		mappedImports: make(map[string]bool),
		registry:      registry,
		schemaLookup:  schemaLookup,
	}
}

// NestedTypes returns all nested types collected during resolution.
func (r *TypeResolver) NestedTypes() []ResolvedType {
	return r.nestedTypes
}

// MappedImports returns all import paths used from import mapping.
func (r *TypeResolver) MappedImports() []string {
	var imports []string
	for pkg := range r.mappedImports {
		imports = append(imports, pkg)
	}
	return imports
}

// packageName extracts the package name from an import path.
func packageName(importPath string) string {
	// Get the last element of the import path
	parts := splitRef("/" + importPath) // Add leading slash for consistent splitting
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return importPath
}

// ResolveType resolves a schema to a Go type name, collecting nested types as needed.
func (r *TypeResolver) ResolveType(s *model.Schema, parentName, fieldName string) string {
	if s == nil {
		return "any"
	}

	if s.Ref != "" {
		// Check import mapping first
		if r.importMapping != nil {
			if pkgPath, ok := r.importMapping[s.Ref]; ok {
				r.mappedImports[pkgPath] = true
				typeName := refToTypeName(s.Ref)
				pkgName := packageName(pkgPath)
				return pkgName + "." + typeName
			}
		}
		return refToTypeName(s.Ref)
	}

	// Handle oneOf/anyOf unions
	if len(s.OneOf) > 0 || len(s.AnyOf) > 0 {
		return r.resolveUnion(s, parentName, fieldName)
	}

	// Handle allOf composition
	if len(s.AllOf) > 0 {
		return r.resolveAllOf(s, parentName, fieldName)
	}

	// Handle inline enums - generate nested enum types
	if len(s.Enum) > 0 && parentName != "" {
		return r.resolveEnum(s, parentName, fieldName)
	}

	switch s.Type {
	case model.TypeString:
		return r.goStringType(s.Format)
	case model.TypeInteger:
		return goIntegerType(s.Format)
	case model.TypeNumber:
		return goNumberType(s.Format)
	case model.TypeBoolean:
		return "bool"
	case model.TypeArray:
		itemType := r.ResolveType(s.Items, parentName, fieldName+"Item")
		return "[]" + itemType
	case model.TypeObject:
		return r.resolveObject(s, parentName, fieldName)
	default:
		return "any"
	}
}

func (r *TypeResolver) goStringType(format string) string {
	switch format {
	case "date-time", "date":
		return "time.Time"
	case "uuid":
		return r.uuidType()
	case "uri":
		return "string"
	case "byte", "binary":
		return "[]byte"
	default:
		return "string"
	}
}

func (r *TypeResolver) uuidType() string {
	if r.cfg == nil {
		return "string"
	}
	switch r.cfg.UUIDPackage {
	case "google":
		return "uuid.UUID"
	case "gofrs":
		return "uuid.UUID"
	default:
		return "string"
	}
}

// UUIDImport returns the import path for UUID if needed.
func (r *TypeResolver) UUIDImport() string {
	if r.cfg == nil {
		return ""
	}
	switch r.cfg.UUIDPackage {
	case "google":
		return "github.com/google/uuid"
	case "gofrs":
		return "github.com/gofrs/uuid"
	default:
		return ""
	}
}

func (r *TypeResolver) resolveObject(s *model.Schema, parentName, fieldName string) string {
	if s.AdditionalProperties != nil {
		valueType := r.ResolveType(s.AdditionalProperties, parentName, fieldName+"Value")
		return "map[string]" + valueType
	}

	if len(s.Properties) == 0 {
		return "map[string]any"
	}

	// Inline object - generate a nested type
	nestedName := parentName + PascalCase(fieldName)
	if r.seen[nestedName] {
		return nestedName
	}
	r.seen[nestedName] = true

	// Recursively resolve property types
	resolvedSchema := *s
	resolvedSchema.Name = nestedName
	for i, prop := range resolvedSchema.Properties {
		r.ResolveType(prop.Schema, nestedName, prop.Name)
		resolvedSchema.Properties[i] = prop
	}

	r.nestedTypes = append(r.nestedTypes, ResolvedType{
		Name:   nestedName,
		Schema: &resolvedSchema,
	})

	return nestedName
}

func (r *TypeResolver) resolveEnum(s *model.Schema, parentName, fieldName string) string {
	// Use registry for stable naming when available
	if r.registry != nil {
		name, ok := r.registry.GetCanonicalName(s.Enum)
		if !ok {
			// Fallback to field-based name if not in registry
			name = PascalCase(fieldName)
		}

		// Skip if already generated by another target
		if r.registry.IsGenerated(name) {
			return name
		}

		// Skip if already seen in this resolver
		if r.seen[name] {
			return name
		}
		r.seen[name] = true

		enumSchema := *s
		enumSchema.Name = name
		rt := ResolvedType{Name: name, Schema: &enumSchema, IsEnum: true}
		r.nestedTypes = append(r.nestedTypes, rt)
		r.registry.MarkGenerated(name, rt)

		return name
	}

	// Fallback: existing behavior without registry
	enumKey := r.enumValuesKey(s.Enum)

	// Check if we've seen this exact enum values before - reuse existing type
	if existingName, ok := r.enumValues[enumKey]; ok {
		return existingName
	}

	// Create new type name
	nestedName := parentName + PascalCase(fieldName)
	if r.seen[nestedName] {
		return nestedName
	}
	r.seen[nestedName] = true

	// Store for future deduplication by values
	r.enumValues[enumKey] = nestedName

	// Create a copy of the schema with the new name
	enumSchema := *s
	enumSchema.Name = nestedName

	r.nestedTypes = append(r.nestedTypes, ResolvedType{
		Name:   nestedName,
		Schema: &enumSchema,
		IsEnum: true,
	})

	return nestedName
}

// enumValuesKey creates a canonical key from enum values for deduplication
func (r *TypeResolver) enumValuesKey(values []any) string {
	strs := make([]string, 0, len(values))
	for _, v := range values {
		if s, ok := v.(string); ok {
			strs = append(strs, s)
		}
	}
	sort.Strings(strs)
	return strings.Join(strs, "|")
}

func (r *TypeResolver) resolveUnion(s *model.Schema, parentName, fieldName string) string {
	schemas := s.OneOf
	if len(schemas) == 0 {
		schemas = s.AnyOf
	}

	nestedName := parentName + PascalCase(fieldName)
	if r.seen[nestedName] {
		return nestedName
	}
	r.seen[nestedName] = true

	var variants []UnionVariant
	for _, variant := range schemas {
		var v UnionVariant
		if variant.Ref != "" {
			v.TypeName = refToTypeName(variant.Ref)
			v.Name = v.TypeName
		} else {
			// Inline schema in union - resolve it
			v.TypeName = r.ResolveType(variant, nestedName, "Variant")
			v.Name = v.TypeName
		}
		v.Schema = variant

		// Check if discriminator mapping provides a value
		if s.Discriminator != nil && s.Discriminator.Mapping != nil {
			for discVal, ref := range s.Discriminator.Mapping {
				if variant.Ref == ref || refToTypeName(ref) == v.TypeName {
					v.DiscValue = discVal
					break
				}
			}
		}

		variants = append(variants, v)
	}

	r.nestedTypes = append(r.nestedTypes, ResolvedType{
		Name:          nestedName,
		Schema:        s,
		IsUnion:       true,
		Discriminator: s.Discriminator,
		Variants:      variants,
	})

	return nestedName
}

func (r *TypeResolver) resolveAllOf(s *model.Schema, parentName, fieldName string) string {
	nestedName := parentName + PascalCase(fieldName)
	if r.seen[nestedName] {
		return nestedName
	}
	r.seen[nestedName] = true

	// Check if flatten strategy is enabled
	shouldFlatten := r.cfg != nil && r.cfg.AllOfStrategy == "flatten" && r.schemaLookup != nil

	if shouldFlatten {
		// Flatten all allOf schemas including $refs
		merged := r.flattenAllOfSchemas(s.AllOf, nestedName)
		merged.Name = nestedName
		r.nestedTypes = append(r.nestedTypes, ResolvedType{
			Name:    nestedName,
			Schema:  merged,
			IsAllOf: false, // No embedding when flattened
		})
		return nestedName
	}

	// Default embed strategy: check if all are refs - can use embedding
	allRefs := true
	for _, sub := range s.AllOf {
		if sub.Ref == "" {
			allRefs = false
			break
		}
	}

	resolvedSchema := *s
	resolvedSchema.Name = nestedName

	if !allRefs {
		// Merge inline properties, keep refs for embedding
		merged := r.mergeAllOfSchemas(s.AllOf, nestedName)
		resolvedSchema = *merged
		resolvedSchema.Name = nestedName
	}

	r.nestedTypes = append(r.nestedTypes, ResolvedType{
		Name:    nestedName,
		Schema:  &resolvedSchema,
		IsAllOf: len(resolvedSchema.AllOf) > 0,
	})

	return nestedName
}

func (r *TypeResolver) mergeAllOfSchemas(schemas []*model.Schema, parentName string) *model.Schema {
	merged := &model.Schema{
		Type: model.TypeObject,
	}

	requiredMap := make(map[string]bool)

	for _, s := range schemas {
		if s.Ref != "" {
			merged.AllOf = append(merged.AllOf, s)
			continue
		}

		for _, prop := range s.Properties {
			r.ResolveType(prop.Schema, parentName, prop.Name)
			merged.Properties = append(merged.Properties, prop)
		}

		for _, req := range s.Required {
			requiredMap[req] = true
		}

		if s.Description != "" && merged.Description == "" {
			merged.Description = s.Description
		}
	}

	for req := range requiredMap {
		merged.Required = append(merged.Required, req)
	}

	return merged
}

// flattenAllOfSchemas merges all allOf schemas (including $refs) into a single flat schema.
// This is used when allof-strategy is set to "flatten".
func (r *TypeResolver) flattenAllOfSchemas(schemas []*model.Schema, parentName string) *model.Schema {
	merged := &model.Schema{
		Type: model.TypeObject,
	}

	requiredMap := make(map[string]bool)
	seenProps := make(map[string]bool)

	var flatten func(schemas []*model.Schema)
	flatten = func(schemas []*model.Schema) {
		for _, s := range schemas {
			if s.Ref != "" {
				// Resolve the referenced schema
				refSchema := r.schemaLookup(s.Ref)
				if refSchema == nil {
					continue
				}
				// Recursively flatten if the referenced schema has allOf
				if len(refSchema.AllOf) > 0 {
					flatten(refSchema.AllOf)
				}
				// Add properties from the referenced schema
				for _, prop := range refSchema.Properties {
					if seenProps[prop.Name] {
						continue // First occurrence wins
					}
					seenProps[prop.Name] = true
					r.ResolveType(prop.Schema, parentName, prop.Name)
					merged.Properties = append(merged.Properties, prop)
				}
				// Add required fields
				for _, req := range refSchema.Required {
					requiredMap[req] = true
				}
				if refSchema.Description != "" && merged.Description == "" {
					merged.Description = refSchema.Description
				}
				continue
			}

			// Inline schema - add properties directly
			for _, prop := range s.Properties {
				if seenProps[prop.Name] {
					continue // First occurrence wins
				}
				seenProps[prop.Name] = true
				r.ResolveType(prop.Schema, parentName, prop.Name)
				merged.Properties = append(merged.Properties, prop)
			}

			for _, req := range s.Required {
				requiredMap[req] = true
			}

			if s.Description != "" && merged.Description == "" {
				merged.Description = s.Description
			}
		}
	}

	flatten(schemas)

	for req := range requiredMap {
		merged.Required = append(merged.Required, req)
	}

	return merged
}
