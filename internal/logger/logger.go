package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var (
	writer  io.Writer
	output  io.Writer
	level   Level = INFO
	mu      sync.Mutex
	timefmt = "2006-01-02T15:04:05.000Z07:00"
)

func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	writer = w
}

func SetLevel(l Level) {
	mu.Lock()
	defer mu.Unlock()
	level = l
}

func init() {
	writer = os.Stdout
}

func Debug(msg string, args ...interface{}) {
	log(DEBUG, msg, args...)
}

func Info(msg string, args ...interface{}) {
	log(INFO, msg, args...)
}

func Warn(msg string, args ...interface{}) {
	log(WARN, msg, args...)
}

func Error(msg string, args ...interface{}) {
	log(ERROR, msg, args...)
}

func log(l Level, msg string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if l < level {
		return
	}

	levelStr := []string{"DEBUG", "INFO", "WARN", "ERROR"}[l]
	timestamp := time.Now().Format(timefmt)
	msg = fmt.Sprintf(msg, args...)

	fmt.Fprintf(writer, "%s %s %s\n", timestamp, levelStr, msg)
}

type Structured struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

func InfoWithFields(msg string, fields map[string]interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if INFO < level {
		return
	}

	s := Structured{
		Timestamp: time.Now().Format(timefmt),
		Level:     "INFO",
		Message:   msg,
		Fields:    fields,
	}
	fmt.Fprintf(writer, "%s\n", mustMarshalJSON(s))
}

func ErrorWithFields(msg string, fields map[string]interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if ERROR < level {
		return
	}

	s := Structured{
		Timestamp: time.Now().Format(timefmt),
		Level:     "ERROR",
		Message:   msg,
		Fields:    fields,
	}
	fmt.Fprintf(writer, "%s\n", mustMarshalJSON(s))
}

func mustMarshalJSON(v interface{}) string {
	s, ok := v.(Structured)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return fmt.Sprintf(`{"timestamp":"%s","level":"%s","message":"%s"}`,
		s.Timestamp, s.Level, s.Message)
}
