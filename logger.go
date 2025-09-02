package main

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

func NewLogger() *Logger {
	return NewLoggerWithLevel(WarnLevel)
}

func NewLoggerWithLevel(level Level) *Logger {
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

func (l *Logger) IsDebugEnabled() bool {
	return l.level <= DebugLevel
}

