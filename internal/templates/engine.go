package templates

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Engine interface {
	Execute(name string, data any) (string, error)
}

type TextTemplateEngine struct {
	templates *template.Template
	funcs     template.FuncMap
	embedded  embed.FS
	customDir string
}

func NewEngine(embedded embed.FS, customDir string, funcs template.FuncMap) (*TextTemplateEngine, error) {
	e := &TextTemplateEngine{
		embedded:  embedded,
		customDir: customDir,
		funcs:     funcs,
	}
	if err := e.load(); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *TextTemplateEngine) load() error {
	e.templates = template.New("").Funcs(e.funcs)

	err := fs.WalkDir(e.embedded, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}
		content, err := e.embedded.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading embedded template %s: %w", path, err)
		}
		name := strings.TrimPrefix(path, "templates/")
		_, err = e.templates.New(name).Parse(string(content))
		if err != nil {
			return fmt.Errorf("parsing embedded template %s: %w", path, err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("loading embedded templates: %w", err)
	}

	if e.customDir != "" {
		err = filepath.WalkDir(e.customDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading custom template %s: %w", path, err)
			}
			relPath, _ := filepath.Rel(e.customDir, path)
			_, err = e.templates.New(relPath).Parse(string(content))
			if err != nil {
				return fmt.Errorf("parsing custom template %s: %w", path, err)
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("loading custom templates: %w", err)
		}
	}

	return nil
}

func (e *TextTemplateEngine) Execute(name string, data any) (string, error) {
	tmpl := e.templates.Lookup(name)
	if tmpl == nil {
		return "", fmt.Errorf("template not found: %s", name)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template %s: %w", name, err)
	}

	return buf.String(), nil
}
