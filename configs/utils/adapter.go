package utils

import (
	"fmt"
	"strconv"
)

func StringToUint(s string) (uint, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string cannot be converted to uint")
	}

	val, err := strconv.ParseUint(s, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to convert '%s' to uint: %w", s, err)
	}

	return uint(val), nil
}

// StringToUintWithDefault converts string to uint with a default value
// Returns the default value if conversion fails
func StringToUintWithDefault(s string, defaultVal uint) uint {
	result, err := StringToUint(s)
	if err != nil {
		return defaultVal
	}
	return result
}
