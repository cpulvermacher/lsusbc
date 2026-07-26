package parser

import (
	"os"
	"strconv"
	"strings"
	"unicode"
)

// extractActiveRole extracts the active role from bracketed format (e.g., "[host] device" -> "host")
func extractActiveRole(content string) string {
	content = strings.TrimSpace(content)
	start := strings.Index(content, "[")
	end := strings.Index(content, "]")
	if start != -1 && end != -1 && end > start {
		return content[start+1 : end]
	}
	return ""
}

// readFile reads a file and returns its (trimmed, sanitized) content, or empty
// string on error. See stripControl: file contents are attacker-controlled
// (USB descriptor strings etc.) and are rendered to the terminal.
func readFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return stripControl(strings.TrimSpace(string(content)))
}

// stripControl removes control characters (C0/C1 and DEL) from device-controlled
// strings so they cannot inject terminal escape sequences (e.g. ESC, BEL) when
// rendered. Operates on runes, so valid multi-byte UTF-8 (accents, CJK, ...) is
// preserved. Returns s unchanged when it contains no control characters.
func stripControl(s string) string {
	if strings.IndexFunc(s, unicode.IsControl) == -1 {
		return s
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
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
