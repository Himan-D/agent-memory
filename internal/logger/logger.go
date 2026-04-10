package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
	LevelFatal Level = "FATAL"
)

type Logger struct {
	mu       sync.Mutex
	out      io.Writer
	minLevel Level
	fields   map[string]interface{}
}

type LogEntry struct {
	Time     string                 `json:"timestamp"`
	Level    Level                  `json:"level"`
	Message  string                 `json:"message"`
	Fields   map[string]interface{} `json:"fields,omitempty"`
	Caller   string                 `json:"caller,omitempty"`
	Duration string                 `json:"duration,omitempty"`
}

var defaultLogger *Logger
var levelPriority = map[Level]int{
	LevelDebug: 0,
	LevelInfo:  1,
	LevelWarn:  2,
	LevelError: 3,
	LevelFatal: 4,
}

func init() {
	defaultLogger = New()
}

func New() *Logger {
	return &Logger{
		out:      os.Stdout,
		minLevel: LevelInfo,
		fields:   make(map[string]interface{}),
	}
}

func SetOutput(w io.Writer) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.out = w
}

func SetLevel(level Level) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.minLevel = level
}

func With(fields map[string]interface{}) *Logger {
	l := &Logger{
		out:      defaultLogger.out,
		minLevel: defaultLogger.minLevel,
		fields:   mergeFields(defaultLogger.fields, fields),
	}
	return l
}

func Debug(msg string, fields ...map[string]interface{}) {
	defaultLogger.log(LevelDebug, msg, fields...)
}

func Info(msg string, fields ...map[string]interface{}) {
	defaultLogger.log(LevelInfo, msg, fields...)
}

func Warn(msg string, fields ...map[string]interface{}) {
	defaultLogger.log(LevelWarn, msg, fields...)
}

func Error(msg string, fields ...map[string]interface{}) {
	defaultLogger.log(LevelError, msg, fields...)
}

func Fatal(msg string, fields ...map[string]interface{}) {
	defaultLogger.log(LevelFatal, msg, fields...)
	os.Exit(1)
}

func Debugf(format string, args ...interface{}) {
	Debug(fmt.Sprintf(format, args...))
}

func Infof(format string, args ...interface{}) {
	Info(fmt.Sprintf(format, args...))
}

func Warnf(format string, args ...interface{}) {
	Warn(fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...interface{}) {
	Error(fmt.Sprintf(format, args...))
}

func Fatalf(format string, args ...interface{}) {
	Fatal(fmt.Sprintf(format, args...))
}

func (l *Logger) log(level Level, msg string, fields ...map[string]interface{}) {
	if levelPriority[level] < levelPriority[l.minLevel] {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	allFields := l.fields
	for _, f := range fields {
		for k, v := range f {
			allFields[k] = v
		}
	}

	entry := LogEntry{
		Time:    time.Now().UTC().Format(time.RFC3339Nano),
		Level:   level,
		Message: msg,
		Fields:  allFields,
	}

	data, _ := json.Marshal(entry)
	l.out.Write(append(data, '\n'))
}

func (l *Logger) With(fields map[string]interface{}) *Logger {
	return &Logger{
		out:      l.out,
		minLevel: l.minLevel,
		fields:   mergeFields(l.fields, fields),
	}
}

func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	l.log(LevelDebug, msg, fields...)
}

func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	l.log(LevelInfo, msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	l.log(LevelWarn, msg, fields...)
}

func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	l.log(LevelError, msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...map[string]interface{}) {
	l.log(LevelFatal, msg, fields...)
	os.Exit(1)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Debug(fmt.Sprintf(format, args...))
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Warn(fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Fatal(fmt.Sprintf(format, args...))
}

func mergeFields(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

type Timer struct {
	start time.Time
	msg   string
}

func StartTimer(msg string) *Timer {
	return &Timer{
		start: time.Now(),
		msg:   msg,
	}
}

func (t *Timer) Stop(fields ...map[string]interface{}) {
	duration := time.Since(t.start)
	merged := map[string]interface{}{
		"duration": duration.String(),
	}
	for _, f := range fields {
		for k, v := range f {
			merged[k] = v
		}
	}
	defaultLogger.log(LevelDebug, t.msg, merged)
}

type StructuredLogger struct {
	component string
	logger    *Logger
}

func ForComponent(component string) *StructuredLogger {
	return &StructuredLogger{
		component: component,
		logger:    defaultLogger,
	}
}

func (s *StructuredLogger) With(fields map[string]interface{}) *StructuredLogger {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["component"] = s.component
	return &StructuredLogger{
		component: s.component,
		logger:    s.logger.With(fields),
	}
}

func (s *StructuredLogger) Debug(msg string, fields ...map[string]interface{}) {
	merged := map[string]interface{}{"component": s.component}
	for _, f := range fields {
		for k, v := range f {
			merged[k] = v
		}
	}
	s.logger.log(LevelDebug, msg, merged)
}

func (s *StructuredLogger) Info(msg string, fields ...map[string]interface{}) {
	merged := map[string]interface{}{"component": s.component}
	for _, f := range fields {
		for k, v := range f {
			merged[k] = v
		}
	}
	s.logger.log(LevelInfo, msg, merged)
}

func (s *StructuredLogger) Warn(msg string, fields ...map[string]interface{}) {
	merged := map[string]interface{}{"component": s.component}
	for _, f := range fields {
		for k, v := range f {
			merged[k] = v
		}
	}
	s.logger.log(LevelWarn, msg, merged)
}

func (s *StructuredLogger) Error(msg string, fields ...map[string]interface{}) {
	merged := map[string]interface{}{"component": s.component}
	for _, f := range fields {
		for k, v := range f {
			merged[k] = v
		}
	}
	s.logger.log(LevelError, msg, merged)
}

func (s *StructuredLogger) Fatal(msg string, fields ...map[string]interface{}) {
	merged := map[string]interface{}{"component": s.component}
	for _, f := range fields {
		for k, v := range f {
			merged[k] = v
		}
	}
	s.logger.log(LevelFatal, msg, merged)
}
