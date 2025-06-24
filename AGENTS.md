# MCPHost Development Context

## Build/Test Commands
- **Build**: `go build -o output/mcphost` or use `./contribute/build.sh`
- **Test**: `go test ./...` (run all tests)
- **Test single package**: `go test ./pkg/llm/anthropic`
- **Lint**: `go vet ./...` (built-in Go linter)
- **Format**: `go fmt ./...`
- **Dependencies**: `go mod tidy`

## Code Style Guidelines
- **Package structure**: `pkg/` for reusable packages, `cmd/` for CLI commands
- **Imports**: Standard library first, then third-party, then local packages with blank lines between groups
- **Naming**: Use camelCase for unexported, PascalCase for exported; descriptive names (e.g., `CreateMessage`, `mcpClients`)
- **Interfaces**: Keep small and focused (e.g., `llm.Provider`, `llm.Message`)
- **Error handling**: Always check errors, wrap with context using `fmt.Errorf("context: %v", err)`
- **Logging**: Use `github.com/charmbracelet/log` with structured logging: `log.Info("message", "key", value)`
- **Types**: Prefer `any` over `interface{}` (modernize hint from linter)
- **JSON tags**: Use snake_case for JSON fields, include omitempty where appropriate
- **Context**: Always pass `context.Context` as first parameter for operations that may block

## Architecture Notes
- Multi-provider LLM support (Anthropic, OpenAI, Ollama, Google)
- MCP (Model Context Protocol) client-server architecture for tool integration
- Provider pattern for LLM abstraction with common `llm.Provider` interface
- History management with message pruning based on configurable window size
- Tool calling support across all providers with unified `llm.Tool` interface

## MCP Configuration Schema
MCPHost supports a simplified configuration schema with three server types:

### New Simplified Format
- **Local servers** (`"type": "local"`): Run commands locally via stdio transport
  - `command`: Array of command and arguments (e.g., `["npx", "server", "args"]`)
  - `environment`: Key-value map of environment variables
- **Remote servers** (`"type": "remote"`): Connect via StreamableHTTP transport
  - `url`: Server endpoint URL
  - Automatically uses StreamableHTTP for optimal performance
- **Builtin servers** (`"type": "builtin"`): Run in-process for optimal performance
  - `name`: Internal name of the builtin server (e.g., `"fs"`)
  - `options`: Configuration options specific to the builtin server

### Legacy Format Support
- Maintains full backward compatibility with existing configurations
- Automatic detection and conversion of legacy formats
- Custom `UnmarshalJSON` method handles format migration seamlessly

### Transport Mapping
- `"local"` type → `stdio` transport (launches local processes)
- `"remote"` type → `streamable` transport (StreamableHTTP protocol)
- `"builtin"` type → `inprocess` transport (in-process execution)
- Legacy `transport` field still supported for backward compatibility

### Configuration Files
- Primary: `~/.mcphost.yml` or `~/.mcphost.json`
- Legacy: `~/.mcp.yml` or `~/.mcp.json`
- Custom location via `--config` flag

## Available Builtin Servers
MCPHost includes several builtin MCP servers for common functionality:

### Filesystem Server (`fs`)
- **Location**: `internal/builtin/registry.go` (registration), uses external `mcp-filesystem-server`
- **Purpose**: Secure filesystem access with configurable allowed directories
- **Options**: `allowed_directories` array (defaults to current working directory)

### Bash Server (`bash`)
- **Location**: `internal/builtin/bash.go`
- **Purpose**: Execute bash commands with security restrictions and timeout controls
- **Features**: Banned commands list, configurable timeouts, output size limits
- **Security**: Blocks network commands (curl, wget, etc.), 30KB output limit, 2-10 minute timeouts

### Todo Server (`todo`)
- **Location**: `internal/builtin/todo.go`
- **Purpose**: Manage ephemeral todo lists for task tracking during sessions
- **Features**: In-memory storage, thread-safe operations, JSON-based API
- **Tools**: `todowrite` (create/update todos), `todoread` (read current todos)

### Fetch Server (`fetch`)
- **Location**: `internal/builtin/fetch.go`
- **Purpose**: Fetch web content and convert to text, markdown, or HTML formats
- **Features**: HTTP/HTTPS support, HTML parsing, markdown conversion, size limits
- **Security**: 5MB response limit, configurable timeouts, localhost-aware HTTPS upgrade