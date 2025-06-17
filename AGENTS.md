# Agent Development Guide

## Build/Test Commands
- **Build**: `go build -o mcphost .` or `go install`
- **Run**: `go run main.go` or `./mcphost`
- **Test**: `go test ./...` (run all tests)
- **Test single package**: `go test ./internal/config`
- **Lint**: `go vet ./...` and `gofmt -s -w .`
- **Dependencies**: `go mod tidy` and `go mod download`

## Code Style Guidelines
- **Imports**: Standard library first, then third-party, then local packages (separated by blank lines)
- **Naming**: Use camelCase for variables/functions, PascalCase for exported types
- **Error handling**: Always check errors, wrap with context using `fmt.Errorf("context: %v", err)`
- **Comments**: Use `//` for single line, document exported functions/types
- **Structs**: Use struct tags for JSON/YAML serialization (`json:"field" yaml:"field"`)
- **Interfaces**: Keep interfaces small and focused (e.g., `tool.BaseTool`, `model.ToolCallingChatModel`)

## Architecture
- **cmd/**: CLI commands and flag handling using Cobra
- **internal/**: Private application code (agent, config, models, tools, ui)
- **main.go**: Entry point, delegates to cmd package
- **go.mod**: Go 1.23+ required, uses Eino framework for LLM integration

## Key Patterns
- Use context.Context for cancellation and timeouts
- Implement proper resource cleanup with defer statements
- Use viper for configuration management (supports YAML/JSON)
- Follow MCP (Model Context Protocol) for tool integration
- Use structured logging and error wrapping