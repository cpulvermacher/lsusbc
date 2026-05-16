package ui

import "charm.land/lipgloss/v2"

const (
	minWidthRatio   = 0.25
	idealWidthRatio = 0.5
)

// adjusts panel orientation and size based on terminal size and list width
func buildPanelLayout(width int, height int, listContent string, detailsContent string) string {
	detailsStyleBase := lipgloss.NewStyle().
		BorderForeground(lipgloss.Color("#8e8e8e"))

	if width < 70 {
		// narrow window => vertical layout
		listStyle := lipgloss.NewStyle().Width(width)
		detailsStyle := detailsStyleBase.
			Border(lipgloss.NormalBorder(), true, false, false, false).
			Padding(1, 0).
			Width(width)

		listPanel := listStyle.Render(listContent)
		detailsPanel := detailsStyle.Render(detailsContent)

		content := lipgloss.JoinVertical(lipgloss.Top, listPanel, detailsPanel)
		// reserve status bar line
		return lipgloss.NewStyle().Width(width).Height(height - 1).Render(content)
	} else {
		// horizontal layout
		reservedForDecorationsAndMargin := 2 // padding is included in details width
		availableWidth := width - reservedForDecorationsAndMargin
		// minimum width for both list and panel
		minWidth := int(float64(availableWidth) * minWidthRatio)
		maxWidth := availableWidth - minWidth

		actualListWidth := lipgloss.Width(listContent)
		actualDetailsWidth := lipgloss.Width(detailsContent)
		listWidth := min(maxWidth, max(minWidth, actualListWidth))
		detailsWidth := min(maxWidth, max(minWidth, actualDetailsWidth))
		idealListWidth := int(float64(availableWidth) * idealWidthRatio)

		if listWidth+detailsWidth > availableWidth {
			// insufficient space,
			layoutRatio := float64(listWidth) / float64(detailsWidth)
			minLayoutRatio := minWidthRatio / (1 - minWidthRatio)
			clampedRatio := min(1/minLayoutRatio, max(minLayoutRatio, layoutRatio))

			listWidth = int(float64(availableWidth) / (1 + 1/clampedRatio))
		} else if listWidth > idealListWidth {
			// constrained by list (nothing to do)
		} else if detailsWidth+idealListWidth > availableWidth {
			// constrained by details
			listWidth = availableWidth - detailsWidth
		} else {
			// both sides free
			listWidth = idealListWidth
		}
		detailsWidth = availableWidth - listWidth

		listStyle := lipgloss.NewStyle().Width(listWidth).Height(height - 1)
		detailsStyle := detailsStyleBase.
			Border(lipgloss.NormalBorder(), false, false, false, true).
			Margin(0, 1).
			Padding(0, 1).
			Width(detailsWidth).Height(height - 1)

		listPanel := listStyle.Render(listContent)
		detailsPanel := detailsStyle.Render(detailsContent)

		return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, detailsPanel)
	}
}
