package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"
)

type ctxKey string

const RequestIDKey ctxKey = "request_id"

var defaultLogger *slog.Logger

func init() {
	defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	}))
}

func Init(env string, level string) {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}

	var h slog.Handler
	if env == "development" {
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     l,
			AddSource: true,
		})
	} else {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     l,
			AddSource: false,
		})
	}

	slog.SetDefault(slog.New(h))
	defaultLogger = slog.Default()
}

func SetOutput(w io.Writer) {
	defaultLogger = slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func With(args ...any) *slog.Logger {
	return defaultLogger.With(args...)
}

func WithRequestID(ctx context.Context) *slog.Logger {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return defaultLogger.With("request_id", id)
	}
	return defaultLogger
}

func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

func Fatal(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
	os.Exit(1)
}

func Debugf(format string, v ...any) {
	msg, args := sprintFormat(format, v...)
	defaultLogger.Debug(msg, args...)
}

func Infof(format string, v ...any) {
	msg, args := sprintFormat(format, v...)
	defaultLogger.Info(msg, args...)
}

func Warnf(format string, v ...any) {
	msg, args := sprintFormat(format, v...)
	defaultLogger.Warn(msg, args...)
}

func Errorf(format string, v ...any) {
	msg, args := sprintFormat(format, v...)
	defaultLogger.Error(msg, args...)
}

func Fatalf(format string, v ...any) {
	msg, args := sprintFormat(format, v...)
	defaultLogger.Error(msg, args...)
	os.Exit(1)
}

func sprintFormat(format string, v ...any) (string, []any) {
	if len(v) == 0 {
		return format, nil
	}
	return format, v
}

type Timer struct {
	start time.Time
	msg   string
	args  []any
}

func StartTimer(msg string, args ...any) *Timer {
	_, file, line, _ := runtime.Caller(1)
	args = append(args, "caller", file+":"+itoa(line))
	return &Timer{start: time.Now(), msg: msg, args: args}
}

func (t *Timer) Stop(args ...any) {
	duration := time.Since(t.start)
	all := append(t.args, "duration", duration.String())
	all = append(all, args...)
	defaultLogger.Debug(t.msg, all...)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	n := len(buf)
	for i > 0 {
		n--
		buf[n] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[n:])
}

type ComponentLogger struct {
	component string
}

func For(component string) *ComponentLogger {
	return &ComponentLogger{component: component}
}

func (c *ComponentLogger) Debug(msg string, args ...any) {
	a := make([]any, 0, len(args)+2)
	a = append(a, "component", c.component)
	a = append(a, args...)
	defaultLogger.Debug(msg, a...)
}

func (c *ComponentLogger) Info(msg string, args ...any) {
	a := make([]any, 0, len(args)+2)
	a = append(a, "component", c.component)
	a = append(a, args...)
	defaultLogger.Info(msg, a...)
}

func (c *ComponentLogger) Warn(msg string, args ...any) {
	a := make([]any, 0, len(args)+2)
	a = append(a, "component", c.component)
	a = append(a, args...)
	defaultLogger.Warn(msg, a...)
}

func (c *ComponentLogger) Error(msg string, args ...any) {
	a := make([]any, 0, len(args)+2)
	a = append(a, "component", c.component)
	a = append(a, args...)
	defaultLogger.Error(msg, a...)
}
