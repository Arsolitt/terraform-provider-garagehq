package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var capacityRegex = regexp.MustCompile(`^(?i)(\d+(?:\.\d+)?)\s*(k|m|g|t|kib|mib|gib|tib)?$`)

func ParseCapacity(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("capacity cannot be empty")
	}

	matches := capacityRegex.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid capacity format: %q (expected: 500M, 1G, 2TiB, etc.)", s)
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid capacity value: %q", matches[1])
	}

	unit := strings.ToLower(matches[2])
	multiplier := int64(1)

	switch unit {
	case "k", "kib":
		multiplier = 1024
	case "m", "mib":
		multiplier = 1024 * 1024
	case "g", "gib":
		multiplier = 1024 * 1024 * 1024
	case "t", "tib":
		multiplier = 1024 * 1024 * 1024 * 1024
	}

	result := int64(value * float64(multiplier))
	if result < 0 {
		return 0, fmt.Errorf("capacity cannot be negative: %q", s)
	}

	return result, nil
}

func FormatCapacity(b int64) string {
	if b <= 0 {
		return "0"
	}

	units := []struct {
		suffix string
		div    int64
	}{
		{"T", 1024 * 1024 * 1024 * 1024},
		{"G", 1024 * 1024 * 1024},
		{"M", 1024 * 1024},
		{"K", 1024},
	}

	for _, u := range units {
		if b >= u.div {
			// Check if it's a clean division
			if b%u.div == 0 {
				return fmt.Sprintf("%d%s", b/u.div, u.suffix)
			}
			// Format with decimal
			value := float64(b) / float64(u.div)
			// Avoid too many decimal places
			if value < 10 {
				return fmt.Sprintf("%.2f%s", value, u.suffix)
			}
			return fmt.Sprintf("%.1f%s", value, u.suffix)
		}
	}

	return fmt.Sprintf("%d", b)
}
