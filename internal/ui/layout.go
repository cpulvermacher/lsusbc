package ui

import "charm.land/lipgloss/v2"

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
		actualListWidth := lipgloss.Width(listContent)
		listWidth := min(actualListWidth, width*4/7)

		listStyle := lipgloss.NewStyle().Width(listWidth).Height(height - 1)
		detailsStyle := detailsStyleBase.
			Border(lipgloss.NormalBorder(), false, false, false, true).
			Margin(0, 1).
			Padding(0, 1).
			Width(width - listWidth).Height(height - 1)

		listPanel := listStyle.Render(listContent)
		detailsPanel := detailsStyle.Render(detailsContent)

		return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, detailsPanel)
	}
}
