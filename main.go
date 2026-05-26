package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/cpulvermacher/lsusbc/internal/ui"
)

var version = "dev"

func main() {
	sysfsDir := flag.String("d", "/sys", "sysfs directory")
	listFlag := flag.Bool("l", false, "list devices and exit")
	verboseFlag := flag.Bool("v", false, "include full device details (implies -l)")
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		return
	}

	if flag.NArg() > 0 {
		*sysfsDir = flag.Arg(0)
	}

	// Check if directory exists
	if _, err := os.Stat(*sysfsDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: directory does not exist: %s\n", *sysfsDir)
		os.Exit(1)
	}

	if *listFlag || *verboseFlag || !term.IsTerminal(os.Stdout.Fd()) {
		ui.ListPorts(*sysfsDir, *verboseFlag)
		return
	}

	if _, err := tea.NewProgram(ui.InitializeModel(*sysfsDir)).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
