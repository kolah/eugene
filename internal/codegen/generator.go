package codegen

import (
	"fmt"

	"github.com/kolah/eugene/internal/config"
	"github.com/kolah/eugene/internal/golang"
	"github.com/kolah/eugene/internal/model"
	"github.com/kolah/eugene/internal/targets/client"
	"github.com/kolah/eugene/internal/targets/server"
	spectarget "github.com/kolah/eugene/internal/targets/spec"
	"github.com/kolah/eugene/internal/targets/strictserver"
	"github.com/kolah/eugene/internal/targets/types"
	"github.com/kolah/eugene/internal/templates"
	embeddedtmpl "github.com/kolah/eugene/templates"
)

type Generator struct {
	config *config.Config
	engine templates.Engine
}

type Output struct {
	Filename string
	Content  string
}

func New(cfg *config.Config) (*Generator, error) {
	if len(cfg.Go.OutputOptions.AdditionalInitialisms) > 0 {
		golang.SetAdditionalInitialisms(cfg.Go.OutputOptions.AdditionalInitialisms)
	}

	engine, err := templates.NewEngine(embeddedtmpl.FS, cfg.Templates.Dir, golang.TemplateFuncsWithResolver(&cfg.Go.Types))
	if err != nil {
		return nil, fmt.Errorf("creating template engine: %w", err)
	}

	return &Generator{
		config: cfg,
		engine: engine,
	}, nil
}

func (g *Generator) Generate(spec *model.Spec, specData []byte) ([]Output, error) {
	var outputs []Output

	if g.config.HasTarget("types") {
		target := types.New()
		content, err := target.Generate(g.engine, spec, g.config.Go.Package, &g.config.Go.Types, &g.config.Go.OutputOptions, g.config.Go.ImportMapping)
		if err != nil {
			return nil, fmt.Errorf("generating types: %w", err)
		}
		formatted, err := golang.Format([]byte(content))
		if err != nil {
			return nil, fmt.Errorf("formatting types: %w", err)
		}
		outputs = append(outputs, Output{
			Filename: "types.go",
			Content:  string(formatted),
		})
	}

	if g.config.HasTarget("server") {
		target, err := server.New(g.config.Go.ServerFramework)
		if err != nil {
			return nil, err
		}
		content, err := target.Generate(g.engine, spec, g.config.Go.Package)
		if err != nil {
			return nil, fmt.Errorf("generating server: %w", err)
		}
		formatted, err := golang.Format([]byte(content))
		if err != nil {
			return nil, fmt.Errorf("formatting server: %w", err)
		}
		outputs = append(outputs, Output{
			Filename: "server.go",
			Content:  string(formatted),
		})
	}

	if g.config.HasTarget("strict-server") {
		target, err := strictserver.New(g.config.Go.ServerFramework)
		if err != nil {
			return nil, err
		}
		// Generate strict types (request/response types + interface)
		typesContent, err := target.GenerateTypes(g.engine, spec, g.config.Go.Package)
		if err != nil {
			return nil, fmt.Errorf("generating strict types: %w", err)
		}
		typesFormatted, err := golang.Format([]byte(typesContent))
		if err != nil {
			return nil, fmt.Errorf("formatting strict types: %w", err)
		}
		outputs = append(outputs, Output{
			Filename: "strict_types.go",
			Content:  string(typesFormatted),
		})
		// Generate strict adapter (framework-specific handler wrapper)
		adapterContent, err := target.GenerateAdapter(g.engine, spec, g.config.Go.Package)
		if err != nil {
			return nil, fmt.Errorf("generating strict adapter: %w", err)
		}
		adapterFormatted, err := golang.Format([]byte(adapterContent))
		if err != nil {
			return nil, fmt.Errorf("formatting strict adapter: %w", err)
		}
		outputs = append(outputs, Output{
			Filename: "strict_server.go",
			Content:  string(adapterFormatted),
		})
	}

	if g.config.HasTarget("client") {
		target := client.New()
		content, err := target.Generate(g.engine, spec, g.config.Go.Package)
		if err != nil {
			return nil, fmt.Errorf("generating client: %w", err)
		}
		formatted, err := golang.Format([]byte(content))
		if err != nil {
			return nil, fmt.Errorf("formatting client: %w", err)
		}
		outputs = append(outputs, Output{
			Filename: "client.go",
			Content:  string(formatted),
		})
	}

	if g.config.HasTarget("spec") {
		target := spectarget.New()
		content, err := target.Generate(g.engine, specData, g.config.Go.Package)
		if err != nil {
			return nil, fmt.Errorf("generating spec: %w", err)
		}
		formatted, err := golang.Format([]byte(content))
		if err != nil {
			return nil, fmt.Errorf("formatting spec: %w", err)
		}
		outputs = append(outputs, Output{
			Filename: "spec.go",
			Content:  string(formatted),
		})
	}

	return outputs, nil
}
