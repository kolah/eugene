package model

import "strings"

type Spec struct {
	Info       Info
	Servers    []Server
	Tags       []Tag
	Paths      []Path
	Operations []Operation
	Schemas    []Schema
	Security   []SecurityScheme
}

// SchemaByRef returns a schema by its $ref path (e.g., "#/components/schemas/User").
// Returns nil if the schema is not found.
func (s *Spec) SchemaByRef(ref string) *Schema {
	parts := strings.Split(ref, "/")
	if len(parts) == 0 {
		return nil
	}
	name := parts[len(parts)-1]
	for i := range s.Schemas {
		if s.Schemas[i].Name == name {
			return &s.Schemas[i]
		}
	}
	return nil
}

type Info struct {
	Title       string
	Description string
	Version     string
}

type Server struct {
	URL         string
	Description string
}

type Tag struct {
	Name        string
	Summary     string // OpenAPI 3.2
	Description string
	Parent      string // OpenAPI 3.2 hierarchical tags
	Kind        string // OpenAPI 3.2 tag classification
}

type Path struct {
	Path       string
	Operations []Operation
}
