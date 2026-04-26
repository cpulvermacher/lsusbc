package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cpulvermacher/lsusbc/internal/ui"
)

func main() {
	// Default to /sys/class/typec, allow override with argument
	typecDir := "/sys/class/typec"
	if len(os.Args) >= 2 {
		typecDir = os.Args[1]
	}

	// Check if directory exists
	if _, err := os.Stat(typecDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: directory does not exist: %s\n", typecDir)
		os.Exit(1)
	}

	if _, err := tea.NewProgram(newModel(typecDir), tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func newModel(typecDir string) ui.UIModel {
	return ui.UIModel{
		TypecDir: typecDir,
	}
}
