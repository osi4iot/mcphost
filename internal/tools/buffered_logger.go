package tools

import (
	"sync"
)

// BufferedDebugLogger stores debug messages until they can be displayed
type BufferedDebugLogger struct {
	enabled  bool
	messages []string
	mu       sync.Mutex
}

// NewBufferedDebugLogger creates a new buffered debug logger
func NewBufferedDebugLogger(enabled bool) *BufferedDebugLogger {
	return &BufferedDebugLogger{
		enabled:  enabled,
		messages: make([]string, 0),
	}
}

// LogDebug stores a debug message
func (l *BufferedDebugLogger) LogDebug(message string) {
	if !l.enabled {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, message)
}

// IsDebugEnabled returns whether debug logging is enabled
func (l *BufferedDebugLogger) IsDebugEnabled() bool {
	return l.enabled
}

// GetMessages returns all buffered messages and clears the buffer
func (l *BufferedDebugLogger) GetMessages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	messages := make([]string, len(l.messages))
	copy(messages, l.messages)
	l.messages = l.messages[:0] // Clear the buffer
	return messages
}
