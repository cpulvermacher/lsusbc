package main

import (
	"fmt"
	"os"

	"github.com/christian/usb-c/internal/parser"
	"github.com/christian/usb-c/internal/ui"
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

	// Load ports
	ports, err := parser.LoadPorts(typecDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ports: %v\n", err)
		os.Exit(1)
	}

	// Render output
	ui.RenderPorts(ports)
}
