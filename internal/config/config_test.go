package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir:       "output",
					Package:         "gen",
					ServerFramework: "echo",
					Types: TypesConfig{
						EnumStrategy: "const",
						UUIDPackage:  "string",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing spec",
			config: Config{
				Go: GoConfig{OutputDir: "output", Package: "gen"},
			},
			wantErr:     true,
			errContains: "spec file is required",
		},
		{
			name: "missing package",
			config: Config{
				Spec: "spec.yaml",
				Go:   GoConfig{OutputDir: "output"},
			},
			wantErr:     true,
			errContains: "package name is required",
		},
		{
			name: "missing output dir",
			config: Config{
				Spec: "spec.yaml",
				Go:   GoConfig{Package: "gen"},
			},
			wantErr:     true,
			errContains: "output directory is required",
		},
		{
			name: "invalid server framework",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir:       "output",
					Package:         "gen",
					ServerFramework: "invalid",
				},
			},
			wantErr:     true,
			errContains: "invalid server framework",
		},
		{
			name: "valid echo framework",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir:       "output",
					Package:         "gen",
					ServerFramework: "echo",
				},
			},
			wantErr: false,
		},
		{
			name: "valid chi framework",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir:       "output",
					Package:         "gen",
					ServerFramework: "chi",
				},
			},
			wantErr: false,
		},
		{
			name: "valid stdlib framework",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir:       "output",
					Package:         "gen",
					ServerFramework: "stdlib",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid enum strategy",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir: "output",
					Package:   "gen",
					Types:     TypesConfig{EnumStrategy: "invalid"},
				},
			},
			wantErr:     true,
			errContains: "invalid enum strategy",
		},
		{
			name: "valid enum strategy const",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir: "output",
					Package:   "gen",
					Types:     TypesConfig{EnumStrategy: "const"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid enum strategy type",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir: "output",
					Package:   "gen",
					Types:     TypesConfig{EnumStrategy: "type"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid enum strategy struct",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir: "output",
					Package:   "gen",
					Types:     TypesConfig{EnumStrategy: "struct"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid uuid package",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir: "output",
					Package:   "gen",
					Types:     TypesConfig{UUIDPackage: "invalid"},
				},
			},
			wantErr:     true,
			errContains: "invalid uuid package",
		},
		{
			name: "valid uuid package string",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir: "output",
					Package:   "gen",
					Types:     TypesConfig{UUIDPackage: "string"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid uuid package google",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir: "output",
					Package:   "gen",
					Types:     TypesConfig{UUIDPackage: "google"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid uuid package gofrs",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir: "output",
					Package:   "gen",
					Types:     TypesConfig{UUIDPackage: "gofrs"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty enum strategy is valid",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir: "output",
					Package:   "gen",
					Types:     TypesConfig{EnumStrategy: ""},
				},
			},
			wantErr: false,
		},
		{
			name: "empty uuid package is valid",
			config: Config{
				Spec: "spec.yaml",
				Go: GoConfig{
					OutputDir: "output",
					Package:   "gen",
					Types:     TypesConfig{UUIDPackage: ""},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `
spec: api.yaml
go:
  output-dir: ./output
  package: gen
  server-framework: echo
  types:
    enum-strategy: const
`
	configPath := filepath.Join(tmpDir, "eugene.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change to temp dir so eugene.yaml is found
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	BindCommonFlags(cmd)
	bindGoFlags(cmd)

	cfg, err := Load(cmd, []string{"types", "server"})
	require.NoError(t, err)

	require.Equal(t, "api.yaml", cfg.Spec)
	require.Equal(t, "gen", cfg.Go.Package)
	require.Equal(t, "./output", cfg.Go.OutputDir)
	require.Equal(t, "echo", cfg.Go.ServerFramework)
	require.Equal(t, "const", cfg.Go.Types.EnumStrategy)
	require.True(t, cfg.HasTarget("types"))
	require.True(t, cfg.HasTarget("server"))
	require.False(t, cfg.HasTarget("client"))
}

func TestLoadFlagsOverrideFile(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `
spec: api.yaml
go:
  output-dir: ./output
  package: gen
  server-framework: echo
`
	configPath := filepath.Join(tmpDir, "eugene.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	BindCommonFlags(cmd)
	bindGoFlags(cmd)

	// Set flags that should override file config
	cmd.Flags().Set("server-framework", "chi")

	cfg, err := Load(cmd, []string{"client"})
	require.NoError(t, err)

	// Flags should override
	require.Equal(t, "chi", cfg.Go.ServerFramework)
	require.True(t, cfg.HasTarget("client"))
}

func TestLoadWithExplicitConfigPath(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `
spec: custom.yaml
go:
  output-dir: ./custom
  package: custom
`
	configPath := filepath.Join(tmpDir, "custom-config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cmd := &cobra.Command{}
	BindCommonFlags(cmd)
	bindGoFlags(cmd)
	cmd.PersistentFlags().Set("config", configPath)

	cfg, err := Load(cmd, []string{"types"})
	require.NoError(t, err)

	require.Equal(t, "custom.yaml", cfg.Spec)
	require.Equal(t, "custom", cfg.Go.Package)
	require.Equal(t, "./custom", cfg.Go.OutputDir)
}

func TestBuildFlagsMap(t *testing.T) {
	cmd := &cobra.Command{}
	BindCommonFlags(cmd)
	bindGoFlags(cmd)

	cmd.PersistentFlags().Set("spec", "test.yaml")
	cmd.Flags().Set("package", "testpkg")
	cmd.Flags().Set("output-dir", "./out")
	cmd.Flags().Set("server-framework", "chi")
	cmd.Flags().Set("enum-strategy", "type")

	m := buildFlagsMap(cmd)

	require.Equal(t, "test.yaml", m["spec"])
	require.Equal(t, "testpkg", m["go.package"])
	require.Equal(t, "./out", m["go.output-dir"])
	require.Equal(t, "chi", m["go.server-framework"])
	require.Equal(t, "type", m["go.types.enum-strategy"])
}

func TestHasTarget(t *testing.T) {
	cfg := &Config{
		Targets: []string{"types", "server"},
	}

	require.True(t, cfg.HasTarget("types"))
	require.True(t, cfg.HasTarget("server"))
	require.False(t, cfg.HasTarget("client"))
	require.False(t, cfg.HasTarget("spec"))
}

// Helper to bind Go-specific flags for testing
func bindGoFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringP("output-dir", "o", "", "Output directory for generated Go code")
	flags.StringP("package", "p", "", "Go package name")
	flags.StringP("server-framework", "f", "", "Server framework: echo, chi, stdlib")
	flags.String("enum-strategy", "", "Enum strategy: const, type, struct")
	flags.String("uuid-package", "", "UUID type: string, google, gofrs")
	flags.String("nullable-strategy", "", "Nullable strategy: pointer, nullable")
	flags.Bool("enable-yaml-tags", false, "Generate yaml tags")
	flags.StringSlice("additional-initialisms", nil, "Additional initialisms")
}
