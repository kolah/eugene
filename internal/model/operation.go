package model

type Operation struct {
	ID          string
	Method      Method
	Path        string
	Summary     string
	Description string
	Tags        []string
	Parameters  []Parameter
	RequestBody *RequestBody
	Responses   []Response
	Deprecated  bool
	Security    []SecurityRequirement
	Streaming   *StreamingConfig // SSE/streaming response
	Callbacks   []Callback
}

type Callback struct {
	Name       string // e.g., "orderProcessed"
	Expression string // e.g., "{$request.body#/callbackUrl}"
	Operations []CallbackOperation
}

type CallbackOperation struct {
	Method      Method
	RequestBody *RequestBody
	Responses   []Response
}

type StreamingConfig struct {
	MediaType   string // e.g., "text/event-stream"
	EventType   string // Schema type for events
	EventSchema *Schema
}

type Method string

const (
	MethodGet     Method = "GET"
	MethodPost    Method = "POST"
	MethodPut     Method = "PUT"
	MethodDelete  Method = "DELETE"
	MethodPatch   Method = "PATCH"
	MethodHead    Method = "HEAD"
	MethodOptions Method = "OPTIONS"
	MethodTrace   Method = "TRACE"
	MethodQuery   Method = "QUERY" // OpenAPI 3.2
)

type ParameterLocation string

const (
	LocationPath        ParameterLocation = "path"
	LocationQuery       ParameterLocation = "query"
	LocationHeader      ParameterLocation = "header"
	LocationCookie      ParameterLocation = "cookie"
	LocationQueryString ParameterLocation = "querystring" // OpenAPI 3.2
)

type Parameter struct {
	Name        string
	In          ParameterLocation
	Description string
	Required    bool
	Deprecated  bool
	Schema      *Schema
}

type RequestBody struct {
	Description string
	Required    bool
	Content     []MediaTypeContent
}

type MediaTypeContent struct {
	MediaType string
	Schema    *Schema
}

type Response struct {
	StatusCode  string
	Description string
	Content     []MediaTypeContent
	Headers     []Header
}

type Header struct {
	Name        string
	Description string
	Required    bool
	Schema      *Schema
}

type SecurityRequirement struct {
	Name   string
	Scopes []string
}
