package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kolah/eugene/internal/codegen"
	"github.com/kolah/eugene/internal/config"
	"github.com/kolah/eugene/internal/loader"
	"github.com/stretchr/testify/require"
)

func TestGeneratedCodeCompiles(t *testing.T) {
	tests := []struct {
		name             string
		targets          []string // types, server, client
		serverFramework  string
		enumStrategy     string
		uuidPackage      string
		nullableStrategy string
		enableYAMLTags   bool
		outputDir        string
		specFile         string // optional, defaults to routing.yaml
	}{
		// Enum strategy tests
		{
			name:         "types_const",
			targets:      []string{"types"},
			enumStrategy: "const",
			uuidPackage:  "string",
			outputDir:    "generated/types_const",
			specFile:     "testdata/specs/types/enums.yaml",
		},
		{
			name:         "types_type",
			targets:      []string{"types"},
			enumStrategy: "type",
			uuidPackage:  "string",
			outputDir:    "generated/types_type",
			specFile:     "testdata/specs/types/enums.yaml",
		},
		{
			name:         "types_struct",
			targets:      []string{"types"},
			enumStrategy: "struct",
			uuidPackage:  "string",
			outputDir:    "generated/types_struct",
			specFile:     "testdata/specs/types/enums.yaml",
		},
		// Server framework tests
		{
			name:            "server_echo",
			targets:         []string{"types", "server"},
			serverFramework: "echo",
			outputDir:       "generated/server_echo",
			specFile:        "testdata/specs/routing.yaml",
		},
		{
			name:            "server_chi",
			targets:         []string{"types", "server"},
			serverFramework: "chi",
			outputDir:       "generated/server_chi",
			specFile:        "testdata/specs/routing.yaml",
		},
		{
			name:            "server_stdlib",
			targets:         []string{"types", "server"},
			serverFramework: "stdlib",
			outputDir:       "generated/server_stdlib",
			specFile:        "testdata/specs/routing.yaml",
		},
		// Client generation test
		{
			name:      "client",
			targets:   []string{"types", "client"},
			outputDir: "generated/client",
			specFile:  "testdata/specs/routing.yaml",
		},
		// Full generation test (types + server + client)
		{
			name:            "full_echo",
			targets:         []string{"types", "server", "client"},
			serverFramework: "echo",
			outputDir:       "generated/full_echo",
			specFile:        "testdata/specs/routing.yaml",
		},
		// OpenAPI 3.2 features tests
		{
			name:            "openapi32_echo",
			targets:         []string{"types", "server", "client"},
			serverFramework: "echo",
			outputDir:       "generated/openapi32_echo",
			specFile:        "testdata/specs/openapi32/features.yaml",
		},
		{
			name:            "openapi32_chi",
			targets:         []string{"types", "server", "client"},
			serverFramework: "chi",
			outputDir:       "generated/openapi32_chi",
			specFile:        "testdata/specs/openapi32/features.yaml",
		},
		{
			name:            "openapi32_stdlib",
			targets:         []string{"types", "server", "client"},
			serverFramework: "stdlib",
			outputDir:       "generated/openapi32_stdlib",
			specFile:        "testdata/specs/openapi32/features.yaml",
		},
		// Spec embedding test
		{
			name:      "spec_embed",
			targets:   []string{"spec"},
			outputDir: "generated/spec_embed",
			specFile:  "testdata/specs/routing.yaml",
		},
		// Nullable types test
		{
			name:             "types_nullable",
			targets:          []string{"types"},
			nullableStrategy: "nullable",
			outputDir:        "generated/types_nullable",
			specFile:         "testdata/specs/types/nullable.yaml",
		},
		// Strict server tests
		{
			name:            "strict_echo",
			targets:         []string{"types", "strict-server"},
			serverFramework: "echo",
			outputDir:       "generated/strict_echo",
			specFile:        "testdata/specs/routing.yaml",
		},
		{
			name:            "strict_chi",
			targets:         []string{"types", "strict-server"},
			serverFramework: "chi",
			outputDir:       "generated/strict_chi",
			specFile:        "testdata/specs/routing.yaml",
		},
		{
			name:            "strict_stdlib",
			targets:         []string{"types", "strict-server"},
			serverFramework: "stdlib",
			outputDir:       "generated/strict_stdlib",
			specFile:        "testdata/specs/routing.yaml",
		},
		// Extensions test
		{
			name:      "extensions",
			targets:   []string{"types"},
			outputDir: "generated/extensions",
			specFile:  "testdata/specs/extensions/x-oink.yaml",
		},
		// YAML tags test
		{
			name:           "yaml_tags",
			targets:        []string{"types"},
			enableYAMLTags: true,
			outputDir:      "generated/yaml_tags",
			specFile:       "testdata/specs/routing.yaml",
		},
		{
			name:      "types_discriminators",
			targets:   []string{"types"},
			outputDir: "generated/types_discriminators",
			specFile:  "testdata/specs/types/discriminators.yaml",
		},
		{
			name:      "types_allof",
			targets:   []string{"types"},
			outputDir: "generated/types_allof",
			specFile:  "testdata/specs/types/allof.yaml",
		},
		{
			name:      "types_anyof",
			targets:   []string{"types"},
			outputDir: "generated/types_anyof",
			specFile:  "testdata/specs/types/anyof.yaml",
		},
		{
			name:      "types_formats",
			targets:   []string{"types"},
			outputDir: "generated/types_formats",
			specFile:  "testdata/specs/types/formats.yaml",
		},
		// Parameter types test
		{
			name:            "params",
			targets:         []string{"types", "server"},
			serverFramework: "echo",
			outputDir:       "generated/params",
			specFile:        "testdata/specs/parameters/all-param-types.yaml",
		},
		// Content type tests
		{
			name:            "multipart",
			targets:         []string{"types", "server", "client"},
			serverFramework: "echo",
			outputDir:       "generated/multipart",
			specFile:        "testdata/specs/content/multipart.yaml",
		},
		{
			name:            "formurlencoded",
			targets:         []string{"types", "server", "client"},
			serverFramework: "echo",
			outputDir:       "generated/formurlencoded",
			specFile:        "testdata/specs/content/formurlencoded.yaml",
		},
		{
			name:            "sse",
			targets:         []string{"types", "server"},
			serverFramework: "echo",
			outputDir:       "generated/sse",
			specFile:        "testdata/specs/content/sse.yaml",
		},
		// Error responses test
		{
			name:            "errors",
			targets:         []string{"types", "server"},
			serverFramework: "echo",
			outputDir:       "generated/errors",
			specFile:        "testdata/specs/responses/errors.yaml",
		},
		// Security test
		{
			name:            "security",
			targets:         []string{"types", "server"},
			serverFramework: "echo",
			outputDir:       "generated/security",
			specFile:        "testdata/specs/security/auth.yaml",
		},
		// OpenAPI 3.2 webhooks test
		{
			name:      "webhooks",
			targets:   []string{"types"},
			outputDir: "generated/webhooks",
			specFile:  "testdata/specs/openapi32/webhooks.yaml",
		},
		// OpenAPI 3.2 callbacks test
		{
			name:            "callbacks",
			targets:         []string{"types", "server"},
			serverFramework: "echo",
			outputDir:       "generated/callbacks",
			specFile:        "testdata/specs/openapi32/callbacks.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir, err := os.Getwd()
			require.NoError(t, err)

			specFile := tt.specFile
			if specFile == "" {
				specFile = "testdata/specs/routing.yaml"
			}
			specPath := filepath.Join(testDir, specFile)
			outputPath := filepath.Join(testDir, tt.outputDir)

			// Clean and create output directory
			err = os.RemoveAll(outputPath)
			require.NoError(t, err)
			err = os.MkdirAll(outputPath, 0755)
			require.NoError(t, err)

			// Load spec
			result, err := loader.LoadFile(specPath)
			require.NoError(t, err, "failed to load spec")

			spec, err := loader.Transform(result)
			require.NoError(t, err, "failed to transform spec")

			// Create config with new structure
			serverFramework := tt.serverFramework
			if serverFramework == "" {
				serverFramework = "echo"
			}

			cfg := &config.Config{
				Spec:    specPath,
				Targets: tt.targets,
				Go: config.GoConfig{
					OutputDir:       outputPath,
					Package:         "gen",
					ServerFramework: serverFramework,
					Types: config.TypesConfig{
						EnumStrategy:     tt.enumStrategy,
						UUIDPackage:      tt.uuidPackage,
						NullableStrategy: tt.nullableStrategy,
					},
					OutputOptions: config.OutputOptions{
						EnableYAMLTags: tt.enableYAMLTags,
					},
				},
			}

			gen, err := codegen.New(cfg)
			require.NoError(t, err, "failed to create generator")

			// Generate
			outputs, err := gen.Generate(spec, result.RawData)
			require.NoError(t, err, "failed to generate")

			// Write generated files
			for _, o := range outputs {
				filePath := filepath.Join(outputPath, o.Filename)
				err := os.WriteFile(filePath, []byte(o.Content), 0644)
				require.NoError(t, err, "failed to write %s", o.Filename)
			}

			// Verify the code compiles
			cmd := exec.Command("go", "build", "./...")
			cmd.Dir = outputPath
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "generated code failed to compile:\n%s", string(output))
		})
	}
}

