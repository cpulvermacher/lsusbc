package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/cpulvermacher/lsusbc/internal/ui"
)

func main() {
	sysfsDir := flag.String("d", "/sys", "sysfs directory")
	listFlag := flag.Bool("l", false, "list devices and exit")
	flag.Parse()

	if flag.NArg() > 0 {
		*sysfsDir = flag.Arg(0)
	}

	// Check if directory exists
	if _, err := os.Stat(*sysfsDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: directory does not exist: %s\n", *sysfsDir)
		os.Exit(1)
	}

	if *listFlag || !term.IsTerminal(os.Stdout.Fd()) {
		ui.ListPorts(*sysfsDir)
		return
	}

	if _, err := tea.NewProgram(newModel(*sysfsDir)).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func newModel(sysfsDir string) ui.UIModel {
	return ui.UIModel{
		SysfsDir: sysfsDir,
	}
}
