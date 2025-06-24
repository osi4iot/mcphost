package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MCPServerConfig represents configuration for an MCP server
type MCPServerConfig struct {
	Type          string            `json:"type"`
	Command       []string          `json:"command,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
	URL           string            `json:"url,omitempty"`
	AllowedTools  []string          `json:"allowedTools,omitempty"`
	ExcludedTools []string          `json:"excludedTools,omitempty"`
	
	// Legacy fields for backward compatibility
	Transport     string         `json:"transport,omitempty"`
	Args          []string       `json:"args,omitempty"`
	Env           map[string]any `json:"env,omitempty"`
	Headers       []string       `json:"headers,omitempty"`
}

// UnmarshalJSON handles both new and legacy config formats
func (s *MCPServerConfig) UnmarshalJSON(data []byte) error {
	// First try to unmarshal as the new format
	type newFormat struct {
		Type          string            `json:"type"`
		Command       []string          `json:"command,omitempty"`
		Environment   map[string]string `json:"environment,omitempty"`
		URL           string            `json:"url,omitempty"`
		AllowedTools  []string          `json:"allowedTools,omitempty"`
		ExcludedTools []string          `json:"excludedTools,omitempty"`
	}
	
	// Also try legacy format
	type legacyFormat struct {
		Transport     string         `json:"transport,omitempty"`
		Command       string         `json:"command,omitempty"`
		Args          []string       `json:"args,omitempty"`
		Env           map[string]any `json:"env,omitempty"`
		URL           string         `json:"url,omitempty"`
		Headers       []string       `json:"headers,omitempty"`
		AllowedTools  []string       `json:"allowedTools,omitempty"`
		ExcludedTools []string       `json:"excludedTools,omitempty"`
	}
	
	// Try new format first
	var newConfig newFormat
	if err := json.Unmarshal(data, &newConfig); err == nil && newConfig.Type != "" {
		s.Type = newConfig.Type
		s.Command = newConfig.Command
		s.Environment = newConfig.Environment
		s.URL = newConfig.URL
		s.AllowedTools = newConfig.AllowedTools
		s.ExcludedTools = newConfig.ExcludedTools
		return nil
	}
	
	// Fall back to legacy format
	var legacyConfig legacyFormat
	if err := json.Unmarshal(data, &legacyConfig); err != nil {
		return err
	}
	
	// Convert legacy format to new format
	s.Transport = legacyConfig.Transport
	if legacyConfig.Command != "" {
		s.Command = append([]string{legacyConfig.Command}, legacyConfig.Args...)
	}
	s.Args = legacyConfig.Args
	s.Env = legacyConfig.Env
	s.URL = legacyConfig.URL
	s.Headers = legacyConfig.Headers
	s.AllowedTools = legacyConfig.AllowedTools
	s.ExcludedTools = legacyConfig.ExcludedTools
	
	return nil
}

// Config represents the application configuration
type Config struct {
	MCPServers     map[string]MCPServerConfig `json:"mcpServers" yaml:"mcpServers"`
	Model          string                     `json:"model,omitempty" yaml:"model,omitempty"`
	MaxSteps       int                        `json:"max-steps,omitempty" yaml:"max-steps,omitempty"`
	Debug          bool                       `json:"debug,omitempty" yaml:"debug,omitempty"`
	SystemPrompt   string                     `json:"system-prompt,omitempty" yaml:"system-prompt,omitempty"`
	ProviderAPIKey string                     `json:"provider-api-key,omitempty" yaml:"provider-api-key,omitempty"`
	ProviderURL    string                     `json:"provider-url,omitempty" yaml:"provider-url,omitempty"`
	Prompt         string                     `json:"prompt,omitempty" yaml:"prompt,omitempty"`
	NoExit         bool                       `json:"no-exit,omitempty" yaml:"no-exit,omitempty"`

	// Model generation parameters
	MaxTokens     int      `json:"max-tokens,omitempty" yaml:"max-tokens,omitempty"`
	Temperature   *float32 `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	TopP          *float32 `json:"top-p,omitempty" yaml:"top-p,omitempty"`
	TopK          *int32   `json:"top-k,omitempty" yaml:"top-k,omitempty"`
	StopSequences []string `json:"stop-sequences,omitempty" yaml:"stop-sequences,omitempty"`
}

