package loader

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func transformYAML(t *testing.T, paths, schemas string) error {
	t.Helper()
	doc := "openapi: 3.0.3\ninfo: {title: t, version: \"1\"}\npaths:\n" + indent(paths, "  ") +
		"\ncomponents:\n  schemas:\n" + indent(schemas, "    ")
	result, err := loadWithConfig([]byte(doc), nil)
	require.NoError(t, err)
	_, err = Transform(result)
	return err
}

func indent(s, prefix string) string {
	if strings.TrimSpace(s) == "" {
		return prefix + "{}"
	}
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

func op(params, responses string) string {
	if responses == "" {
		responses = `"204": {description: ok}`
	}
	return "/x:\n  get:\n    operationId: getX\n    summary: Get x\n" +
		params + "\n    responses:\n" + indent(responses, "      ")
}

func TestTransformAcceptsValidSpec(t *testing.T) {
	err := transformYAML(t,
		op(`    parameters:
      - {name: X-Request-ID, in: header, schema: {type: string}}
      - {name: "filter[name]", in: query, schema: {type: string}}`,
			`"200":
  description: "ok \"quoted\""
  headers:
    X-Rate-Limit: {schema: {type: integer}}
  content:
    application/json: {schema: {$ref: "#/components/schemas/Pet"}}
default: {description: error}`),
		`Pet:
  description: "multi\nline \"description\""
  type: object
  properties:
    "@type": {type: string}
    id:
      type: string
      x-oink-go-type: "map[string][]*uuid.UUID"
      x-oink-go-type-import: {path: github.com/google/uuid}
      x-oink-go-name: PetID
      x-oink-extra-tags: {validate: "required,email"}
Level:
  type: integer
  enum: [1, 2, 3]
Kind:
  type: string
  enum: ["a\"b", "c d"]`)
	require.NoError(t, err)
}

func TestTransformRejectsUnsafeSpecStrings(t *testing.T) {
	tests := []struct {
		name    string
		paths   string
		schemas string
		want    string
	}{
		{
			name:    "go type breaking out of the struct",
			schemas: "Pet:\n  type: object\n  properties:\n    name:\n      type: string\n      x-oink-go-type: 'string }; func init() { println(1) }; type _ struct { X string'",
			want:    `x-oink-go-type "string }`,
		},
		{
			name:    "go type calling a function",
			schemas: "Pet:\n  type: object\n  properties:\n    n:\n      type: string\n      x-oink-go-type: '[len(f())]int'",
			want:    "x-oink-go-type",
		},
		{
			name:    "go name that is not an identifier",
			schemas: "Pet:\n  type: object\n  properties:\n    n:\n      type: string\n      x-oink-go-name: 'A }; func init() {}; type B struct {'",
			want:    "x-oink-go-name",
		},
		{
			name:    "go type import path with a quote",
			schemas: "Pet:\n  type: object\n  properties:\n    n:\n      type: string\n      x-oink-go-type: x.T\n      x-oink-go-type-import: {path: 'x\"; import \"os'}",
			want:    "x-oink-go-type-import path",
		},
		{
			name:    "go type import alias that is not an identifier",
			schemas: "Pet:\n  type: object\n  properties:\n    n:\n      type: string\n      x-oink-go-type: x.T\n      x-oink-go-type-import: {path: x, alias: 'x \"os\"; y'}",
			want:    "x-oink-go-type-import alias",
		},
		{
			name:    "extra tag value with a backtick",
			schemas: "Pet:\n  type: object\n  properties:\n    n:\n      type: string\n      x-oink-extra-tags: {validate: 'a`}; func init() {}; type _ struct { Y string `'}",
			want:    "x-oink-extra-tags",
		},
		{
			name:    "extra tag key with a space",
			schemas: "Pet:\n  type: object\n  properties:\n    n:\n      type: string\n      x-oink-extra-tags: {'a b': c}",
			want:    "x-oink-extra-tags",
		},
		{
			name:    "property name with a backtick",
			schemas: "Pet:\n  type: object\n  properties:\n    'a`}; func init() {}; type _ struct { Y string `': {type: string}",
			want:    "property name",
		},
		{
			name:    "discriminator property name with a quote",
			schemas: "A: {type: object, properties: {k: {type: string}}}\nPet:\n  oneOf: [{$ref: '#/components/schemas/A'}]\n  discriminator: {propertyName: 'k\"'}",
			want:    "discriminator property name",
		},
		{
			name:    "discriminator mapping key with a backslash",
			schemas: "A: {type: object, properties: {k: {type: string}}}\nPet:\n  oneOf: [{$ref: '#/components/schemas/A'}]\n  discriminator: {propertyName: k, mapping: {'a\\': '#/components/schemas/A'}}",
			want:    "discriminator mapping key",
		},
		{
			name:    "integer enum with a string value",
			schemas: "Level:\n  type: integer\n  enum: [1, '2 + f()']",
			want:    "enum value",
		},
		{
			name:  "parameter name with a quote",
			paths: op("    parameters:\n      - {name: 'X-A\") + f() + (\"', in: header, schema: {type: string}}", ""),
			want:  "parameter name",
		},
		{
			name:  "summary with a newline",
			paths: "/x:\n  get:\n    operationId: getX\n    summary: \"Get x\\nfunc init() {}\"\n    responses:\n      \"204\": {description: ok}",
			want:  "summary",
		},
		{
			name:  "path with a quote",
			paths: "'/x\"+f()+\"':\n  get:\n    operationId: getX\n    responses:\n      \"204\": {description: ok}",
			want:  "path",
		},
		{
			name:  "media type with a quote",
			paths: op("", "\"200\":\n  description: ok\n  content:\n    'application/json\"': {schema: {type: string}}"),
			want:  "media type",
		},
		{
			name:  "response header name with a quote",
			paths: op("", "\"200\":\n  description: ok\n  headers:\n    'X-A\"': {schema: {type: string}}"),
			want:  "header name",
		},
		{
			name:  "status code that is not one",
			paths: op("", "'200 x': {description: ok}"),
			want:  "status code",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := transformYAML(t, tt.paths, tt.schemas)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestTransformAcceptsScalarCookies(t *testing.T) {
	err := transformYAML(t,
		op(`    parameters:
      - {name: sid, in: cookie, required: true, schema: {type: string}}
      - {name: visits, in: cookie, schema: {type: integer, format: int64}}
      - {name: ratio, in: cookie, schema: {type: number}}
      - {name: beta, in: cookie, schema: {type: boolean}}
      - {name: at, in: cookie, schema: {type: string, format: date-time}}
      - {name: theme, in: cookie, schema: {$ref: "#/components/schemas/Theme"}}
      - {name: mode, in: cookie, schema: {type: string, enum: [a, b]}}`, ""),
		`Theme: {type: string, enum: [light, dark]}`)
	require.NoError(t, err)
}

func TestTransformRejectsUnsupportedCookies(t *testing.T) {
	tests := map[string]string{
		"array":             `{type: array, items: {type: string}}`,
		"object":            `{type: object, properties: {a: {type: string}}}`,
		"object ref":        `{$ref: "#/components/schemas/Obj"}`,
		"binary string":     `{type: string, format: binary}`,
		"integer enum":      `{type: integer, enum: [1, 2]}`,
		"custom go type":    `{type: string, x-oink-go-type: time.Duration}`,
		"oneOf composition": `{oneOf: [{type: string}, {type: integer}]}`,
	}
	for name, schema := range tests {
		t.Run(name, func(t *testing.T) {
			err := transformYAML(t,
				op("    parameters:\n      - {name: prefs, in: cookie, schema: "+schema+"}", ""),
				`Obj: {type: object, properties: {a: {type: string}}}`)
			require.ErrorContains(t, err, `cookie parameter "prefs" in operation "getX"`)
		})
	}
}
