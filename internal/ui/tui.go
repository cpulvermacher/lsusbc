package ui

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"

	"github.com/christian/usb-c/internal/model"
	"github.com/christian/usb-c/internal/parser"
)

var (
	inactive           = lipgloss.NewStyle().Foreground(lipgloss.Color("#4e4e4e"))
	selectedPort       = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#2f2f2f"))
	powerArrowCharging = lipgloss.NewStyle().Foreground(lipgloss.Color("#aad700"))

	powerModePd     = lipgloss.NewStyle().Foreground(lipgloss.Color("#91e500"))
	powerMode3000mA = lipgloss.NewStyle().Foreground(lipgloss.Color("#d0e440"))
	powerMode1500mA = lipgloss.NewStyle().Foreground(lipgloss.Color("#fae470"))
	powerModeUsb    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6f453d"))
)

type UIModel struct {
	TypecDir string

	ports        []model.Port
	selectedPort int
}

type RefreshTick time.Time

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return RefreshTick(t)
	})
}

func (m UIModel) Init() tea.Cmd {
	return nil
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.ports == nil {
		return refresh(m), doTick()
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			return refresh(m), nil
		case "j", "down":
			return moveSelection(m, +1), nil
		case "k", "up":
			return moveSelection(m, -1), nil
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case RefreshTick:
		return refresh(m), doTick()

	}

	return m, nil
}

func moveSelection(m UIModel, increment int) tea.Model {
	// no wrap-around
	if len(m.ports) == 0 || m.selectedPort+increment < 0 || m.selectedPort+increment >= len(m.ports) {
		return m
	}

	m.selectedPort += increment
	return m
}

func (m UIModel) View() string {
	if m.ports == nil {
		return "Loading..."
	}
	if len(m.ports) == 0 {
		return "No USB-C ports found"
	}

	var lines string
	for i, port := range m.ports {
		lines += renderPort(port, i == m.selectedPort) + " "
		if port.Partner == nil {
			lines += fmt.Sprintf("%s\n", inactive.Render("(no device connected)"))
		} else {
			lines += renderConnection(port)
		}
	}
	return lines
}

func refresh(m UIModel) UIModel {
	// Load ports
	ports, err := parser.LoadPorts(m.TypecDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ports: %v\n", err)
		os.Exit(1)
	}

	m.ports = ports
	return m
}

func renderPort(port model.Port, selected bool) string {
	if !selected {
		return port.Name
	} else {
		return selectedPort.Render(port.Name)
	}
}

// renderConnection renders a port-partner connection
func renderConnection(port model.Port) string {
	capabilities := formatCapabilities(port.Partner, port.PowerOperationMode)

	// Arrow direction based on power flow
	// If port is sink, it receives power (arrow points toward port)
	// If port is source, it provides power (arrow points toward device)
	var arrow string
	if port.PowerRole == "sink" {
		arrow = powerArrowCharging.Render("<==󱐋===")
	} else {
		arrow = "===󱐋==>"
	}

	// Handle single device vs multiple devices
	if len(port.Partner.USBDevices) == 0 {
		// No USB device info available - use generic name
		deviceName := getFriendlyDeviceName(port.Partner)
		return fmt.Sprintf("%s %s  %s\n", arrow, deviceName, capabilities)
	} else if len(port.Partner.USBDevices) == 1 {
		// Single USB device - show on same line
		device := port.Partner.USBDevices[0]
		deviceName := formatUSBDevice(device)
		return fmt.Sprintf("%s %s  %s\n", arrow, deviceName, capabilities)
	} else {
		// Multiple USB devices - show as tree
		t := tree.New().Enumerator(tree.RoundedEnumerator)
		for _, device := range port.Partner.USBDevices {
			deviceName := formatUSBDevice(device)
			t.Child(deviceName)
		}
		indentedTree := lipgloss.NewStyle().PaddingLeft(12).Render(t.String())
		return fmt.Sprintf("%s %s\n%s\n", arrow, capabilities, indentedTree)
	}
}

// formatUSBDevice formats a USB device name
func formatUSBDevice(device model.USBDevice) string {
	if device.Manufacturer != "" && device.Product != "" {
		return device.Manufacturer + " " + device.Product
	}
	if device.Product != "" {
		return device.Product
	}
	if device.Manufacturer != "" {
		return device.Manufacturer + " Device"
	}
	return "USB Device"
}

// getFriendlyDeviceName generates a friendly device description when USB device info is not available
func getFriendlyDeviceName(partner *model.Partner) string {
	// Priority 1: Alternate mode description(s)
	if len(partner.AlternateModes) > 0 {
		// If there's only one alternate mode, show it directly
		if len(partner.AlternateModes) == 1 {
			return partner.AlternateModes[0].Description + " Device"
		}
		// If multiple alternate modes, concatenate them
		var modes string
		for i, mode := range partner.AlternateModes {
			if i > 0 {
				modes += ", "
			}
			modes += mode.Description
		}
		return modes + " Device"
	}

	// Priority 2: Charger (device role + sink power role)
	if partner.DataRole == "device" && partner.PowerRole == "sink" {
		return "Charger"
	}

	// Priority 3: Phone/Device (device role + source power role)
	if partner.DataRole == "device" && partner.PowerRole == "source" {
		return "Phone/Device"
	}

	// Priority 4: Audio accessory
	if partner.AccessoryMode == "audio" {
		return "Audio Accessory"
	}

	// Fallback
	return "USB Device"
}

// formatCapabilities formats power capabilities based on power operation mode
func formatCapabilities(partner *model.Partner, powerOperationMode string) string {
	// Use power_operation_mode to decide what to show
	switch powerOperationMode {
	case "default":
		return powerModeUsb.Render("[USB]")
	case "1.5A":
		return powerMode1500mA.Render("[1.5A]")
	case "3.0A":
		return powerMode3000mA.Render("[3A]")
	case "usb_power_delivery":
		// Show PD version only
		if partner.PDRevision != "" && partner.PDRevision != "0.0" {
			return powerModePd.Render("[PD " + partner.PDRevision + "]")
		}
		return powerModePd.Render("[PD]")
	default:
		return ""
	}
}
