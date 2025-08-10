package mcphost

import (
	"github.com/cloudwego/eino/schema"
	"github.com/nats-io/nats.go"
	"github.com/osi4iot/mcphost/internal/config"
)

type MCPServerConfig = config.MCPServerConfig

type HostConfig struct {
	NatsClient     *nats.Conn
	MCPServers     map[string]MCPServerConfig
	Model          string
	MaxSteps       int
	Debug          bool
	SystemPrompt   string
	ProviderAPIKey string
	ProviderURL    string
	MaxTokens      int
	Temperature    *float32
	TopP           *float32
	TopK           *int32
	SavedMessages  []*schema.Message
	InputChan      chan ChatMessage
	OutputChan     chan string
}

type AgenticLoopConfig struct {
	// Mode configuration
	IsInteractive    bool   // true for interactive mode, false for non-interactive
	InitialPrompt    string // initial prompt for non-interactive mode
	ContinueAfterRun bool   // true to continue to interactive mode after initial run (--no-exit)

	// UI configuration
	Quiet bool // suppress all output except final response

	// Context data
	ServerNames []string       // for slash commands
	ToolNames   []string       // for slash commands
	ModelName   string         // for display
	MCPConfig   *config.Config // for continuing to interactive mode
}

type ChatMessage struct {
	UserName string `json:"user_name"`
	Prompt   string `json:"prompt"`
}
