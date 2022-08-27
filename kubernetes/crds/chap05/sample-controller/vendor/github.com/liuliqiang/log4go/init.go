package log4go

import (
	"context"
	"log"
	"os"

	"github.com/hashicorp/logutils"
)

var defaultIdName = "x-log4go-id"
var loggerMap map[string]Logger
// debug log level
var LogLevelDebug = logutils.LogLevel("DBUG")
// info log level
var LogLevelInfo = logutils.LogLevel("INFO")
// warning log level
var LogLevelWarn = logutils.LogLevel("WARN")
// error log level
var LogLevelError = logutils.LogLevel("EROR")
var logLevel = []logutils.LogLevel{
	LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError,
}

func init() {
	loggerMap = map[string]Logger{}
}

// Interface for logger, you can implement your own logger with this.
type Logger interface {
	Debug(ctx context.Context, format string, v ...interface{})
	Info(ctx context.Context, format string, v ...interface{})
	Warn(ctx context.Context, format string, v ...interface{})
	Error(ctx context.Context, format string, v ...interface{})
	SetFilter(filter *logutils.LevelFilter)
	GetFilter() (filter *logutils.LevelFilter)
}

// Options for logger.
type LoggerOpts struct {
	WithId bool
	IdName string
}

// Create a named logger with specify options.
func NewLogger(name string, opts *LoggerOpts) (logger Logger) {
	var exists bool
	if logger, exists = loggerMap[name]; !exists {
		filter := &logutils.LevelFilter{
			Levels:   logLevel,
			MinLevel: LogLevelInfo,
			Writer:   os.Stdout,
		}
		log.SetOutput(filter)
		logger = &log4GoLogger{filter: filter, opts: formatOpts(opts)}
		loggerMap[name] = logger
	}

	return loggerMap[name]
}

func formatOpts(opts *LoggerOpts) *LoggerOpts {
	if opts == nil {
		return &LoggerOpts{
			WithId: false,
		}
	}

	if opts.IdName == "" {
		opts.IdName = defaultIdName
	}

	return opts
}
