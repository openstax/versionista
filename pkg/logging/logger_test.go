package logging

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", DebugLevel},
		{"DEBUG", DebugLevel},
		{"info", InfoLevel},
		{"INFO", InfoLevel},
		{"warn", WarnLevel},
		{"WARN", WarnLevel},
		{"warning", WarnLevel},
		{"WARNING", WarnLevel},
		{"error", ErrorLevel},
		{"ERROR", ErrorLevel},
		{"invalid", WarnLevel}, // default
		{"", WarnLevel},        // default
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := ParseLevel(test.input)
			if result != test.expected {
				t.Errorf("ParseLevel(%q) = %v, expected %v", test.input, result, test.expected)
			}
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		level         Level
		shouldLogDebug bool
		shouldLogInfo  bool
		shouldLogWarn  bool
		shouldLogError bool
	}{
		{DebugLevel, true, true, true, true},
		{InfoLevel, false, true, true, true},
		{WarnLevel, false, false, true, true},
		{ErrorLevel, false, false, false, true},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			logger := NewWithLevel(test.level)

			// Capture output
			var debugBuf, infoBuf, warnBuf, errorBuf bytes.Buffer
			logger.debugLogger = log.New(&debugBuf, "[DEBUG] ", 0)
			logger.infoLogger = log.New(&infoBuf, "[INFO] ", 0)
			logger.warnLogger = log.New(&warnBuf, "[WARN] ", 0)
			logger.errorLogger = log.New(&errorBuf, "[ERROR] ", 0)

			// Test each log method
			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message")

			// Check debug output
			debugOutput := debugBuf.String()
			if test.shouldLogDebug && !strings.Contains(debugOutput, "debug message") {
				t.Errorf("Expected debug message to be logged at level %v", test.level)
			}
			if !test.shouldLogDebug && strings.Contains(debugOutput, "debug message") {
				t.Errorf("Expected debug message NOT to be logged at level %v", test.level)
			}

			// Check info output
			infoOutput := infoBuf.String()
			if test.shouldLogInfo && !strings.Contains(infoOutput, "info message") {
				t.Errorf("Expected info message to be logged at level %v", test.level)
			}
			if !test.shouldLogInfo && strings.Contains(infoOutput, "info message") {
				t.Errorf("Expected info message NOT to be logged at level %v", test.level)
			}

			// Check warn output
			warnOutput := warnBuf.String()
			if test.shouldLogWarn && !strings.Contains(warnOutput, "warn message") {
				t.Errorf("Expected warn message to be logged at level %v", test.level)
			}
			if !test.shouldLogWarn && strings.Contains(warnOutput, "warn message") {
				t.Errorf("Expected warn message NOT to be logged at level %v", test.level)
			}

			// Check error output
			errorOutput := errorBuf.String()
			if test.shouldLogError && !strings.Contains(errorOutput, "error message") {
				t.Errorf("Expected error message to be logged at level %v", test.level)
			}
			if !test.shouldLogError && strings.Contains(errorOutput, "error message") {
				t.Errorf("Expected error message NOT to be logged at level %v", test.level)
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	logger := New()
	if logger.level != WarnLevel {
		t.Errorf("Expected default level to be WarnLevel, got %v", logger.level)
	}
}

func TestNewLoggerWithLevel(t *testing.T) {
	logger := NewWithLevel(DebugLevel)
	if logger.level != DebugLevel {
		t.Errorf("Expected level to be DebugLevel, got %v", logger.level)
	}
}