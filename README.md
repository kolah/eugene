# Eugene - OpenAPI INterface Kit - oink! 🐷

Eugene is a CLI tool for generating Go code from OpenAPI specifications (3.0, 3.1, 3.2). It generates type-safe clients, servers, and type definitions from your API specs.

This project is heavily inspired by [oapi-codegen/oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) and is an attempt to bring newest OpenAPI specifications while offering similar code generation experience.

Underneath, the tool is built upon [pb33f/libopenapi](https://github.com/pb33f/libopenapi) that made creating this tool possible.

## Installation

```bash
go install github.com/kolah/eugene/cmd/eugene@latest
```

or use it as a tool dependency:

```bash
go get -tool github.com/kolah/eugene/cmd/eugene@latest
```

## Quick Start

1. Create a configuration file `eugene.yaml`:

```yaml
spec: ./api/openapi.yaml

go:
  package: api
  output-dir: ./gen
  targets:
    - types
    - server
    - client
```

2. Run the generator:

```bash
eugene generate go
```

Or generate specific targets:

```bash
eugene generate go types
eugene generate go server
eugene generate go all
```

## CLI Usage

```
eugene generate go [target] [flags]

Targets:
  types          Generate Go type definitions
  server         Generate Go server code
  strict-server  Generate Go strict server with typed responses
  client         Generate Go HTTP client
  spec           Generate embedded OpenAPI spec
  all            Generate all targets

Common Flags:
  -c, --config string              Config file (default: eugene.yaml)
  -s, --spec string                OpenAPI spec path
      --templates string           Custom templates directory
      --exclude-schemas strings    Schemas to exclude
      --include-tags strings       Tags to include (exclusive)
      --exclude-tags strings       Tags to exclude
      --dry-run                    Print output without writing files

Go Flags:
  -o, --output-dir string          Output directory
  -p, --package string             Go package name
  -f, --server-framework string    Server framework: echo, chi, stdlib
      --enum-strategy string       Enum strategy: const, type, struct
      --uuid-package string        UUID type: string, google, gofrs
      --nullable-strategy string   Nullable strategy: pointer, nullable
      --enable-yaml-tags           Generate yaml tags alongside json tags
      --additional-initialisms     Custom initialisms for naming (e.g., GTIN,SKU)
```

## Configuration

Eugene supports configuration via YAML file, CLI flags, and environment variables. Loading order: defaults -> YAML file -> CLI flags.

### Full Configuration Example

```yaml
spec: api/openapi.yaml

templates:
  dir: ./custom-templates

exclude-schemas:
  - InternalType

include-tags:
  - public

go:
  package: api
  output-dir: ./gen
  server-framework: echo

  targets:
    - types
    - server
    - strict-server
    - client
    - spec

  types:
    enum-strategy: const      # const, type, or struct
    uuid-package: google      # string, google, or gofrs
    nullable-strategy: pointer # pointer or nullable

  output-options:
    enable-yaml-tags: true
    additional-initialisms:
      - GTIN
      - SKU

  import-mapping:
    "#/components/schemas/Error": github.com/myorg/api/common
```

## Generated Code

### Types (`types.go`)

Generates Go structs from OpenAPI schemas:

```go
type Pet struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
    Tag  string `json:"tag,omitempty"`
}
```

### Server (`server.go`)

Generates a server interface and registration function:

```go
type ServerInterface interface {
    ListPets(ctx echo.Context) error
    CreatePet(ctx echo.Context) error
    GetPet(ctx echo.Context, id int64) error
}

func RegisterHandlers(e *echo.Echo, si ServerInterface) {
    // ...
}
```

### Strict Server (`strict_types.go`, `strict_server.go`)

Type-safe server with parsed request/response objects:

```go
type StrictServerInterface interface {
    GetPet(ctx context.Context, request GetPetRequest) (GetPetResponse, error)
}

type GetPetRequest struct {
    ID int64
}

type GetPetResponse interface {
    VisitGetPetResponse(w http.ResponseWriter) error
}
```

### Client (`client.go`)

HTTP client with typed methods:

```go
type Client struct {
    baseURL    string
    httpClient *http.Client
}

func (c *Client) GetPet(ctx context.Context, id int64) (*Pet, error) {
    // ...
}
```

## Server Frameworks

Eugene supports three server frameworks:

| Framework | Flag | Import |
|-----------|------|--------|
| Echo | `echo` | `github.com/labstack/echo/v4` |
| Chi | `chi` | `github.com/go-chi/chi/v5` |
| stdlib | `stdlib` | `net/http` |

## OpenAPI Extensions

Eugene supports custom extensions for fine-grained control:

| Extension | Purpose | Example |
|-----------|---------|---------|
| `x-oink-go-type` | Override Go type | `x-oink-go-type: time.Duration` |
| `x-oink-go-type-import` | Import path | `x-oink-go-type-import: {path: "time"}` |
| `x-oink-go-name` | Override field/type name | `x-oink-go-name: CustomerID` |
| `x-oink-extra-tags` | Add struct tags | `x-oink-extra-tags: {validate: "required"}` |
| `x-oink-omitempty` | Force omitempty | `x-oink-omitempty: true` |
| `x-oink-omitzero` | Force omitzero | `x-oink-omitzero: true` |
| `x-oink-json-ignore` | Exclude from JSON | `x-oink-json-ignore: true` |

### Example

```yaml
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
          x-oink-go-type: uuid.UUID
          x-oink-go-type-import:
            path: github.com/google/uuid
        email:
          type: string
          x-oink-extra-tags:
            validate: required,email
            db: email
```

## Enum Strategies

### `const` (default)
```go
type PetStatus string

const (
    PetStatusAvailable PetStatus = "available"
    PetStatusPending   PetStatus = "pending"
)
```

### `type`
```go
type PetStatus string

const (
    PetStatusAvailable PetStatus = "available"
    PetStatusPending   PetStatus = "pending"
)
```

### `struct`
```go
type PetStatus struct {
    value string
}

func (e PetStatus) String() string { return e.value }
func (e PetStatus) IsValid() bool { ... }

var (
    PetStatusAvailable = PetStatus{value: "available"}
    PetStatusPending   = PetStatus{value: "pending"}
)
```

## Union Types (oneOf/anyOf)

Eugene generates union types with discriminator support:

```go
type PaymentSource struct {
    Type string          `json:"-"`
    Raw  json.RawMessage `json:"-"`
}

func (s *PaymentSource) AsCard() (*Card, error) { ... }
func (s *PaymentSource) AsBankAccount() (*BankAccount, error) { ... }
```

## SSE/Streaming Support

Both client and server support Server-Sent Events:

**Client:**
```go
stream, err := client.StreamEvents(ctx)
defer stream.Close()

for stream.Next() {
    event := stream.Current()
    var data MyEvent
    event.Decode(&data)
}
```

**Server:**
```go
func (s *Server) StreamEvents(w http.ResponseWriter, r *http.Request) {
    sse, _ := NewSSEWriter(w)
    sse.WriteEvent("message", myData)
}
```

## Custom Templates

Override built-in templates by providing a custom templates directory:

```yaml
templates:
  dir: ./my-templates
```

Templates use Go's `text/template` with custom functions:
- `pascalCase`, `camelCase`, `snakeCase` - naming conventions
- `goType` - OpenAPI schema to Go type
- `goComment` - format as Go comment
- `isRequired`, `isNullable` - schema helpers

## Project Structure

```
eugene/
├── cmd/main.go           # CLI entry point
├── internal/
│   ├── cli/              # Cobra commands
│   ├── config/           # Configuration
│   ├── loader/           # OpenAPI parsing (libopenapi)
│   ├── model/            # Internal representation
│   ├── codegen/          # Generation pipeline
│   ├── golang/           # Go-specific logic
│   ├── templates/        # Template engine
│   └── targets/          # Generation targets
├── templates/            # Embedded templates
└── testdata/             # Test fixtures
```

## Dependencies

| Purpose | Library |
|---------|---------|
| OpenAPI Parsing | `pb33f/libopenapi` |
| CLI | `spf13/cobra` |
| Config | `knadh/koanf` |
| Formatting | `golang.org/x/tools/imports` |

## Testing

Integration tests live in a separate `tests/` module to avoid polluting the main module with framework-specific dependencies (Echo, Chi).

```bash
cd tests
go test ./...
```

## License

MIT
