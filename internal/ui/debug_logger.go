package ui

import (
	"fmt"
	"strings"
	"time"
)

// CLIDebugLogger implements the tools.DebugLogger interface using CLI rendering
type CLIDebugLogger struct {
	cli *CLI
}

// NewCLIDebugLogger creates a new CLI debug logger
func NewCLIDebugLogger(cli *CLI) *CLIDebugLogger {
	return &CLIDebugLogger{cli: cli}
}

// LogDebug logs a debug message using the CLI's debug message renderer
func (l *CLIDebugLogger) LogDebug(message string) {
	if l.cli == nil || !l.cli.debug {
		return
	}

	// Format the message to include all the debug info in a structured way
	var formattedMessage string

	// Check if this is a multi-line debug output (like connection info)
	if strings.Contains(message, "[DEBUG]") || strings.Contains(message, "[POOL]") {
		// Extract the tag and content
		if strings.HasPrefix(message, "[DEBUG]") {
			content := strings.TrimPrefix(message, "[DEBUG]")
			content = strings.TrimSpace(content)
			formattedMessage = fmt.Sprintf("🔍 DEBUG: %s", content)
		} else if strings.HasPrefix(message, "[POOL]") {
			content := strings.TrimPrefix(message, "[POOL]")
			content = strings.TrimSpace(content)

			// Add appropriate emoji based on the message content
			if strings.Contains(content, "Creating new connection") {
				formattedMessage = fmt.Sprintf("🆕 POOL: %s", content)
			} else if strings.Contains(content, "Created connection") || strings.Contains(content, "Initialized") {
				formattedMessage = fmt.Sprintf("✅ POOL: %s", content)
			} else if strings.Contains(content, "Reusing") {
				formattedMessage = fmt.Sprintf("🔄 POOL: %s", content)
			} else if strings.Contains(content, "unhealthy") || strings.Contains(content, "failed") {
				formattedMessage = fmt.Sprintf("❌ POOL: %s", content)
			} else if strings.Contains(content, "closed") {
				formattedMessage = fmt.Sprintf("🛑 POOL: %s", content)
			} else if strings.Contains(content, "Failed to close") {
				formattedMessage = fmt.Sprintf("⚠️ POOL: %s", content)
			} else {
				formattedMessage = fmt.Sprintf("🔍 POOL: %s", content)
			}
		} else {
			formattedMessage = message
		}
	} else {
		formattedMessage = message
	}

	// Use the CLI's debug message rendering
	var msg UIMessage
	if l.cli.compactMode {
		msg = l.cli.compactRenderer.RenderDebugMessage(formattedMessage, time.Now())
	} else {
		msg = l.cli.messageRenderer.RenderDebugMessage(formattedMessage, time.Now())
	}
	l.cli.messageContainer.AddMessage(msg)
	l.cli.displayContainer()
}

// IsDebugEnabled returns whether debug logging is enabled
func (l *CLIDebugLogger) IsDebugEnabled() bool {
	return l.cli != nil && l.cli.debug
}
