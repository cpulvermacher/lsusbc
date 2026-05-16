package ui

import (
	"strings"
	"testing"
)

// findPosition returns the (line, col) of the first occurrence of s in text, or (-1, -1) if not found.
// Column counts only visible characters (ANSI escape sequences are ignored).
func findPosition(text, s string) (int, int) {
	for i, line := range strings.Split(text, "\n") {
		stripped := ansiEscape.ReplaceAllString(line, "")
		if col := strings.Index(stripped, s); col != -1 {
			return i, col
		}
	}
	return -1, -1
}

func TestBuildPanelLayout_NarrowUsesVerticalLayout(t *testing.T) {
	result := buildPanelLayout(60, 20, "list", "details")
	if !strings.Contains(result, "list") || !strings.Contains(result, "details") {
		t.Errorf("expected both panels in output, got: %q", result)
	}

	listLine, listCol := findPosition(result, "list")
	detailsLine, detailsCol := findPosition(result, "details")
	if listLine == -1 || detailsLine == -1 || listLine >= detailsLine {
		t.Errorf("expected list (line %d) above details (line %d)", listLine, detailsLine)
	}
	if listCol != 0 || detailsCol != 0 {
		t.Errorf("expected both panels at column 0, got list col %d, details col %d", listCol, detailsCol)
	}
}

func TestBuildPanelLayout_WideUsesHorizontalLayout(t *testing.T) {
	result := buildPanelLayout(100, 20, "list", "details")
	if !strings.Contains(result, "list") || !strings.Contains(result, "details") {
		t.Errorf("expected both panels in output, got: %q", result)
	}

_, detailsCol := findPosition(result, "│")
	if detailsCol != 50 {
		t.Errorf("expected details at column %d, got %d", 50, detailsCol)
	}
}

func TestBuildPanelLayout_HorizontalLayoutWithWideListGrowsUntilMaximum(t *testing.T) {
	result := buildPanelLayout(100, 20, "list"+strings.Repeat("x", 200), "details")
	if !strings.Contains(result, "list") || !strings.Contains(result, "details") {
		t.Errorf("expected both panels in output, got: %q", result)
	}

	_, detailsCol := findPosition(result, "│")
	if detailsCol != 75 {
		t.Errorf("expected details at column %d, got %d", 75, detailsCol)
	}
}

func TestBuildPanelLayout_HorizontalLayoutWithWideDetailsGrowsUntilMaximum(t *testing.T) {
	result := buildPanelLayout(100, 20, "list", "details"+strings.Repeat("x", 200))
	if !strings.Contains(result, "list") || !strings.Contains(result, "details") {
		t.Errorf("expected both panels in output, got: %q", result)
	}

	_, detailsCol := findPosition(result, "│")
	if detailsCol != 25 {
		t.Errorf("expected details at column %d, got %d", 25, detailsCol)
	}
}

func TestBuildPanelLayout_HorizontalLayoutWithWideContentOnBothSidesSplitsFairly(t *testing.T) {
	result := buildPanelLayout(100, 20, "list"+strings.Repeat("x", 200), "details"+strings.Repeat("x", 200))
	if !strings.Contains(result, "list") || !strings.Contains(result, "details") {
		t.Errorf("expected both panels in output, got: %q", result)
	}

	_, detailsCol := findPosition(result, "│")
	if detailsCol != 50 {
		t.Errorf("expected details at column %d, got %d", 50, detailsCol)
	}
}
