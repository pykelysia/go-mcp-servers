// Package logger provides a shared zerolog setup for all MCP servers.
// Logs go to stderr; stdout is reserved for the MCP stdio transport.
package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init configures a zerolog logger tagged with the server name and
// returns it. LOG_LEVEL=debug enables debug output; otherwise info.
func Init(serverName string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339
	level := zerolog.InfoLevel
	if os.Getenv("LOG_LEVEL") == "debug" {
		level = zerolog.DebugLevel
	}
	logger := zerolog.New(os.Stderr).
		Level(level).
		With().
		Timestamp().
		Str("server", serverName).
		Logger()
	log.Logger = logger
	return logger
}
