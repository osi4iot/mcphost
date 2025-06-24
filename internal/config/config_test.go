package config

import (
	"encoding/json"
	"testing"
)

func TestMCPServerConfig_NewFormat(t *testing.T) {
	// Test new simplified format
	jsonData := `{
		"type": "local",
		"command": ["bun", "x", "my-mcp-command"],
		"environment": {
			"MY_ENV_VAR": "my_env_var_value"
		}
	}`

	var config MCPServerConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal new format: %v", err)
	}

	if config.Type != "local" {
		t.Errorf("Expected type 'local', got '%s'", config.Type)
	}

	if len(config.Command) != 3 {
		t.Errorf("Expected 3 command parts, got %d", len(config.Command))
	}

	if config.Command[0] != "bun" || config.Command[1] != "x" || config.Command[2] != "my-mcp-command" {
		t.Errorf("Command parts incorrect: %v", config.Command)
	}

	if config.Environment["MY_ENV_VAR"] != "my_env_var_value" {
		t.Errorf("Environment variable not set correctly")
	}

	// Test transport type detection
	transportType := config.GetTransportType()
	if transportType != "stdio" {
		t.Errorf("Expected transport type 'stdio', got '%s'", transportType)
	}
}

func TestMCPServerConfig_RemoteFormat(t *testing.T) {
	// Test remote format
	jsonData := `{
		"type": "remote",
		"url": "https://my-mcp-server.com"
	}`

	var config MCPServerConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal remote format: %v", err)
	}

	if config.Type != "remote" {
		t.Errorf("Expected type 'remote', got '%s'", config.Type)
	}

	if config.URL != "https://my-mcp-server.com" {
		t.Errorf("Expected URL 'https://my-mcp-server.com', got '%s'", config.URL)
	}

	// Test transport type detection
	transportType := config.GetTransportType()
	if transportType != "streamable" {
		t.Errorf("Expected transport type 'streamable', got '%s'", transportType)
	}
}

func TestMCPServerConfig_LegacyFormat(t *testing.T) {
	// Test legacy format still works
	jsonData := `{
		"command": "npx",
		"args": ["@modelcontextprotocol/server-filesystem", "/path"],
		"env": {
			"MY_VAR": "value"
		}
	}`

	var config MCPServerConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal legacy format: %v", err)
	}

	if len(config.Command) != 3 {
		t.Errorf("Expected 3 command parts, got %d", len(config.Command))
	}

	if config.Command[0] != "npx" || config.Command[1] != "@modelcontextprotocol/server-filesystem" || config.Command[2] != "/path" {
		t.Errorf("Command parts incorrect: %v", config.Command)
	}

	if config.Env["MY_VAR"] != "value" {
		t.Errorf("Legacy environment variable not set correctly")
	}

	// Test transport type detection
	transportType := config.GetTransportType()
	if transportType != "stdio" {
		t.Errorf("Expected transport type 'stdio', got '%s'", transportType)
	}
}

func TestMCPServerConfig_BuiltinFormat(t *testing.T) {
	// Test builtin format with allowed_directories
	jsonData := `{
		"type": "builtin",
		"name": "fs",
		"options": {
			"allowed_directories": ["/tmp", "/home/user"]
		}
	}`

	var config MCPServerConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal builtin format: %v", err)
	}

	if config.Type != "builtin" {
		t.Errorf("Expected type 'builtin', got '%s'", config.Type)
	}

	if config.Name != "fs" {
		t.Errorf("Expected name 'fs', got '%s'", config.Name)
	}

	if config.Options == nil {
		t.Errorf("Expected options to be set")
	}

	// Test transport type detection
	transportType := config.GetTransportType()
	if transportType != "inprocess" {
		t.Errorf("Expected transport type 'inprocess', got '%s'", transportType)
	}
}

func TestMCPServerConfig_BuiltinFormatMinimal(t *testing.T) {
	// Test builtin format without allowed_directories (should default to cwd)
	jsonData := `{
		"type": "builtin",
		"name": "fs"
	}`

	var config MCPServerConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal minimal builtin format: %v", err)
	}

	if config.Type != "builtin" {
		t.Errorf("Expected type 'builtin', got '%s'", config.Type)
	}

	if config.Name != "fs" {
		t.Errorf("Expected name 'fs', got '%s'", config.Name)
	}

	// Test transport type detection
	transportType := config.GetTransportType()
	if transportType != "inprocess" {
		t.Errorf("Expected transport type 'inprocess', got '%s'", transportType)
	}
}

func TestConfig_Validate(t *testing.T) {
	config := &Config{
		MCPServers: map[string]MCPServerConfig{
			"local-server": {
				Type:    "local",
				Command: []string{"echo", "hello"},
			},
			"remote-server": {
				Type: "remote",
				URL:  "https://example.com",
			},
			"builtin-server": {
				Type: "builtin",
				Name: "fs",
				Options: map[string]any{
					"allowed_directories": []string{"/tmp"},
				},
			},
		},
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Validation failed: %v", err)
	}
}
