package main

import (
	"reflect"
	"testing"
)

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"A", "B", "C"}, // removeDuplicates converts to uppercase
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"A", "B", "C"}, // removeDuplicates converts to uppercase
		},
		{
			name:     "all duplicates",
			input:    []string{"a", "a", "a"},
			expected: []string{"A"}, // removeDuplicates converts to uppercase
		},
		{
			name:     "single element",
			input:    []string{"a"},
			expected: []string{"A"}, // removeDuplicates converts to uppercase
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeDuplicates(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("removeDuplicates(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMdQuote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "normal text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "text with pipe",
			input:    "hello | world",
			expected: "hello \\| world",
		},
		{
			name:     "multiple pipes",
			input:    "a | b | c",
			expected: "a \\| b \\| c",
		},
		{
			name:     "pipe at start",
			input:    "| hello",
			expected: "\\| hello",
		},
		{
			name:     "pipe at end",
			input:    "hello |",
			expected: "hello \\|",
		},
		{
			name:     "only pipe",
			input:    "|",
			expected: "\\|",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mdQuote(tt.input)
			if result != tt.expected {
				t.Errorf("mdQuote(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}