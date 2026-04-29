package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"

	"github.com/cpulvermacher/lsusbc/internal/model"
	"github.com/cpulvermacher/lsusbc/internal/parser"
)

var (
	inactive           = lipgloss.NewStyle().Foreground(lipgloss.Color("#4e4e4e"))
	selectedPort       = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#2f2f2f"))
	powerArrowCharging = lipgloss.NewStyle().Foreground(lipgloss.Color("#aad700"))

	powerModePd     = lipgloss.NewStyle().Foreground(lipgloss.Color("#91e500"))
	powerMode3000mA = lipgloss.NewStyle().Foreground(lipgloss.Color("#d0e440"))
	powerMode1500mA = lipgloss.NewStyle().Foreground(lipgloss.Color("#fae470"))
	powerModeUsb    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6f453d"))

	popupStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#8e8e8e")).
			Padding(1, 2).
			MarginLeft(10).
			Width(70)

	helpText = lipgloss.NewStyle().Foreground(lipgloss.Color("#6e6e6e"))
)

type UIModel struct {
	TypecDir string

	ports          []model.Port
	selectedPort   int
	showingDetails bool
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
		case " ", "enter":
			m.showingDetails = true
			return m, nil
		case "r":
			return refresh(m), nil
		case "j", "down":
			return moveSelection(m, +1), nil
		case "k", "up":
			return moveSelection(m, -1), nil
		case "q", "esc":
			if m.showingDetails {
				m.showingDetails = false
				return m, nil
			}
			return m, tea.Quit
		case "ctrl+c":
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

	if m.showingDetails && len(m.ports) > 0 {
		return renderPopupOverlay(lines, m.ports[m.selectedPort])
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

func portDisplayName(name string) string {
	if strings.HasPrefix(name, "port") {
		var i int
		if _, err := fmt.Sscanf(name[4:], "%d", &i); err == nil {
			return fmt.Sprintf("Port %d", i)
		}
	}
	return name
}

func renderPort(port model.Port, selected bool) string {
	label := portDisplayName(port.Name)
	if !selected {
		return " " + label
	} else {
		return ">" + selectedPort.Render(label)
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

	devices := port.Partner.USBDevices
	if len(devices) == 0 {
		deviceName := getFriendlyDeviceName(&port, port.Partner)
		return fmt.Sprintf("%s %s  %s\n", arrow, deviceName, capabilities)
	} else if len(devices) == 1 && len(devices[0].USBDevices) == 0 {
		return fmt.Sprintf("%s %s  %s\n", arrow, formatUSBDevice(devices[0]), capabilities)
	} else {
		t := tree.New().Enumerator(tree.RoundedEnumerator)
		for _, device := range devices {
			t.Child(usbDeviceTree(device))
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

func usbDeviceTree(device model.USBDevice) *tree.Tree {
	t := tree.New().Root(formatUSBDevice(device))
	for _, sub := range device.USBDevices {
		t.Child(usbDeviceTree(sub))
	}
	return t
}

// getFriendlyDeviceName generates a friendly device description when USB device info is not available
func getFriendlyDeviceName(port *model.Port, partner *model.Partner) string {
	// Priority 1: Alternate mode description(s)
	if len(partner.AlternateModes) > 0 {
		// concatenate them
		var modesBuilder strings.Builder
		for _, mode := range partner.AlternateModes {
			if mode.Description == "" {
				continue
			}
			if modesBuilder.Len() > 0 {
				modesBuilder.WriteString(", ")
			}
			modesBuilder.WriteString(mode.Description)
		}
		modes := modesBuilder.String()
		if modes != "" {
			return modes + " Device"
		}
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
		label := "PD"
		if partner.PDRevision != "" && partner.PDRevision != "0.0" {
			label = "PD " + partner.PDRevision
		}
		if watts := model.MaxWatts(partner.SourceCapabilities); watts > 0 {
			label = fmt.Sprintf("%s, %dW", label, watts)
		}
		if partner.ACPowered {
			label += ", AC"
		}
		return powerModePd.Render("[" + label + "]")
	default:
		return ""
	}
}

func renderUSBDeviceDetails(device model.USBDevice, indent string) string {
	var s string
	s += fmt.Sprintf("%s%s\n", indent, device.DeviceID)
	if device.Manufacturer != "" {
		s += fmt.Sprintf("%s  Manufacturer: %s\n", indent, device.Manufacturer)
	}
	if device.Product != "" {
		s += fmt.Sprintf("%s  Product: %s\n", indent, device.Product)
	}
	if device.Serial != "" {
		s += fmt.Sprintf("%s  Serial: %s\n", indent, device.Serial)
	}
	if device.IDVendor != "" {
		s += fmt.Sprintf("%s  Vendor ID: %s\n", indent, device.IDVendor)
	}
	if device.IDProduct != "" {
		s += fmt.Sprintf("%s  Product ID: %s\n", indent, device.IDProduct)
	}
	if device.Version != "" {
		s += fmt.Sprintf("%s  USB Version: %s\n", indent, device.Version)
	}
	if device.Speed != "" {
		s += fmt.Sprintf("%s  Speed: %s Mb/s\n", indent, device.Speed)
	}
	for _, sub := range device.USBDevices {
		s += renderUSBDeviceDetails(sub, indent+"  ")
	}
	return s
}

// renderPortDetails formats all Port model fields for display
func renderPortDetails(port model.Port) string {
	var content string

	// Port basic info
	content += fmt.Sprintf("Port: %s\n", portDisplayName(port.Name))
	content += fmt.Sprintf("Data Role: %s\n", port.DataRole)
	content += fmt.Sprintf("Power Role: %s\n", port.PowerRole)
	content += fmt.Sprintf("Power Operation Mode: %s\n\n", port.PowerOperationMode)

	// Cable info
	if port.Cable != nil {
		cable := port.Cable
		content += "Cable:\n"
		if cable.Type != "" && cable.Type != "undefined" {
			content += fmt.Sprintf("  Type: %s\n", cable.Type)
		}
		if cable.PlugType != "" {
			content += fmt.Sprintf("  Plug: %s\n", cable.PlugType)
		}
		if len(cable.AlternateModes) > 0 {
			content += "  Alternate Modes:\n"
			for _, mode := range cable.AlternateModes {
				marker := " "
				if mode.Active == "yes" {
					marker = "*"
				}
				content += fmt.Sprintf("   %s[%d] %s (SVID: %s, VDO: %s)\n", marker, mode.Index, mode.Description, mode.SVID, mode.VDO)
			}
		}
		content += "\n"
	}

	// Partner info
	if port.Partner == nil {
		content += fmt.Sprintf("Connected Device: %s\n", inactive.Render("(no device connected)"))
	} else {
		partner := port.Partner
		content += fmt.Sprintf("Connected Device: %s\n", partner.Name)
		content += fmt.Sprintf("  PD Revision: %s\n", partner.PDRevision)
		if partner.ACPowered {
			content += "  Power Source: AC Powered\n"
		}
		if partner.AccessoryMode != "none" {
			content += fmt.Sprintf("  Accessory Mode: %s\n\n", partner.AccessoryMode)
		}

		// Source capabilities
		if len(partner.SourceCapabilities) > 0 {
			content += fmt.Sprintf("  Charger Capabilities:  %dW\n", model.MaxWatts(partner.SourceCapabilities))
			for i, cap := range partner.SourceCapabilities {
				content += fmt.Sprintf("    [%d] %s @ %s\n", i, cap.FormatVoltage(), cap.FormatCurrent())
			}
			content += "\n"
		}

		// Sink capabilities
		if len(partner.SinkCapabilities) > 0 {
			content += "  Sink Capabilities:\n"
			for i, cap := range partner.SinkCapabilities {
				content += fmt.Sprintf("    [%d] %s @ %s\n", i, cap.FormatVoltage(), cap.FormatCurrent())
			}
			content += "\n"
		}

		// Alternate modes
		if len(partner.AlternateModes) > 0 {
			content += "  Alternate Modes:\n"
			for _, mode := range partner.AlternateModes {
				marker := " "
				if mode.Active == "yes" {
					marker = "*"
				}
				content += fmt.Sprintf("   %s[%d] %s (SVID: %s, VDO: %s)\n", marker, mode.Index, mode.Description, mode.SVID, mode.VDO)
			}
			content += "\n"
		}

		// USB devices
		if len(partner.USBDevices) > 0 {
			content += "  USB Devices:\n"
			for _, device := range partner.USBDevices {
				content += renderUSBDeviceDetails(device, "    ")
			}
		}
	}

	return content
}

// ListPorts loads and prints details for all ports to stdout.
func ListPorts(typecDir string) {
	ports, err := parser.LoadPorts(typecDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ports: %v\n", err)
		os.Exit(1)
	}
	for _, port := range ports {
		fmt.Print(renderPortDetails(port))
		fmt.Println()
	}
}

// renderPopupOverlay renders the port details as a popup overlay
func renderPopupOverlay(background string, port model.Port) string {
	details := renderPortDetails(port)

	instruction := helpText.Render("\nPress Escape or q to close")

	popup := popupStyle.Render(details + instruction)

	return lipgloss.JoinHorizontal(lipgloss.Top, background, popup)
}
