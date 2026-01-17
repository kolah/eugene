package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kolah/eugene/internal/codegen"
	"github.com/kolah/eugene/internal/config"
	"github.com/kolah/eugene/internal/loader"
	"github.com/spf13/cobra"
)

func NewGoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "go",
		Short: "Generate Go code from OpenAPI spec",
	}

	flags := cmd.PersistentFlags()
	flags.StringP("output-dir", "o", "", "Output directory for generated Go code")
	flags.StringP("package", "p", "", "Go package name")
	flags.StringP("server-framework", "f", "", "Server framework: echo, chi, stdlib")
	flags.String("enum-strategy", "", "Enum strategy: const, type, struct")
	flags.String("uuid-package", "", "UUID type: string, google, gofrs")
	flags.String("nullable-strategy", "", "Nullable strategy: pointer, nullable")
	flags.Bool("enable-yaml-tags", false, "Generate yaml tags")
	flags.StringSlice("additional-initialisms", nil, "Additional initialisms")

	cmd.AddCommand(
		newGoTypesCmd(),
		newGoServerCmd(),
		newGoStrictServerCmd(),
		newGoClientCmd(),
		newGoSpecCmd(),
		newGoAllCmd(),
	)

	return cmd
}

func newGoTypesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "types",
		Short: "Generate Go type definitions",
		RunE:  runGoGenerate("types"),
	}
}

func newGoServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Generate Go server code",
		RunE:  runGoGenerate("server"),
	}
}

func newGoStrictServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "strict-server",
		Short: "Generate Go strict server with typed responses",
		RunE:  runGoGenerate("strict-server"),
	}
}

func newGoClientCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "client",
		Short: "Generate Go HTTP client",
		RunE:  runGoGenerate("client"),
	}
}

func newGoSpecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "spec",
		Short: "Generate embedded OpenAPI spec",
		RunE:  runGoGenerate("spec"),
	}
}

func newGoAllCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "all",
		Short: "Generate all Go targets (types, server, client, spec, strict-server)",
		RunE:  runGoGenerate("all"),
	}
}

func runGoGenerate(target string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cmd, expandTargets(target))
		if err != nil {
			return err
		}

		result, err := loader.LoadFile(cfg.Spec)
		if err != nil {
			return fmt.Errorf("loading spec: %w", err)
		}

		for _, w := range result.Warnings {
			cmd.PrintErrf("Warning: %s\n", w)
		}

		spec, err := loader.Transform(result)
		if err != nil {
			return fmt.Errorf("transforming spec: %w", err)
		}

		cmd.PrintErrf("Loaded OpenAPI %s: %s v%s\n", result.Version, spec.Info.Title, spec.Info.Version)
		cmd.PrintErrf("  Schemas: %d\n", len(spec.Schemas))
		cmd.PrintErrf("  Operations: %d\n", len(spec.Operations))

		gen, err := codegen.New(cfg)
		if err != nil {
			return fmt.Errorf("creating generator: %w", err)
		}

		outputs, err := gen.Generate(spec, result.RawData)
		if err != nil {
			return fmt.Errorf("generating code: %w", err)
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			for _, out := range outputs {
				cmd.Printf("// %s\n%s\n", out.Filename, out.Content)
			}
			return nil
		}

		if err := os.MkdirAll(cfg.Go.OutputDir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}

		for _, out := range outputs {
			path := filepath.Join(cfg.Go.OutputDir, out.Filename)
			if err := os.WriteFile(path, []byte(out.Content), 0644); err != nil {
				return fmt.Errorf("writing %s: %w", path, err)
			}
			cmd.PrintErrf("Written: %s\n", path)
		}

		return nil
	}
}

func expandTargets(target string) []string {
	if target == "all" {
		return []string{"types", "server", "client", "spec", "strict-server"}
	}
	return []string{target}
}
