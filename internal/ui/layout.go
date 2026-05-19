package ui

import "charm.land/lipgloss/v2"

const (
	minWidthRatio                   = 0.25
	idealWidthRatio                 = 0.5
	reservedForDecorationsAndMargin = 2
)

// computeListWidth calculates the list panel width for horizontal layout.
// Returns 0 for narrow terminals that use vertical layout.
func computeListWidth(termWidth int, listContent string, detailsContent string) int {
	if termWidth < 70 {
		return 0
	}
	availableWidth := termWidth - reservedForDecorationsAndMargin
	minWidth := int(float64(availableWidth) * minWidthRatio)
	maxWidth := availableWidth - minWidth

	actualListWidth := lipgloss.Width(listContent)
	actualDetailsWidth := lipgloss.Width(detailsContent)
	listWidth := min(maxWidth, max(minWidth, actualListWidth))
	detailsWidth := min(maxWidth, max(minWidth, actualDetailsWidth))
	idealListWidth := int(float64(availableWidth) * idealWidthRatio)

	if listWidth+detailsWidth > availableWidth {
		layoutRatio := float64(listWidth) / float64(detailsWidth)
		minLayoutRatio := minWidthRatio / (1 - minWidthRatio)
		clampedRatio := min(1/minLayoutRatio, max(minLayoutRatio, layoutRatio))
		listWidth = int(float64(availableWidth) / (1 + 1/clampedRatio))
	} else if listWidth > idealListWidth {
		// constrained by list (nothing to do)
	} else if detailsWidth+idealListWidth > availableWidth {
		listWidth = availableWidth - detailsWidth
	} else {
		listWidth = idealListWidth
	}
	return listWidth
}

// buildPanelLayout renders list and details panels with the given panel widths.
// listWidth should come from computeListWidth; pass 0 to force vertical layout.
func buildPanelLayout(termWidth int, termHeight int, listWidth int, listContent string, detailsContent string) string {
	detailsStyleBase := lipgloss.NewStyle().
		BorderForeground(lipgloss.Color("#8e8e8e"))

	if termWidth < 70 {
		// narrow window => vertical layout
		listStyle := lipgloss.NewStyle().Width(termWidth)
		detailsStyle := detailsStyleBase.
			Border(lipgloss.NormalBorder(), true, false, false, false).
			Padding(1, 0).
			Width(termWidth)

		listPanel := listStyle.Render(listContent)
		detailsPanel := detailsStyle.Render(detailsContent)

		content := lipgloss.JoinVertical(lipgloss.Top, listPanel, detailsPanel)
		// reserve status bar line
		return lipgloss.NewStyle().Width(termWidth).Height(termHeight - 1).Render(content)
	}

	availableWidth := termWidth - reservedForDecorationsAndMargin
	detailsWidth := availableWidth - listWidth

	listStyle := lipgloss.NewStyle().Width(listWidth).Height(termHeight - 1)
	detailsStyle := detailsStyleBase.
		Border(lipgloss.NormalBorder(), false, false, false, true).
		Margin(0, 1).
		Padding(0, 1).
		Width(detailsWidth).Height(termHeight - 1)

	listPanel := listStyle.Render(listContent)
	detailsPanel := detailsStyle.Render(detailsContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, detailsPanel)
}
