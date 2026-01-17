# Eugene Integration Tests

This is a separate Go module for integration tests. It's kept separate from the main module to avoid polluting it with framework-specific dependencies (Echo, Chi).

## Structure

```
tests/
├── go.mod              # Separate module with test dependencies
├── codegen_test.go     # Code generation pipeline tests
├── compile_test.go     # Compilation verification tests
├── server_test.go      # HTTP server behavior tests
├── sse_test.go         # Server-Sent Events tests
├── generated/          # Generated code for compilation tests
│   ├── echo/
│   ├── chi/
│   └── stdlib/
└── golden/             # Expected output files
```

## Running Tests

```bash
cd tests
go test ./...
```

To update golden files:

```bash
go test ./... -update
```

## Test Categories

### Compilation Tests (`compile_test.go`)

Verifies that generated code compiles successfully for all targets:
- Types with all enum strategies (const, type, struct)
- Server code for Echo, Chi, and stdlib
- Client code
- Strict server code for all frameworks

### Server Routing Tests (`server_test.go`)

Tests HTTP routing behavior:
- Path parameter extraction
- Query parameter handling
- Handler registration

### SSE Tests (`sse_test.go`)

Tests Server-Sent Events streaming:
- Stream parsing
- Event decoding
- Multiple events

### Custom Template Tests (`codegen_test.go`)

Verifies custom template override functionality.

## Dependencies

The test module depends on:
- `github.com/kolah/eugene` - main generator module
- `github.com/labstack/echo/v4` - Echo framework tests
- `github.com/go-chi/chi/v5` - Chi framework tests
- `github.com/stretchr/testify` - assertions
