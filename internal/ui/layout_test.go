package ui

import (
	"strings"
	"testing"
)

// findPosition returns the (line, col) of the first occurrence of s in text, or (-1, -1) if not found.
func findPosition(text, s string) (int, int) {
	for i, line := range strings.Split(text, "\n") {
		if col := strings.Index(line, s); col != -1 {
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

	listLine, _ := findPosition(result, "list")
	detailsLine, detailsCol := findPosition(result, "details")
	if listLine != detailsLine {
		t.Errorf("expected list and details on the same line in horizontal layout")
	}
	if detailsCol != 52 {
		t.Errorf("expected details at column %d, got %d", 52, detailsCol)
	}
}

func TestBuildPanelLayout_HorizontalLayoutWithWideListGrowsUntilMaximum(t *testing.T) {
	result := buildPanelLayout(100, 20, "list"+strings.Repeat("x", 200), "details")
	if !strings.Contains(result, "list") || !strings.Contains(result, "details") {
		t.Errorf("expected both panels in output, got: %q", result)
	}

	listLine, _ := findPosition(result, "list")
	detailsLine, detailsCol := findPosition(result, "details")
	if listLine != detailsLine {
		t.Errorf("expected list and details on the same line in horizontal layout")
	}
	if detailsCol != 77 {
		t.Errorf("expected details at column %d, got %d", 77, detailsCol)
	}
}

func TestBuildPanelLayout_HorizontalLayoutWithWideDetailsGrowsUntilMaximum(t *testing.T) {
	result := buildPanelLayout(100, 20, "list", "details"+strings.Repeat("x", 200))
	if !strings.Contains(result, "list") || !strings.Contains(result, "details") {
		t.Errorf("expected both panels in output, got: %q", result)
	}

	listLine, _ := findPosition(result, "list")
	detailsLine, detailsCol := findPosition(result, "details")
	if listLine != detailsLine {
		t.Errorf("expected list and details on the same line in horizontal layout")
	}
	if detailsCol != 26 {
		t.Errorf("expected details at column %d, got %d", 26, detailsCol)
	}
}

func TestBuildPanelLayout_HorizontalLayoutWithWideContentOnBothSidesSplitsFairly(t *testing.T) {
	result := buildPanelLayout(100, 20, "list"+strings.Repeat("x", 200), "details"+strings.Repeat("x", 200))
	if !strings.Contains(result, "list") || !strings.Contains(result, "details") {
		t.Errorf("expected both panels in output, got: %q", result)
	}

	listLine, _ := findPosition(result, "list")
	detailsLine, detailsCol := findPosition(result, "details")
	if listLine != detailsLine {
		t.Errorf("expected list and details on the same line in horizontal layout")
	}
	if detailsCol != 52 {
		t.Errorf("expected details at column %d, got %d", 52, detailsCol)
	}
}
