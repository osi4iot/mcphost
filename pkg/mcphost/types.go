package mcphost

import (
	"time"

	"github.com/osi4iot/mcphost/internal/config"
)

type Config = config.Config
type MCPServerConfig = config.MCPServerConfig

type NATSConfig struct {
    URL            string        `json:"url"`
    SubjectPrefix  string        `json:"subject_prefix"`  // Prefijo para subjects (ej: "mcphost")
    Timeout        time.Duration `json:"timeout"`
    MaxReconnects  int           `json:"max_reconnects"`
    ReconnectWait  time.Duration `json:"reconnect_wait"`
}

// Opciones para ProcessPrompt
type PromptOptions struct {
    Message      string            `json:"message"`
    Model        string            `json:"model,omitempty"`
    MaxTokens    int               `json:"max_tokens,omitempty"`
    Temperature  *float32          `json:"temperature,omitempty"`
    SystemPrompt string            `json:"system_prompt,omitempty"`
    Context      map[string]any    `json:"context,omitempty"`
}

// Respuesta del prompt
type PromptResponse struct {
    Content     string                 `json:"content"`
    ToolCalls   []ToolCall            `json:"tool_calls,omitempty"`
    TokensUsed  int                   `json:"tokens_used,omitempty"`
    Model       string                `json:"model"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type ToolCall struct {
    Name      string                 `json:"name"`
    Arguments map[string]interface{} `json:"arguments"`
    Result    interface{}            `json:"result,omitempty"`
}