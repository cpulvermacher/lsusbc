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

func TestComputeListWidth_NarrowReturnsZero(t *testing.T) {
	w := computeListWidth(60, "list", "details")
	if w != 0 {
		t.Errorf("expected 0 for narrow terminal, got %d", w)
	}
}

func TestComputeListWidth_WideDefaultsToIdeal(t *testing.T) {
	w := computeListWidth(100, "list", "details")
	if w != 49 { // idealWidthRatio * (100-2) = 49
		t.Errorf("expected 49, got %d", w)
	}
}

func TestComputeListWidth_WideListGrowsUntilMaximum(t *testing.T) {
	w := computeListWidth(100, "list"+strings.Repeat("x", 200), "details")
	if w != 74 { // maxWidth = availableWidth - minWidth = 98 - 24 = 74
		t.Errorf("expected 74, got %d", w)
	}
}

func TestComputeListWidth_WideDetailsGrowsUntilMaximum(t *testing.T) {
	w := computeListWidth(100, "list", "details"+strings.Repeat("x", 200))
	if w != 24 { // minWidth = (100-2) * 0.25 = 24
		t.Errorf("expected 24, got %d", w)
	}
}

func TestComputeListWidth_WideContentOnBothSidesSplitsFairly(t *testing.T) {
	w := computeListWidth(100, "list"+strings.Repeat("x", 200), "details"+strings.Repeat("x", 200))
	if w != 49 {
		t.Errorf("expected 49, got %d", w)
	}
}

func TestBuildPanelLayout_NarrowUsesVerticalLayout(t *testing.T) {
	result := buildPanelLayout(60, 20, 0, "list", "details")
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
	listWidth := computeListWidth(100, "list", "details")
	result := buildPanelLayout(100, 20, listWidth, "list", "details")
	if !strings.Contains(result, "list") || !strings.Contains(result, "details") {
		t.Errorf("expected both panels in output, got: %q", result)
	}

	// │ appears one column after listWidth due to left margin on the details panel
	_, detailsCol := findPosition(result, "│")
	if detailsCol != listWidth+1 {
		t.Errorf("expected divider at column %d, got %d", listWidth+1, detailsCol)
	}
}

func TestBuildPanelLayout_CachedListWidthIsStable(t *testing.T) {
	// Simulate: compute layout with short details, then render with wide details.
	// The divider should not move.
	listContent := "list"
	shortDetails := "details"
	wideDetails := "details" + strings.Repeat("x", 200)

	listWidth := computeListWidth(100, listContent, shortDetails)

	_, shortCol := findPosition(buildPanelLayout(100, 20, listWidth, listContent, shortDetails), "│")
	_, wideCol := findPosition(buildPanelLayout(100, 20, listWidth, listContent, wideDetails), "│")

	if shortCol != wideCol {
		t.Errorf("expected divider stable at column %d, but moved to %d when details grew", shortCol, wideCol)
	}
}
