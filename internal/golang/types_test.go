package golang

import (
	"testing"

	"github.com/kolah/eugene/internal/config"
	"github.com/kolah/eugene/internal/model"
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
		{"string date", &model.Schema{Type: model.TypeString, Format: "date"}, "time.Time"},
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
		{"date", &model.Schema{Type: model.TypeString, Format: "date"}, true},
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

func TestTypeResolver_Reset(t *testing.T) {
	r := NewTypeResolver(&config.TypesConfig{})

	schema := &model.Schema{
		Type: model.TypeObject,
		Properties: []model.Property{
			{Name: "name", Schema: &model.Schema{Type: model.TypeString}},
		},
	}

	r.ResolveType(schema, "User", "Preferences")
	require.Len(t, r.NestedTypes(), 1)

	r.Reset()
	require.Empty(t, r.NestedTypes())
}
