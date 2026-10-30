package golang

import (
	"testing"

	"github.com/kolah/eugene/v2/internal/config"
	"github.com/kolah/eugene/v2/internal/model"
	"github.com/stretchr/testify/require"
)

func TestGoType(t *testing.T) {
	tests := []struct {
		name     string
		schema   *model.Schema
		expected string
	}{
		{"nil schema", nil, "any"},
		{"string", &model.Schema{Type: model.TypeString}, "string"},
		{"string uuid", &model.Schema{Type: model.TypeString, Format: "uuid"}, "string"},
		{"string date-time", &model.Schema{Type: model.TypeString, Format: "date-time"}, "time.Time"},
		{"string date", &model.Schema{Type: model.TypeString, Format: "date"}, "string"},
		{"string byte", &model.Schema{Type: model.TypeString, Format: "byte"}, "[]byte"},
		{"string binary", &model.Schema{Type: model.TypeString, Format: "binary"}, "[]byte"},
		{"integer", &model.Schema{Type: model.TypeInteger}, "int"},
		{"integer int32", &model.Schema{Type: model.TypeInteger, Format: "int32"}, "int32"},
		{"integer int64", &model.Schema{Type: model.TypeInteger, Format: "int64"}, "int64"},
		{"number", &model.Schema{Type: model.TypeNumber}, "float64"},
		{"number float", &model.Schema{Type: model.TypeNumber, Format: "float"}, "float32"},
		{"number double", &model.Schema{Type: model.TypeNumber, Format: "double"}, "float64"},
		{"boolean", &model.Schema{Type: model.TypeBoolean}, "bool"},
		{"array of strings", &model.Schema{Type: model.TypeArray, Items: &model.Schema{Type: model.TypeString}}, "[]string"},
		{"array of integers", &model.Schema{Type: model.TypeArray, Items: &model.Schema{Type: model.TypeInteger}}, "[]int"},
		{"empty object", &model.Schema{Type: model.TypeObject}, "map[string]any"},
		{"object with additional properties", &model.Schema{Type: model.TypeObject, AdditionalProperties: &model.Schema{Type: model.TypeString}}, "map[string]string"},
		{"ref", &model.Schema{Ref: "#/components/schemas/Pet"}, "Pet"},
		{"ref with path", &model.Schema{Ref: "#/components/schemas/my_pet"}, "MyPet"},
		{"oneOf", &model.Schema{OneOf: []*model.Schema{{Type: model.TypeString}}}, "any"},
		{"anyOf", &model.Schema{AnyOf: []*model.Schema{{Type: model.TypeString}}}, "any"},
		{"allOf", &model.Schema{AllOf: []*model.Schema{{Type: model.TypeString}}}, "any"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GoType(tt.schema)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestGoZeroValue(t *testing.T) {
	tests := []struct {
		name     string
		schema   *model.Schema
		expected string
	}{
		{"nil", nil, "nil"},
		{"string", &model.Schema{Type: model.TypeString}, `""`},
		{"integer", &model.Schema{Type: model.TypeInteger}, "0"},
		{"number", &model.Schema{Type: model.TypeNumber}, "0"},
		{"boolean", &model.Schema{Type: model.TypeBoolean}, "false"},
		{"array", &model.Schema{Type: model.TypeArray}, "nil"},
		{"object", &model.Schema{Type: model.TypeObject}, "nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GoZeroValue(tt.schema)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestJSONTag(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		required bool
		expected string
	}{
		{"required", "id", true, "`json:\"id\"`"},
		{"optional", "name", false, "`json:\"name,omitempty\"`"},
		{"snake case required", "user_id", true, "`json:\"user_id\"`"},
		{"snake case optional", "created_at", false, "`json:\"created_at,omitempty\"`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JSONTag(tt.field, tt.required)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestNeedsTimeImport(t *testing.T) {
	tests := []struct {
		name     string
		schema   *model.Schema
		expected bool
	}{
		{"nil", nil, false},
		{"string", &model.Schema{Type: model.TypeString}, false},
		{"date-time", &model.Schema{Type: model.TypeString, Format: "date-time"}, true},
		{"date", &model.Schema{Type: model.TypeString, Format: "date"}, false},
		{"array of date-time", &model.Schema{Type: model.TypeArray, Items: &model.Schema{Type: model.TypeString, Format: "date-time"}}, true},
		{"object with date property", &model.Schema{
			Type: model.TypeObject,
			Properties: []model.Property{
				{Name: "created", Schema: &model.Schema{Type: model.TypeString, Format: "date-time"}},
			},
		}, true},
		{"object without date", &model.Schema{
			Type: model.TypeObject,
			Properties: []model.Property{
				{Name: "name", Schema: &model.Schema{Type: model.TypeString}},
			},
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NeedsTimeImport(tt.schema)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestTypeResolver_UUIDType(t *testing.T) {
	tests := []struct {
		name       string
		uuidPkg    string
		expected   string
		importPath string
	}{
		{"default string", "", "string", ""},
		{"explicit string", "string", "string", ""},
		{"google", "google", "uuid.UUID", "github.com/google/uuid"},
		{"gofrs", "gofrs", "uuid.UUID", "github.com/gofrs/uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewTypeResolver(&config.TypesConfig{UUIDPackage: tt.uuidPkg})
			schema := &model.Schema{Type: model.TypeString, Format: "uuid"}
			got := r.ResolveType(schema, "", "")
			require.Equal(t, tt.expected, got)

			importPath := r.UUIDImport()
			require.Equal(t, tt.importPath, importPath)
		})
	}
}

func TestTypeResolver_NestedObject(t *testing.T) {
	r := NewTypeResolver(&config.TypesConfig{})

	schema := &model.Schema{
		Type: model.TypeObject,
		Properties: []model.Property{
			{Name: "name", Schema: &model.Schema{Type: model.TypeString}},
		},
	}

	typeName := r.ResolveType(schema, "User", "Preferences")
	require.Equal(t, "UserPreferences", typeName)

	nested := r.NestedTypes()
	require.Len(t, nested, 1)
	require.Equal(t, "UserPreferences", nested[0].Name)
}

func TestTypeResolver_Union(t *testing.T) {
	r := NewTypeResolver(&config.TypesConfig{})

	schema := &model.Schema{
		OneOf: []*model.Schema{
			{Ref: "#/components/schemas/Cat"},
			{Ref: "#/components/schemas/Dog"},
		},
	}

	typeName := r.ResolveType(schema, "Pet", "Animal")
	require.Equal(t, "PetAnimal", typeName)

	nested := r.NestedTypes()
	require.Len(t, nested, 1)
	require.True(t, nested[0].IsUnion)
	require.Len(t, nested[0].Variants, 2)
}

func TestTypeResolver_UnionInlineVariantNames(t *testing.T) {
	object := func(prop string) *model.Schema {
		return &model.Schema{
			Type:       model.TypeObject,
			Properties: []model.Property{{Name: prop, Schema: &model.Schema{Type: model.TypeString}}},
		}
	}

	r := NewTypeResolver(&config.TypesConfig{})
	r.ResolveType(&model.Schema{
		AnyOf: []*model.Schema{
			{Type: model.TypeString},
			{Type: model.TypeInteger, Format: "int64"},
			{Type: model.TypeString, Format: "date-time"},
			{Type: model.TypeString, Format: "date"},
			{Type: model.TypeArray, Items: &model.Schema{Type: model.TypeString}},
			{Type: model.TypeArray, Items: &model.Schema{Type: model.TypeArray, Items: &model.Schema{Type: model.TypeString}}},
			{Type: model.TypeObject, AdditionalProperties: &model.Schema{Type: model.TypeString}},
			{Type: model.TypeObject},
			object("a"),
			object("b"),
			{Ref: "#/components/schemas/List"},
		},
	}, "Pet", "Value")

	var union ResolvedType
	for _, n := range r.NestedTypes() {
		if n.IsUnion {
			union = n
		}
	}

	type variant struct{ Name, TypeName string }
	var got []variant
	for _, v := range union.Variants {
		got = append(got, variant{v.Name, v.TypeName})
	}
	require.Equal(t, []variant{
		{"String", "string"},
		{"Int64", "int64"},
		{"Time", "time.Time"},
		{"StringList", "[]string"},
		{"StringListList", "[][]string"},
		{"StringMap", "map[string]string"},
		{"AnyMap", "map[string]any"},
		{"PetValueVariant", "PetValueVariant"},
		{"PetValueVariant2", "PetValueVariant2"},
		{"List", "List"},
	}, got)

	nested := map[string]*model.Schema{}
	for _, n := range r.NestedTypes() {
		nested[n.Name] = n.Schema
	}
	require.Equal(t, "a", nested["PetValueVariant"].Properties[0].Name)
	require.Equal(t, "b", nested["PetValueVariant2"].Properties[0].Name)
}

func TestTypeResolver_UnionInlineVariantsOfSameTypeShareAccessor(t *testing.T) {
	r := NewTypeResolver(&config.TypesConfig{})
	r.ResolveType(&model.Schema{
		AnyOf: []*model.Schema{
			{Type: model.TypeString},
			{Type: model.TypeString, Format: "uri"},
			{Type: model.TypeString, Format: "date-time"},
			{Type: model.TypeString, Format: "date"},
		},
	}, "Pet", "Value")

	variants := r.NestedTypes()[0].Variants
	require.Len(t, variants, 2)
	require.Equal(t, "String", variants[0].Name)
	require.Equal(t, "Time", variants[1].Name)
}

func TestTypeResolver_UnionRefKeepsNameOverInlineVariant(t *testing.T) {
	r := NewTypeResolver(&config.TypesConfig{})
	r.ResolveType(&model.Schema{
		OneOf: []*model.Schema{
			{Type: model.TypeString},
			{Ref: "#/components/schemas/String"},
		},
	}, "Pet", "Value")

	variants := r.NestedTypes()[0].Variants
	require.Equal(t, "String2", variants[0].Name)
	require.Equal(t, "string", variants[0].TypeName)
	require.Equal(t, "String", variants[1].Name)
}

func goNamed(name string, s model.Schema) *model.Schema {
	s.Extensions = &model.SchemaExtensions{GoName: name}
	return &s
}

func TestTypeResolver_UnionVariantGoNameOverridesAccessor(t *testing.T) {
	r := NewTypeResolver(&config.TypesConfig{})
	r.ResolveType(&model.Schema{
		AnyOf: []*model.Schema{
			{Type: model.TypeString},
			goNamed("StartsAt", model.Schema{Type: model.TypeString, Format: "date-time"}),
			goNamed("EndsAt", model.Schema{Type: model.TypeString, Format: "date-time"}),
		},
	}, "Pet", "Value")

	variants := r.NestedTypes()[0].Variants
	require.Len(t, variants, 3)
	require.Equal(t, "String", variants[0].Name)
	require.Equal(t, "StartsAt", variants[1].Name)
	require.Equal(t, "time.Time", variants[1].TypeName)
	require.Equal(t, "EndsAt", variants[2].Name)
	require.Empty(t, r.Warnings())
	require.NoError(t, r.Err())
}

func TestTypeResolver_UnionVariantGoNameIsReservedBeforeGeneratedNames(t *testing.T) {
	r := NewTypeResolver(&config.TypesConfig{})
	r.ResolveType(&model.Schema{
		AnyOf: []*model.Schema{
			{Type: model.TypeString},
			goNamed("String", model.Schema{Type: model.TypeInteger}),
		},
	}, "Pet", "Value")

	variants := r.NestedTypes()[0].Variants
	require.Equal(t, "String2", variants[0].Name)
	require.Equal(t, "String", variants[1].Name)
}

func TestTypeResolver_UnionWarnsWhenNameGetsSuffix(t *testing.T) {
	r := NewTypeResolver(&config.TypesConfig{})
	r.ResolveType(&model.Schema{
		OneOf: []*model.Schema{
			{Ref: "#/components/schemas/Time"},
			{Type: model.TypeString, Format: "date-time"},
		},
	}, "Pet", "Value")

	require.Equal(t, []string{
		"union PetValue: inline variant named AsTime2 because AsTime is taken; set x-oink-go-name to choose a name",
	}, r.Warnings())
}

func TestTypeResolver_UnionSameTypeVariantsDoNotWarn(t *testing.T) {
	r := NewTypeResolver(&config.TypesConfig{})
	r.ResolveType(&model.Schema{
		AnyOf: []*model.Schema{
			{Type: model.TypeString},
			{Type: model.TypeString, Format: "uri"},
		},
	}, "Pet", "Value")

	require.Empty(t, r.Warnings())
}

func TestTypeResolver_UnionGoNameClashIsAnError(t *testing.T) {
	tests := []struct {
		name     string
		variants []*model.Schema
	}{
		{"two go names", []*model.Schema{
			goNamed("Same", model.Schema{Type: model.TypeString}),
			goNamed("Same", model.Schema{Type: model.TypeInteger}),
		}},
		{"go name and ref", []*model.Schema{
			goNamed("Cat", model.Schema{Type: model.TypeString}),
			{Ref: "#/components/schemas/Cat"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewTypeResolver(&config.TypesConfig{})
			r.ResolveType(&model.Schema{OneOf: tt.variants}, "Pet", "Value")
			require.ErrorContains(t, r.Err(), "union PetValue: x-oink-go-name")
		})
	}
}

func TestTypeResolver_AllOf(t *testing.T) {
	r := NewTypeResolver(&config.TypesConfig{})

	schema := &model.Schema{
		AllOf: []*model.Schema{
			{Ref: "#/components/schemas/Base"},
			{Ref: "#/components/schemas/Extended"},
		},
	}

	typeName := r.ResolveType(schema, "Model", "Combined")
	require.Equal(t, "ModelCombined", typeName)

	nested := r.NestedTypes()
	require.Len(t, nested, 1)
	require.True(t, nested[0].IsAllOf)
}

