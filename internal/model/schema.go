package model

type Schema struct {
	Name        string
	Description string
	Type        SchemaType
	Format      string
	Nullable    bool
	Deprecated  bool
	Default     any
	Example     any

	// Object properties
	Properties []Property
	Required   []string

	// Array items
	Items *Schema

	// Enum values
	Enum []any

	// Composition
	AllOf []*Schema
	OneOf []*Schema
	AnyOf []*Schema

	// Discriminator for oneOf/anyOf polymorphism
	Discriminator *Discriminator

	// Reference
	Ref string

	// Additional properties for maps
	AdditionalProperties *Schema

	// Constraints
	Minimum          *float64
	Maximum          *float64
	MinLength        *int64
	MaxLength        *int64
	Pattern          string
	MinItems         *int64
	MaxItems         *int64
	UniqueItems      bool
	MinProperties    *int64
	MaxProperties    *int64
	ExclusiveMinimum bool
	ExclusiveMaximum bool

	// x-oink-* extensions
	Extensions *SchemaExtensions
}

// SchemaExtensions holds x-oink-* extension values for customizing code generation.
type SchemaExtensions struct {
	// GoType overrides the generated Go type (e.g., "time.Duration")
	GoType string
	// GoTypeImport specifies the import path for the custom Go type
	GoTypeImport *GoTypeImport
	// GoName overrides the generated field/type name
	GoName string
	// ExtraTags adds additional struct tags (e.g., {"validate": "required,email"})
	ExtraTags map[string]string
	// OmitEmpty forces the omitempty JSON tag option
	OmitEmpty *bool
	// OmitZero forces the omitzero JSON tag option (Go 1.24+)
	OmitZero *bool
	// JSONIgnore excludes the field from JSON marshaling
	JSONIgnore bool
}

// GoTypeImport specifies an import for a custom Go type.
type GoTypeImport struct {
	Path  string // Import path (e.g., "time", "github.com/google/uuid")
	Alias string // Optional import alias
}

type SchemaType string

const (
	TypeString  SchemaType = "string"
	TypeNumber  SchemaType = "number"
	TypeInteger SchemaType = "integer"
	TypeBoolean SchemaType = "boolean"
	TypeArray   SchemaType = "array"
	TypeObject  SchemaType = "object"
	TypeNull    SchemaType = "null"
)

type Property struct {
	Name   string
	Schema *Schema
}

type Discriminator struct {
	PropertyName string
	Mapping      map[string]string
}

type SecurityScheme struct {
	Name         string
	Type         SecuritySchemeType
	Description  string
	In           string
	Scheme       string
	BearerFormat string
	Flows        *OAuthFlows
}

type SecuritySchemeType string

const (
	SecurityTypeAPIKey        SecuritySchemeType = "apiKey"
	SecurityTypeHTTP          SecuritySchemeType = "http"
	SecurityTypeOAuth2        SecuritySchemeType = "oauth2"
	SecurityTypeOpenIDConnect SecuritySchemeType = "openIdConnect"
	SecurityTypeMutualTLS     SecuritySchemeType = "mutualTLS"
)

type OAuthFlows struct {
	Implicit          *OAuthFlow
	Password          *OAuthFlow
	ClientCredentials *OAuthFlow
	AuthorizationCode *OAuthFlow
	DeviceCode        *OAuthFlow // OpenAPI 3.2
}

type OAuthFlow struct {
	AuthorizationURL string
	TokenURL         string
	RefreshURL       string
	DeviceAuthURL    string // OpenAPI 3.2
	Scopes           map[string]string
}
