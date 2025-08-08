package tools

// DebugLogger interface for debug logging
type DebugLogger interface {
	LogDebug(message string)
	IsDebugEnabled() bool
}

// SimpleDebugLogger is a simple implementation that prints to stdout
type SimpleDebugLogger struct {
	enabled bool
}

// NewSimpleDebugLogger creates a new simple debug logger
func NewSimpleDebugLogger(enabled bool) *SimpleDebugLogger {
	return &SimpleDebugLogger{enabled: enabled}
}

// LogDebug logs a debug message
func (l *SimpleDebugLogger) LogDebug(message string) {
	// Silent by default - messages will only appear when using CLI debug logger
	// This prevents duplicate or unstyled debug output during initialization
}

// IsDebugEnabled returns whether debug logging is enabled
func (l *SimpleDebugLogger) IsDebugEnabled() bool {
	return l.enabled
}
