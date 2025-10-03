package ft_supabase

import (
	"fmt"
	"log"
	"sync"
)

// Logger manages logging output with thread-safe operations.
// Enabled controls whether logging is active (default: true).
// mu is a mutex for thread-safe logging operations.
type Logger struct {
	Enabled bool
	mu      sync.Mutex
}

// globalLogger is the package-level logger instance used by all logging functions.
var globalLogger = &Logger{
	Enabled: true,
}

// SetLoggingEnabled enables or disables logging globally.
// enabled is true to enable logging, false to disable.
func SetLoggingEnabled(enabled bool) {
	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()
	globalLogger.Enabled = enabled
}

// IsLoggingEnabled returns whether logging is currently enabled.
// Returns true if logging is enabled, false otherwise.
func IsLoggingEnabled() bool {
	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()
	return globalLogger.Enabled
}

// Log logs a message with context information.
// context is the function or method name providing context.
// message is the log message content.
func Log(context, message string) {
	var (
		logMessage string
	)

	// check if logging is enabled
	globalLogger.mu.Lock()
	if !globalLogger.Enabled {
		globalLogger.mu.Unlock()
		return
	}
	globalLogger.mu.Unlock()

	// format log message with context
	logMessage = fmt.Sprintf("[%s] %s", context, message)

	// print log message
	log.Println(logMessage)
}

// Logf logs a formatted message with context information.
// context is the function or method name providing context.
// format is the format string (printf-style).
// args are the format arguments.
func Logf(context, format string, args ...any) {
	var (
		message string
	)

	// check if logging is enabled
	globalLogger.mu.Lock()
	if !globalLogger.Enabled {
		globalLogger.mu.Unlock()
		return
	}
	globalLogger.mu.Unlock()

	// format message
	message = fmt.Sprintf(format, args...)

	// format log message with context
	message = fmt.Sprintf("[%s] %s", context, message)

	// print log message
	log.Println(message)
}
