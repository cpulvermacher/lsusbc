package parser

import (
	"os"
	"strconv"
	"strings"
)

// extractActiveRole extracts the active role from bracketed format (e.g., "[host] device" -> "host")
func extractActiveRole(content string) string {
	content = strings.TrimSpace(content)
	start := strings.Index(content, "[")
	end := strings.Index(content, "]")
	if start != -1 && end != -1 && end > start {
		return content[start+1 : end]
	}
	return content
}

// readFile reads a file and returns its (trimmed) content, or empty string on error
func readFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

// parseMilliValue parses values like "5000mV" or "3000mA" and returns the integer value
func parseMilliValue(s string) (int, error) {
	s = strings.TrimSpace(s)
	// Remove the unit suffix (mV, mA, etc.)
	s = strings.TrimSuffix(s, "mV")
	s = strings.TrimSuffix(s, "mA")
	s = strings.TrimSpace(s)

	return strconv.Atoi(s)
}
