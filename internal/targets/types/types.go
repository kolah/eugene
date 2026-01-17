package types

import (
	"github.com/kolah/eugene/internal/config"
	"github.com/kolah/eugene/internal/golang"
	"github.com/kolah/eugene/internal/model"
	"github.com/kolah/eugene/internal/templates"
)

type Target struct{}

func New() *Target {
	return &Target{}
}

func (t *Target) Name() string {
	return "types"
}

type templateData struct {
	Package          string
	Schemas          []model.Schema
	NestedTypes      []golang.ResolvedType
	NeedsTime        bool
	NeedsJSON        bool
	UUIDImport       string
	EnumStrategy     string
	UseNullable      bool
	EnableYAMLTags   bool
	ExtensionImports []model.GoTypeImport
	MappedImports    []string
}

func (t *Target) Generate(engine templates.Engine, spec *model.Spec, pkg string, cfg *config.TypesConfig, opts *config.OutputOptions, importMapping map[string]string) (string, error) {
	resolver := golang.NewTypeResolverWithImportMapping(cfg, importMapping)

	// Process all schemas to resolve types and collect nested types
	for _, s := range spec.Schemas {
		schema := s
		if len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 || len(schema.AllOf) > 0 {
			resolver.ResolveType(&schema, "", schema.Name)
			continue
		}
		for _, prop := range schema.Properties {
			resolver.ResolveType(prop.Schema, schema.Name, prop.Name)
		}
	}

	needsTime := false
	needsJSON := false

	for _, s := range spec.Schemas {
		if golang.NeedsTimeImport(&s) {
			needsTime = true
			break
		}
	}

	// Check if we have any union types that need json.RawMessage
	for _, nested := range resolver.NestedTypes() {
		if nested.IsUnion {
			needsJSON = true
			break
		}
	}

	enumStrategy := "const"
	if cfg != nil && cfg.EnumStrategy != "" {
		enumStrategy = cfg.EnumStrategy
	}

	// struct enum strategy needs JSON for marshal/unmarshal
	if enumStrategy == "struct" {
		for _, s := range spec.Schemas {
			if len(s.Enum) > 0 {
				needsJSON = true
				break
			}
		}
	}

	useNullable := cfg != nil && cfg.NullableStrategy == "nullable"
	enableYAMLTags := opts != nil && opts.EnableYAMLTags

	// Collect custom imports from x-oink-go-type-import extensions
	extensionImports := golang.CollectExtensionImports(spec.Schemas)

	data := templateData{
		Package:          pkg,
		Schemas:          spec.Schemas,
		NestedTypes:      resolver.NestedTypes(),
		NeedsTime:        needsTime,
		NeedsJSON:        needsJSON,
		UUIDImport:       resolver.UUIDImport(),
		EnumStrategy:     enumStrategy,
		UseNullable:      useNullable,
		EnableYAMLTags:   enableYAMLTags,
		ExtensionImports: extensionImports,
		MappedImports:    resolver.MappedImports(),
	}

	return engine.Execute("go/types.tmpl", data)
}
