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
	config        *config.Config
	engine        templates.Engine
	registry      *golang.EnumRegistry
	resolverState *golang.TemplateResolverState
}

type Output struct {
	Filename string
	Content  string
}

func New(cfg *config.Config) (*Generator, error) {
	if len(cfg.Go.OutputOptions.AdditionalInitialisms) > 0 {
		golang.SetAdditionalInitialisms(cfg.Go.OutputOptions.AdditionalInitialisms)
	}

	funcs, resolverState := golang.TemplateFuncsWithResolver(&cfg.Go.Types)
	engine, err := templates.NewEngine(embeddedtmpl.FS, cfg.Templates.Dir, funcs)
	if err != nil {
		return nil, fmt.Errorf("creating template engine: %w", err)
	}

	return &Generator{
		config:        cfg,
		engine:        engine,
		resolverState: resolverState,
	}, nil
}

func (g *Generator) Generate(spec *model.Spec, specData []byte) ([]Output, error) {
	var outputs []Output

	g.registry = golang.NewEnumRegistry()
	g.collectEnums(spec)

	var schemaNames []string
	for _, s := range spec.Schemas {
		schemaNames = append(schemaNames, golang.PascalCase(s.Name))
	}
	g.registry.AddReservedNames(schemaNames...)

	var opNames []string
	for _, op := range spec.Operations {
		base := golang.PascalCase(op.ID)
		opNames = append(opNames, base+"Response", base+"Request", base+"Params")
		opNames = append(opNames, base+"MultipartRequest", base+"FormRequest", base+"QueryParams")
		opNames = append(opNames, base+"RequestObject", base+"ResponseObject")
		for _, r := range op.Responses {
			opNames = append(opNames, base+r.StatusCode+"Response", base+r.StatusCode+"JSONResponse")
		}
	}
	g.registry.AddReservedNames(opNames...)

	g.registry.ResolveNames()
	g.resolverState.SetRegistry(g.registry)

	if g.config.Go.ServerFramework == "echo" && (g.config.HasTarget("server") || g.config.HasTarget("strict-server")) {
		content, err := g.engine.Execute("go/server/echo_router.tmpl", map[string]string{"Package": g.config.Go.Package})
		if err != nil {
			return nil, fmt.Errorf("generating router: %w", err)
		}
		formatted, err := golang.Format([]byte(content))
		if err != nil {
			return nil, fmt.Errorf("formatting router: %w", err)
		}
		outputs = append(outputs, Output{
			Filename: "router.eugene.go",
			Content:  string(formatted),
		})
	}

	if g.config.HasTarget("types") {
		target := types.New()
		content, err := target.Generate(g.engine, spec, g.config.Go.Package, &g.config.Go.Types, &g.config.Go.OutputOptions, g.config.Go.ImportMapping, g.registry)
		if err != nil {
			return nil, fmt.Errorf("generating types: %w", err)
		}
		formatted, err := golang.Format([]byte(content))
		if err != nil {
			return nil, fmt.Errorf("formatting types: %w", err)
		}
		outputs = append(outputs, Output{
			Filename: "types.eugene.go",
			Content:  string(formatted),
		})
	}

	if g.config.HasTarget("server") {
		target, err := server.New(g.config.Go.ServerFramework)
		if err != nil {
			return nil, err
		}
		content, err := target.Generate(g.engine, spec, g.config.Go.Package, &g.config.Go.Types, g.registry)
		if err != nil {
			return nil, fmt.Errorf("generating server: %w", err)
		}
		formatted, err := golang.Format([]byte(content))
		if err != nil {
			return nil, fmt.Errorf("formatting server: %w", err)
		}
		outputs = append(outputs, Output{
			Filename: "server.eugene.go",
			Content:  string(formatted),
		})
	}

	if g.config.HasTarget("strict-server") {
		target, err := strictserver.New(g.config.Go.ServerFramework)
		if err != nil {
			return nil, err
		}
		typesContent, err := target.GenerateTypes(g.engine, spec, g.config.Go.Package, &g.config.Go.Types, g.registry)
		if err != nil {
			return nil, fmt.Errorf("generating strict types: %w", err)
		}
		typesFormatted, err := golang.Format([]byte(typesContent))
		if err != nil {
			return nil, fmt.Errorf("formatting strict types: %w", err)
		}
		outputs = append(outputs, Output{
			Filename: "strict_types.eugene.go",
			Content:  string(typesFormatted),
		})
		adapterContent, err := target.GenerateAdapter(g.engine, spec, g.config.Go.Package, &g.config.Go.Types, g.registry)
		if err != nil {
			return nil, fmt.Errorf("generating strict adapter: %w", err)
		}
		adapterFormatted, err := golang.Format([]byte(adapterContent))
		if err != nil {
			return nil, fmt.Errorf("formatting strict adapter: %w", err)
		}
		outputs = append(outputs, Output{
			Filename: "strict_server.eugene.go",
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
			Filename: "client.eugene.go",
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
			Filename: "spec.eugene.go",
			Content:  string(formatted),
		})
	}

	return outputs, nil
}

// collectEnums walks the spec and collects all enum usages for stable naming.
func (g *Generator) collectEnums(spec *model.Spec) {
	// Collect from operation parameters
	for _, op := range spec.Operations {
		for _, p := range op.Parameters {
			if p.Schema != nil && len(p.Schema.Enum) > 0 {
				g.registry.CollectEnum(p.Name, op.ID, p.Schema.Enum)
			}
		}
	}

	for _, s := range spec.Schemas {
		for _, prop := range s.Properties {
			if prop.Schema != nil && len(prop.Schema.Enum) > 0 {
				g.registry.CollectEnum(prop.Name, s.Name, prop.Schema.Enum)
			}
		}
	}
}
