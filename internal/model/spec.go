package model

type Spec struct {
	Info       Info
	Servers    []Server
	Tags       []Tag
	Paths      []Path
	Operations []Operation
	Schemas    []Schema
	Security   []SecurityScheme
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