// GetTransportType returns the transport type for the server config
func (s *MCPServerConfig) GetTransportType() string {
	// New simplified format
	if s.Type != "" {
		switch s.Type {
		case "local":
			return "stdio"
		case "remote":
			return "streamable"
		default:
			return s.Type
		}
	}
	
	// Legacy format support
	if s.Transport != "" {
		return s.Transport
	}
	// Backward compatibility: infer transport type
	if len(s.Command) > 0 {
		return "stdio"
	}
	if s.URL != "" {
		return "sse"
	}
	return "stdio" // default
}

// Validate validates the configuration
func (c *Config) Validate() error {
	for serverName, serverConfig := range c.MCPServers {
		if len(serverConfig.AllowedTools) > 0 && len(serverConfig.ExcludedTools) > 0 {
			return fmt.Errorf("server %s: allowedTools and excludedTools are mutually exclusive", serverName)
		}

		transport := serverConfig.GetTransportType()
		switch transport {
		case "stdio":
			// Check both new and legacy command formats
			if len(serverConfig.Command) == 0 && serverConfig.Transport == "" {
				return fmt.Errorf("server %s: command is required for stdio transport", serverName)
			}
		case "sse", "streamable":
			if serverConfig.URL == "" {
				return fmt.Errorf("server %s: url is required for %s transport", serverName, transport)
			}
		default:
			return fmt.Errorf("server %s: unsupported transport type '%s'. Supported types: stdio, sse, streamable", serverName, transport)
		}
	}
	return nil
}

// LoadSystemPrompt loads system prompt from file or returns the string directly
func LoadSystemPrompt(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	// Check if input is a file that exists
	if _, err := os.Stat(input); err == nil {
		// Read the entire file as plain text
		content, err := os.ReadFile(input)
		if err != nil {
			return "", fmt.Errorf("error reading system prompt file: %v", err)
		}
		return strings.TrimSpace(string(content)), nil
	}

	// Treat as direct string
	return input, nil
}

// EnsureConfigExists checks if a config file exists and creates a default one if not
func EnsureConfigExists() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %v", err)
	}

	// Check for existing config files (new format first, then legacy)
	configNames := []string{".mcphost", ".mcp"}
	configTypes := []string{"yml", "yaml", "json"}

	for _, configName := range configNames {
		for _, configType := range configTypes {
			configPath := filepath.Join(homeDir, configName+"."+configType)
			if _, err := os.Stat(configPath); err == nil {
				// Config file exists, no need to create
				return nil
			}
		}
	}

	// No config file found, create default
	return createDefaultConfig(homeDir)
}

// createDefaultConfig creates a default .mcphost.yml file in the user's home directory
func createDefaultConfig(homeDir string) error {
	configPath := filepath.Join(homeDir, ".mcphost.yml")

	// Create the file
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("error creating config file: %v", err)
	}
	defer file.Close()

	// Write a clean YAML template
	content := `# MCPHost Configuration File
# All command-line flags can be configured here

# MCP Servers configuration
# Add your MCP servers here
# Examples for different server types:
# mcpServers:
#   # Local servers - run commands locally
#   filesystem:
#     type: "local"
#     command: ["npx", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/files"]
#     environment:
#       MY_ENV_VAR: "my_env_var_value"
#   sqlite:
#     type: "local"
#     command: ["uvx", "mcp-server-sqlite", "--db-path", "/tmp/example.db"]
#   
#   # Remote servers - connect via StreamableHTTP
#   websearch:
#     type: "remote"
#     url: "https://api.example.com/mcp"
#   
#   # Legacy format still supported for backward compatibility:
#   # legacy-server:
#   #   command: npx
#   #   args: ["@modelcontextprotocol/server-filesystem", "/path"]
#   #   env:
#   #     MY_VAR: "value"

mcpServers:

# Application settings (all optional)
# model: "anthropic:claude-sonnet-4-20250514"  # Default model to use
# max-steps: 20                                # Maximum agent steps (0 for unlimited)
# debug: false                                 # Enable debug logging
# system-prompt: "/path/to/system-prompt.txt" # System prompt text file

# Model generation parameters (all optional)
# max-tokens: 4096                             # Maximum tokens in response
# temperature: 0.7                             # Randomness (0.0-1.0)
# top-p: 0.95                                  # Nucleus sampling (0.0-1.0)
# top-k: 40                                    # Top K sampling
# stop-sequences: ["Human:", "Assistant:"]     # Custom stop sequences

# API Configuration (can also use environment variables)
# provider-api-key: "your-api-key"         # API key for OpenAI, Anthropic, or Google
# provider-url: "https://api.openai.com/v1" # Base URL for OpenAI, Anthropic, or Ollama
`

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("error writing config content: %v", err)
	}

	return nil
}
