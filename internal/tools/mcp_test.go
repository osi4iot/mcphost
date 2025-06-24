package tools

import (
	"context"
	"testing"
	"time"

	"github.com/mark3labs/mcphost/internal/config"
)

func TestMCPToolManager_LoadTools_WithTimeout(t *testing.T) {
	manager := NewMCPToolManager()

	// Create a config with a non-existent command that should fail
	cfg := &config.Config{
		MCPServers: map[string]config.MCPServerConfig{
			"test-server": {
				Command: "non-existent-command",
				Args:    []string{"arg1", "arg2"},
			},
		},
	}

	// Create a context with a reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// This should not hang indefinitely and should return an error
	start := time.Now()
	err := manager.LoadTools(ctx, cfg)
	duration := time.Since(start)

	// The operation should complete within our timeout
	if duration > 14*time.Second {
		t.Errorf("LoadTools took too long: %v, expected to complete within 14 seconds", duration)
	}

	// We expect an error since the command doesn't exist, but it shouldn't be a timeout
	if err == nil {
		t.Error("Expected an error for non-existent command, but got nil")
	}

	t.Logf("LoadTools completed in %v with error: %v", duration, err)
}

func TestMCPToolManager_LoadTools_GracefulFailure(t *testing.T) {
	manager := NewMCPToolManager()

	// Create a config with multiple servers, some good and some bad
	cfg := &config.Config{
		MCPServers: map[string]config.MCPServerConfig{
			"bad-server-1": {
				Command: "non-existent-command-1",
				Args:    []string{"arg1"},
			},
			"bad-server-2": {
				Command: "non-existent-command-2",
				Args:    []string{"arg2"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should fail gracefully and return an error since all servers failed
	err := manager.LoadTools(ctx, cfg)

	// We expect an error since all servers failed
	if err == nil {
		t.Error("Expected an error when all servers fail, but got nil")
	}

	// The error should mention that all servers failed
	if err != nil && !contains(err.Error(), "all MCP servers failed") {
		t.Errorf("Expected error message to mention all servers failed, got: %v", err)
	}

	t.Logf("LoadTools failed gracefully with error: %v", err)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
