package log4go

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/logutils"
)

type log4GoLogger struct {
	opts *LoggerOpts
}

func (l *log4GoLogger) SetFilter(filter *logutils.LevelFilter) {
	f := &logutils.LevelFilter{
		Levels:   logLevel,
		MinLevel: LogLevelInfo,
		Writer:   filter.Writer,
	}

	switch filter.MinLevel {
	case LogLevelDebug:
		fallthrough
	case LogLevelInfo:
		fallthrough
	case LogLevelWarn:
		fallthrough
	case LogLevelError:
		f.MinLevel = filter.MinLevel
	default:
		f.MinLevel = LogLevelInfo
	}

	log.SetOutput(f)
}

func (l *log4GoLogger) Debug(ctx context.Context, format string, v ...interface{}) {
	format = "[DBUG]" + format
	l.printf(ctx, format, v...)
}

func (l *log4GoLogger) Info(ctx context.Context, format string, v ...interface{}) {
	format = "[INFO]" + format
	l.printf(ctx, format, v...)
}

func (l *log4GoLogger) Warn(ctx context.Context, format string, v ...interface{}) {
	format = "[WARN]" + format
	l.printf(ctx, format, v...)
}

func (l *log4GoLogger) Error(ctx context.Context, format string, v ...interface{}) {
	format = "[EROR]" + format
	l.printf(ctx, format, v...)
}

func (l *log4GoLogger) printf(ctx context.Context, format string, v ...interface{}) {
	var id string
	var line string
	var x, y int

	line = fmt.Sprintf(format, v...)

	var logStr string
	if l.opts.WithId {
		id = getIdFromContext(ctx, l.opts.IdName)
		var tag = "[" + id + "]"
		// Check for a log level
		x = strings.Index(line, "[")
		if x >= 0 {
			y = strings.Index(line[x:], "]")
			if y >= 0 {
				logStr = line[:x+y+1] + tag + line[x+y+1:]
			}
		}
	} else {
		logStr = line
	}

	log.Output(3, logStr)
}

func getIdFromContext(ctx context.Context, idName string) string {
	id, ok := ctx.Value(idName).(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(id)
}