func TestCustomTemplateOverride(t *testing.T) {
	testDir, err := os.Getwd()
	require.NoError(t, err)

	specPath := filepath.Join(testDir, "testdata/specs/routing.yaml")
	customTemplatesDir := filepath.Join(testDir, "testdata/custom-templates")
	outputPath := filepath.Join(testDir, "generated/custom_template")

	// Clean and create output directory
	err = os.RemoveAll(outputPath)
	require.NoError(t, err)
	err = os.MkdirAll(outputPath, 0755)
	require.NoError(t, err)

	// Load spec
	result, err := loader.LoadFile(specPath)
	require.NoError(t, err)

	spec, err := loader.Transform(result)
	require.NoError(t, err)

	// Create config with custom templates
	cfg := &config.Config{
		Spec: specPath,
		Templates: config.TemplateConfig{
			Dir: customTemplatesDir,
		},
		Targets: []string{"types"},
		Go: config.GoConfig{
			OutputDir: outputPath,
			Package:   "gen",
		},
	}

	gen, err := codegen.New(cfg)
	require.NoError(t, err)

	outputs, err := gen.Generate(spec, result.RawData)
	require.NoError(t, err)

	// Check that custom template marker is present
	var typesContent string
	for _, o := range outputs {
		if o.Filename == "types.go" {
			typesContent = o.Content
			break
		}
	}

	require.True(t, strings.Contains(typesContent, "CUSTOM TEMPLATE"), "custom template was not used")
}
