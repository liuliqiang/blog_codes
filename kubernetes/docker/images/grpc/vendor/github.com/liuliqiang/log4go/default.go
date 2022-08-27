package log4go

import (
	"context"

	"github.com/hashicorp/logutils"
)

func DefaultLogger() (logger Logger) {
	return NewLogger("", nil)
}

func Debug(ctx context.Context, format string, v ...interface{}) {
	DefaultLogger().Debug(ctx, format, v...)
}

func Info(ctx context.Context, format string, v ...interface{}) {
	DefaultLogger().Info(ctx, format, v...)
}

func Warn(ctx context.Context, format string, v ...interface{}) {
	DefaultLogger().Warn(ctx, format, v...)
}

func Error(ctx context.Context, format string, v ...interface{}) {
	DefaultLogger().Error(ctx, format, v...)
}

func SetFilter(filter *logutils.LevelFilter) {
	DefaultLogger().SetFilter(filter)
}
