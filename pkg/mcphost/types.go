package mcphost

import (
	"github.com/cloudwego/eino/schema"
	"github.com/osi4iot/mcphost/internal/config"
)


type MCPServerConfig = config.MCPServerConfig

type NATSConfig struct {
	ServersURL string `json:"servers_url"`
	SubjectIn  string `json:"subject_in"`
	SubjectOut string `json:"subject_out"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

type HostConfig struct {
	NATSConfig     *NATSConfig                `json:"nats_config"`
	MCPServers     map[string]MCPServerConfig `json:"mcpServers" yaml:"mcpServers"`
	Model          string                     `json:"model,omitempty" yaml:"model,omitempty"`
	MaxSteps       int                        `json:"max-steps,omitempty" yaml:"max-steps,omitempty"`
	Debug          bool                       `json:"debug,omitempty" yaml:"debug,omitempty"`
	SystemPrompt   string                     `json:"system-prompt,omitempty" yaml:"system-prompt,omitempty"`
	ProviderAPIKey string                     `json:"provider-api-key,omitempty" yaml:"provider-api-key,omitempty"`
	ProviderURL    string                     `json:"provider-url,omitempty" yaml:"provider-url,omitempty"`
	MaxTokens      int                        `json:"max-tokens,omitempty" yaml:"max-tokens,omitempty"`
	Temperature    *float32                   `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	TopP           *float32                   `json:"top-p,omitempty" yaml:"top-p,omitempty"`
	TopK           *int32                     `json:"top-k,omitempty" yaml:"top-k,omitempty"`
	SavedMessages  []*schema.Message          `json:"messages,omitempty" yaml:"messages,omitempty"`
}

type AgenticLoopConfig struct {
	// Mode configuration
	IsInteractive    bool   // true for interactive mode, false for non-interactive
	InitialPrompt    string // initial prompt for non-interactive mode
	ContinueAfterRun bool   // true to continue to interactive mode after initial run (--no-exit)

	// UI configuration
	Quiet bool // suppress all output except final response

	// Context data
	ServerNames    []string         // for slash commands
	ToolNames      []string         // for slash commands
	ModelName      string           // for display
	MCPConfig      *config.Config   // for continuing to interactive mode
}
