package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cpulvermacher/lsusbc/internal/ui"
)

func main() {
	typecDir := flag.String("d", "/sys/class/typec", "typec sysfs directory")
	listFlag := flag.Bool("l", false, "list devices and exit")
	flag.Parse()

	if flag.NArg() > 0 {
		*typecDir = flag.Arg(0)
	}

	// Check if directory exists
	if _, err := os.Stat(*typecDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: directory does not exist: %s\n", *typecDir)
		os.Exit(1)
	}

	if *listFlag {
		ui.ListPorts(*typecDir)
		return
	}

	if _, err := tea.NewProgram(newModel(*typecDir), tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func newModel(typecDir string) ui.UIModel {
	return ui.UIModel{
		TypecDir: typecDir,
	}
}
