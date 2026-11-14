package loader

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
	"unicode"

	"github.com/kolah/eugene/v2/internal/model"
)

var (
	statusCodePattern = regexp.MustCompile(`^([1-5][0-9][0-9]|[1-5]XX|default)$`)
	tagKeyPattern     = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	numberPattern     = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)
)

// validate rejects spec strings that would be copied into generated Go source
// in a form that changes its meaning: a quote, backtick or backslash ending a
// string literal or struct tag, a newline ending a line comment, or an
// extension that is not the Go identifier or type it claims to be. It also
// rejects cookie parameters the generated code cannot bind.
func validate(spec *model.Spec) error {
	v := &validator{seen: map[*model.Schema]bool{}}
	for i := range spec.Schemas {
		v.schema(&spec.Schemas[i], "schema "+spec.Schemas[i].Name)
	}
	for _, p := range spec.Paths {
		v.literal(p.Path, "path "+p.Path)
	}
	for _, op := range spec.Operations {
		where := fmt.Sprintf("operation %q", op.ID)
		v.noControl(op.ID, "operationId in "+where)
		v.noControl(op.Summary, "summary in "+where)
		for _, p := range op.Parameters {
			v.literal(p.Name, "parameter name in "+where)
			if p.In == model.LocationCookie && !isScalarCookie(spec, p.Schema) {
				v.fail("cookie parameter %q in %s: only strings, numbers, booleans and string enums can be cookie parameters", p.Name, where)
			}
			v.schema(p.Schema, fmt.Sprintf("parameter %q in %s", p.Name, where))
		}
		if op.RequestBody != nil {
			v.content(op.RequestBody.Content, "request body in "+where)
		}
		v.responses(op.Responses, where)
		for _, cb := range op.Callbacks {
			for _, cbOp := range cb.Operations {
				cbWhere := fmt.Sprintf("callback %q in %s", cb.Name, where)
				if cbOp.RequestBody != nil {
					v.content(cbOp.RequestBody.Content, "request body in "+cbWhere)
				}
				v.responses(cbOp.Responses, cbWhere)
			}
		}
	}
	return errors.Join(v.errs...)
}

type validator struct {
	errs []error
	seen map[*model.Schema]bool
}

func (v *validator) fail(format string, args ...any) {
	v.errs = append(v.errs, fmt.Errorf(format, args...))
}

func (v *validator) noControl(s, what string) {
	for _, r := range s {
		if unicode.IsControl(r) {
			v.fail("%s: %q contains the control character %U", what, s, r)
			return
		}
	}
}

func (v *validator) literal(s, what string) {
	if i := strings.IndexAny(s, "\"`\\"); i >= 0 {
		v.fail("%s: %q contains %q, which cannot appear in generated Go source", what, s, s[i])
		return
	}
	v.noControl(s, what)
}

func (v *validator) identifier(s, what string) {
	if !token.IsIdentifier(s) {
		v.fail("%s: %q is not a Go identifier", what, s)
	}
}

func (v *validator) responses(responses []model.Response, where string) {
	for _, r := range responses {
		rWhere := fmt.Sprintf("response %q in %s", r.StatusCode, where)
		if !statusCodePattern.MatchString(r.StatusCode) {
			v.fail("status code in %s: %q is not an HTTP status code, a range such as 2XX, or default", where, r.StatusCode)
		}
		v.content(r.Content, rWhere)
		for _, h := range r.Headers {
			v.literal(h.Name, "header name in "+rWhere)
			v.schema(h.Schema, fmt.Sprintf("header %q in %s", h.Name, rWhere))
		}
	}
}

func (v *validator) content(content []model.MediaTypeContent, where string) {
	for _, c := range content {
		v.literal(c.MediaType, "media type in "+where)
		v.schema(c.Schema, fmt.Sprintf("%s in %s", c.MediaType, where))
	}
}

func (v *validator) schema(s *model.Schema, where string) {
	if s == nil || v.seen[s] {
		return
	}
	v.seen[s] = true

	for _, p := range s.Properties {
		v.literal(p.Name, "property name in "+where)
		v.schema(p.Schema, fmt.Sprintf("property %q in %s", p.Name, where))
	}
	if s.Type == model.TypeInteger || s.Type == model.TypeNumber {
		for _, e := range s.Enum {
			if !numberPattern.MatchString(fmt.Sprint(e)) {
				v.fail("enum value in %s: %q is not a %s", where, fmt.Sprint(e), s.Type)
			}
		}
	}
	if d := s.Discriminator; d != nil {
		v.literal(d.PropertyName, "discriminator property name in "+where)
		for key, ref := range d.Mapping {
			v.literal(key, "discriminator mapping key in "+where)
			v.literal(ref, "discriminator mapping in "+where)
		}
	}
	if ext := s.Extensions; ext != nil {
		v.extensions(ext, where)
	}

	v.schema(s.Items, "items of "+where)
	v.schema(s.AdditionalProperties, "additionalProperties of "+where)
	for _, group := range [][]*model.Schema{s.AllOf, s.OneOf, s.AnyOf} {
		for _, member := range group {
			v.schema(member, where)
		}
	}
}

func (v *validator) extensions(ext *model.SchemaExtensions, where string) {
	if ext.GoName != "" {
		v.identifier(ext.GoName, "x-oink-go-name in "+where)
	}
	if ext.GoType != "" && !isGoType(ext.GoType) {
		v.fail("x-oink-go-type %q in %s is not a Go type", ext.GoType, where)
	}
	if imp := ext.GoTypeImport; imp != nil {
		v.literal(imp.Path, "x-oink-go-type-import path in "+where)
		if imp.Alias != "" {
			v.identifier(imp.Alias, "x-oink-go-type-import alias in "+where)
		}
	}
	for key, value := range ext.ExtraTags {
		if !tagKeyPattern.MatchString(key) {
			v.fail("x-oink-extra-tags in %s: %q is not a struct tag key", where, key)
		}
		v.literal(value, fmt.Sprintf("x-oink-extra-tags %q in %s", key, where))
	}
}

// isGoType reports whether s parses as a Go type expression that runs no code.
func isGoType(s string) bool {
	expr, err := parser.ParseExpr(s)
	if err != nil {
		return false
	}
	ok := true
	ast.Inspect(expr, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr, *ast.FuncLit, *ast.CompositeLit, *ast.BinaryExpr, *ast.UnaryExpr:
			ok = false
		case *ast.BasicLit:
			ok = n.Kind == token.INT
		}
		return ok
	})
	return ok
}

func isScalarCookie(spec *model.Spec, s *model.Schema) bool {
	if s != nil && s.Type == "" && s.Ref != "" {
		s = spec.SchemaByRef(s.Ref)
	}
	if s == nil || len(s.AllOf)+len(s.OneOf)+len(s.AnyOf) > 0 || (s.Extensions != nil && s.Extensions.GoType != "") {
		return false
	}
	switch s.Type {
	case model.TypeString:
		return s.Format != "byte" && s.Format != "binary"
	case model.TypeInteger, model.TypeNumber, model.TypeBoolean:
		return len(s.Enum) == 0
	}
	return false
}
