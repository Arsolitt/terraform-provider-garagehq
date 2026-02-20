package main

import (
	"testing"
)

func TestParseCapacity(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"1024", 1024, false},
		{"1K", 1024, false},
		{"1k", 1024, false},
		{"1KiB", 1024, false},
		{"1M", 1024 * 1024, false},
		{"1MiB", 1024 * 1024, false},
		{"1G", 1024 * 1024 * 1024, false},
		{"1GiB", 1024 * 1024 * 1024, false},
		{"1T", 1024 * 1024 * 1024 * 1024, false},
		{"1TiB", 1024 * 1024 * 1024 * 1024, false},
		{"500M", 500 * 1024 * 1024, false},
		{"1.5G", int64(1.5 * 1024 * 1024 * 1024), false},
		{"2.5T", int64(2.5 * 1024 * 1024 * 1024 * 1024), false},
		{" 1G ", 1024 * 1024 * 1024, false},
		{"", 0, true},
		{"abc", 0, true},
		{"-1G", 0, true},
		{"1X", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseCapacity(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("ParseCapacity(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("ParseCapacity(%q) unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("ParseCapacity(%q) = %d, expected %d", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestFormatCapacity(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{500, "500"},
		{1024, "1K"},
		{1024 * 1024, "1M"},
		{1024 * 1024 * 1024, "1G"},
		{1024 * 1024 * 1024 * 1024, "1T"},
		{500 * 1024 * 1024, "500M"},
		{1536 * 1024 * 1024, "1.50G"},
		{2560 * 1024 * 1024 * 1024, "2.50T"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatCapacity(tt.input)
			if result != tt.expected {
				t.Errorf("FormatCapacity(%d) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
