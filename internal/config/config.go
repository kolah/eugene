package config

import (
	"fmt"
	"os"
	"slices"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
)

type Config struct {
	Spec           string         `koanf:"spec"`
	Templates      TemplateConfig `koanf:"templates"`
	ExcludeSchemas []string       `koanf:"exclude-schemas"`
	IncludeTags    []string       `koanf:"include-tags"`
	ExcludeTags    []string       `koanf:"exclude-tags"`
	Go             GoConfig       `koanf:"go"`
}

type GoConfig struct {
	OutputDir       string            `koanf:"output-dir"`
	Package         string            `koanf:"package"`
	ServerFramework string            `koanf:"server-framework"`
	Types           TypesConfig       `koanf:"types"`
	OutputOptions   OutputOptions     `koanf:"output-options"`
	ImportMapping   map[string]string `koanf:"import-mapping"`
	Targets         []string          `koanf:"targets"`
}

type TemplateConfig struct {
	Dir string `koanf:"dir"`
}

type TypesConfig struct {
	EnumStrategy     string `koanf:"enum-strategy"`
	UUIDPackage      string `koanf:"uuid-package"`
	NullableStrategy string `koanf:"nullable-strategy"`
}

type OutputOptions struct {
	EnableYAMLTags        bool     `koanf:"enable-yaml-tags"`
	AdditionalInitialisms []string `koanf:"additional-initialisms"`
}

// BindCommonFlags binds language-agnostic flags to the generate command
func BindCommonFlags(cmd *cobra.Command) {
	flags := cmd.PersistentFlags()

	flags.StringP("config", "c", "", "Config file path (default: eugene.yaml)")
	flags.StringP("spec", "s", "", "OpenAPI spec file path")
	flags.String("templates", "", "Custom templates directory")
	flags.StringSlice("exclude-schemas", nil, "Schemas to exclude")
	flags.StringSlice("include-tags", nil, "Tags to include (exclusive)")
	flags.StringSlice("exclude-tags", nil, "Tags to exclude")
	flags.Bool("dry-run", false, "Print output without writing files")
}

func Load(cmd *cobra.Command, targets []string) (*Config, error) {
	k := koanf.New(".")

	configFile, _ := cmd.Flags().GetString("config")
	if configFile == "" {
		configFile, _ = cmd.PersistentFlags().GetString("config")
	}
	if configFile == "" {
		if _, err := os.Stat("eugene.yaml"); err == nil {
			configFile = "eugene.yaml"
		}
	}

	if configFile != "" {
		if err := k.Load(file.Provider(configFile), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	flagsMap := buildFlagsMap(cmd)
	if len(flagsMap) > 0 {
		if err := k.Load(confmap.Provider(flagsMap, "."), nil); err != nil {
			return nil, fmt.Errorf("loading flags: %w", err)
		}
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	// CLI targets override config file targets
	if len(targets) > 0 {
		cfg.Go.Targets = targets
	}

	// Expand "all" target
	cfg.Go.Targets = expandTargets(cfg.Go.Targets)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func expandTargets(targets []string) []string {
	var result []string
	for _, t := range targets {
		if t == "all" {
			result = append(result, "types", "server", "client", "spec", "strict-server")
		} else {
			result = append(result, t)
		}
	}
	return result
}

func buildFlagsMap(cmd *cobra.Command) map[string]any {
	m := make(map[string]any)

	getString := func(name string) string {
		if v, err := cmd.Flags().GetString(name); err == nil && v != "" {
			return v
		}
		if v, err := cmd.PersistentFlags().GetString(name); err == nil && v != "" {
			return v
		}
		return ""
	}

	getStringSlice := func(name string) []string {
		if v, err := cmd.Flags().GetStringSlice(name); err == nil && len(v) > 0 {
			return v
		}
		if v, err := cmd.PersistentFlags().GetStringSlice(name); err == nil && len(v) > 0 {
			return v
		}
		return nil
	}

	flagChanged := func(name string) bool {
		return cmd.Flags().Changed(name) || cmd.PersistentFlags().Changed(name)
	}

	getBool := func(name string) bool {
		if v, err := cmd.Flags().GetBool(name); err == nil {
			return v
		}
		if v, err := cmd.PersistentFlags().GetBool(name); err == nil {
			return v
		}
		return false
	}

	if v := getString("spec"); v != "" {
		m["spec"] = v
	}
	if v := getString("output-dir"); v != "" {
		m["go.output-dir"] = v
	}
	if v := getString("templates"); v != "" {
		m["templates.dir"] = v
	}
	if v := getStringSlice("exclude-schemas"); len(v) > 0 {
		m["exclude-schemas"] = v
	}
	if v := getStringSlice("include-tags"); len(v) > 0 {
		m["include-tags"] = v
	}
	if v := getStringSlice("exclude-tags"); len(v) > 0 {
		m["exclude-tags"] = v
	}

	// Go-specific flags (under go. namespace)
	if v := getString("package"); v != "" {
		m["go.package"] = v
	}
	if v := getString("server-framework"); v != "" {
		m["go.server-framework"] = v
	}
	if v := getString("enum-strategy"); v != "" {
		m["go.types.enum-strategy"] = v
	}
	if v := getString("uuid-package"); v != "" {
		m["go.types.uuid-package"] = v
	}
	if v := getString("nullable-strategy"); v != "" {
		m["go.types.nullable-strategy"] = v
	}
	if flagChanged("enable-yaml-tags") {
		m["go.output-options.enable-yaml-tags"] = getBool("enable-yaml-tags")
	}
	if v := getStringSlice("additional-initialisms"); len(v) > 0 {
		m["go.output-options.additional-initialisms"] = v
	}

	return m
}

func (c *Config) Validate() error {
	if c.Spec == "" {
		return fmt.Errorf("spec file is required")
	}
	if c.Go.Package == "" {
		return fmt.Errorf("package name is required")
	}
	if c.Go.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	validFrameworks := map[string]bool{"": true, "echo": true, "chi": true, "stdlib": true}
	if !validFrameworks[c.Go.ServerFramework] {
		return fmt.Errorf("invalid server framework: %s (valid: echo, chi, stdlib)", c.Go.ServerFramework)
	}

	validEnumStrategies := map[string]bool{"": true, "const": true, "type": true, "struct": true}
	if !validEnumStrategies[c.Go.Types.EnumStrategy] {
		return fmt.Errorf("invalid enum strategy: %s (valid: const, type, struct)", c.Go.Types.EnumStrategy)
	}

	validUUIDPackages := map[string]bool{"": true, "string": true, "google": true, "gofrs": true}
	if !validUUIDPackages[c.Go.Types.UUIDPackage] {
		return fmt.Errorf("invalid uuid package: %s (valid: string, google, gofrs)", c.Go.Types.UUIDPackage)
	}

	validNullableStrategies := map[string]bool{"": true, "pointer": true, "nullable": true}
	if !validNullableStrategies[c.Go.Types.NullableStrategy] {
		return fmt.Errorf("invalid nullable strategy: %s (valid: pointer, nullable)", c.Go.Types.NullableStrategy)
	}

	validTargets := map[string]bool{
		"types": true, "server": true, "client": true,
		"spec": true, "strict-server": true,
	}
	for _, t := range c.Go.Targets {
		if !validTargets[t] {
			return fmt.Errorf("invalid target: %s (valid: types, server, client, spec, strict-server)", t)
		}
	}

	return nil
}

// HasTarget checks if a specific target should be generated
func (c *Config) HasTarget(target string) bool {
	return slices.Contains(c.Go.Targets, target)
}
