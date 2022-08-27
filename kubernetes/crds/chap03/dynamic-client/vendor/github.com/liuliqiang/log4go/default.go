package log4go

import (
	"context"
	"log"

	"github.com/hashicorp/logutils"
)

// Return a default logger
func DefaultLogger() (logger Logger) {
	return NewLogger("", nil)
}

// Logging debug log with default logger
func Debug(format string, v ...interface{}) {
	DefaultLogger().Debug(context.Background(), format, v...)
}

// Logging info log with default logger
func Info(format string, v ...interface{}) {
	DefaultLogger().Info(context.Background(), format, v...)
}

// Logging warning log with default logger
func Warn(format string, v ...interface{}) {
	DefaultLogger().Warn(context.Background(), format, v...)
}

// Logging error log with default logger
func Error(format string, v ...interface{}) {
	DefaultLogger().Error(context.Background(), format, v...)
}

// Set log level for default logger
func SetLevel(level logutils.LogLevel) {
	f := DefaultLogger().GetFilter()
	f.MinLevel = level
	DefaultLogger().SetFilter(f)
}

// Set logger flag for default logger
func SetFlags(flag int) {
	log.SetFlags(flag)
}

// Set logger filter for default logger
func SetFilter(filter *logutils.LevelFilter) {
	DefaultLogger().SetFilter(filter)
}
