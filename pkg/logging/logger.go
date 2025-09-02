package logging

import (
	"log"
	"os"
	"strings"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

func ParseLevel(levelStr string) Level {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return DebugLevel
	case "INFO":
		return InfoLevel
	case "WARN", "WARNING":
		return WarnLevel
	case "ERROR":
		return ErrorLevel
	default:
		return WarnLevel
	}
}

type Logger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	level       Level
}

func New() *Logger {
	return NewWithLevel(WarnLevel)
}

func NewWithLevel(level Level) *Logger {
	return &Logger{
		infoLogger:  log.New(os.Stdout, "[INFO] ", log.LstdFlags),
		warnLogger:  log.New(os.Stdout, "[WARN] ", log.LstdFlags),
		errorLogger: log.New(os.Stderr, "[ERROR] ", log.LstdFlags),
		debugLogger: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags),
		level:       level,
	}
}

func (l *Logger) Info(format string, args ...interface{}) {
	if l.level <= InfoLevel {
		l.infoLogger.Printf(format, args...)
	}
}

func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= WarnLevel {
		l.warnLogger.Printf(format, args...)
	}
}

func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= ErrorLevel {
		l.errorLogger.Printf(format, args...)
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level <= DebugLevel {
		l.debugLogger.Printf(format, args...)
	}
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	l.errorLogger.Printf(format, args...)
	os.Exit(1)
}

func (l *Logger) FatalErr(err error, message string) {
	if err != nil {
		l.errorLogger.Printf("%s: %v", message, err)
		os.Exit(1)
	}
}

var DefaultLogger = New()

func Info(format string, args ...interface{}) {
	DefaultLogger.Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	DefaultLogger.Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	DefaultLogger.Error(format, args...)
}

func Debug(format string, args ...interface{}) {
	DefaultLogger.Debug(format, args...)
}

func Fatal(format string, args ...interface{}) {
	DefaultLogger.Fatal(format, args...)
}

func FatalErr(err error, message string) {
	DefaultLogger.FatalErr(err, message)
}