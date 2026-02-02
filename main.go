package main

import (
	"fmt"
	"os"

	"github.com/christian/usb-c/internal/parser"
	"github.com/christian/usb-c/internal/ui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <snapshot-directory>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s snapshots/charger-mac\n", os.Args[0])
		os.Exit(1)
	}

	snapshotDir := os.Args[1]

	// Check if directory exists
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: snapshot directory does not exist: %s\n", snapshotDir)
		os.Exit(1)
	}

	// Load snapshot
	ports, err := parser.LoadSnapshot(snapshotDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading snapshot: %v\n", err)
		os.Exit(1)
	}

	// Render output
	ui.RenderPorts(ports)
}
