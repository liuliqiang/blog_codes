package log4go

import (
	"context"
	"log"
	"os"

	"github.com/hashicorp/logutils"
)

var defaultIdName = "x-log4go-id"
var loggerMap map[string]Logger
var LogLevelDebug = logutils.LogLevel("DBUG")
var LogLevelInfo = logutils.LogLevel("INFO")
var LogLevelWarn = logutils.LogLevel("WARN")
var LogLevelError = logutils.LogLevel("EROR")
var logLevel = []logutils.LogLevel{
	LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError,
}

func init() {
	loggerMap = map[string]Logger{}
}

type Logger interface {
	Debug(ctx context.Context, format string, v ...interface{})
	Info(ctx context.Context, format string, v ...interface{})
	Warn(ctx context.Context, format string, v ...interface{})
	Error(ctx context.Context, format string, v ...interface{})
	SetFilter(filter *logutils.LevelFilter)
}

type LoggerOpts struct {
	WithId bool
	IdName string
}

func NewLogger(name string, opts *LoggerOpts) (logger Logger) {
	var exists bool
	if logger, exists = loggerMap[name]; !exists {
		filter := &logutils.LevelFilter{
			Levels:   logLevel,
			MinLevel: LogLevelInfo,
			Writer:   os.Stdout,
		}
		log.SetOutput(filter)
		logger = &log4GoLogger{formatOpts(opts)}
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
