// Package ui implements the terminal user interface
package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cpulvermacher/lsusbc/internal/model"
	"github.com/cpulvermacher/lsusbc/internal/parser"
)

var (
	inactiveStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#4e4e4e"))
	selectedStyle      = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#2f2f2f"))
	powerArrowCharging = lipgloss.NewStyle().Foreground(lipgloss.Color("#aad700"))

	portListStyle = lipgloss.NewStyle().Width(40)
	detailsStyle  = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#8e8e8e")).
			Padding(1, 2).
			MarginLeft(10).
			Width(70)

	helpText  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6e6e6e"))
	statusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#000000")).
			Foreground(lipgloss.Color("#9e9e9e"))

	batteryCharging = lipgloss.NewStyle().Foreground(lipgloss.Color("#91e500"))
	batteryNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color("#d0e440"))
	batteryLow      = lipgloss.NewStyle().Foreground(lipgloss.Color("#fec400"))
	batteryCritical = lipgloss.NewStyle().Foreground(lipgloss.Color("#fe8000"))
)

type itemKind int

const (
	kindPort itemKind = iota
	kindUSBDevice
)

type listItem struct {
	kind    itemKind
	portIdx int
	device  *model.USBDevice // nil for kindPort
}

type UIModel struct {
	// user/env controlled
	sysfsDir       string
	termWidth      int
	termHeight     int
	selectedItem   int
	showingDetails bool

	// displayed items
	ports                []model.Port
	standaloneUSBDevices []model.USBDevice
	items                []listItem
	battery              *model.BatteryInfo
}

func InitializeModel(sysfsDir string) UIModel {
	return UIModel{
		sysfsDir:   sysfsDir,
		termWidth:  80,
		termHeight: 24,
	}
}

type RefreshTick time.Time

func doTick() tea.Cmd {
	const refreshInterval = 1 * time.Second
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return RefreshTick(t)
	})
}

func (m UIModel) Init() tea.Cmd {
	return nil
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "space", "enter":
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
		case "ctrl+z":
			return m, tea.Suspend
		}

	case RefreshTick:
		return refresh(m), doTick()

	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
	}

	if m.ports == nil {
		return refresh(m), doTick()
	}

	return m, nil
}

func buildItemList(ports []model.Port, standaloneUSBDevices []model.USBDevice) []listItem {
	var items []listItem
	for i := range ports {
		items = append(items, listItem{kind: kindPort, portIdx: i})
		if ports[i].Partner != nil {
			for j := range ports[i].Partner.USBDevices {
				items = appendDeviceItems(items, i, &ports[i].Partner.USBDevices[j])
			}
		}
	}
	for i := range standaloneUSBDevices {
		items = appendDeviceItems(items, -1, &standaloneUSBDevices[i])
	}
	return items
}

func appendDeviceItems(items []listItem, portIdx int, dev *model.USBDevice) []listItem {
	items = append(items, listItem{kind: kindUSBDevice, portIdx: portIdx, device: dev})
	for j := range dev.USBDevices {
		items = appendDeviceItems(items, portIdx, &dev.USBDevices[j])
	}
	return items
}

func moveSelection(m UIModel, increment int) tea.Model {
	// no wrap-around
	if len(m.items) == 0 || m.selectedItem+increment < 0 || m.selectedItem+increment >= len(m.items) {
		return m
	}

	m.selectedItem += increment
	return m
}

func (m UIModel) View() tea.View {
	var view tea.View
	view.AltScreen = true

	if m.ports == nil {
		view.SetContent("Loading...")
		return view
	} else if len(m.ports) == 0 && len(m.standaloneUSBDevices) == 0 {
		view.SetContent("No USB devices found")
		return view
	}

	// 1: port list
	var ports string
	itemIdx := 0
	for _, port := range m.ports {
		ports += renderPort(port, itemIdx == m.selectedItem) + " "
		if port.Partner == nil {
			ports += inactiveStyle.Render("(no device connected)") + "\n"
		} else {
			ports += renderConnection(port) + "\n"
		}
		itemIdx++
		if port.Partner != nil {
			var treeLines []string
			treeLines, itemIdx = renderUSBDeviceTree(port.Partner.USBDevices, itemIdx, m.selectedItem, "    ")
			for _, line := range treeLines {
				ports += line + "\n"
			}
		}
	}
	if len(m.standaloneUSBDevices) > 0 {
		if len(m.ports) > 0 {
			ports += "\n"
		}
		ports += " Other USB Devices\n"
		var treeLines []string
		treeLines, _ = renderUSBDeviceTree(m.standaloneUSBDevices, itemIdx, m.selectedItem, "    ")
		for _, line := range treeLines {
			ports += line + "\n"
		}
	}
	lines := portListStyle.Render(ports)

	// 2: details
	if m.showingDetails && len(m.items) > 0 {
		selected := m.items[m.selectedItem]
		var details string
		if selected.kind == kindPort {
			details = renderPortDetails(m.ports[selected.portIdx])
		} else {
			details = renderUSBDevicePanel(*selected.device)
		}
		instruction := helpText.Render("\nPress Escape or q to close")
		popup := detailsStyle.Render(details + instruction)

		lines = lipgloss.JoinHorizontal(lipgloss.Top, lines, popup)
	}

	// set fixed height for 1+2
	if m.termHeight > 1 {
		lines = lipgloss.NewStyle().Width(m.termWidth).Height(m.termHeight - 1).Render(lines)
	}

	// 3: status bar
	bar := renderStatusBar(m)
	lines = lipgloss.JoinVertical(lipgloss.Left, lines, bar)

	view.SetContent(lines)
	return view
}

func renderStatusBar(m UIModel) string {
	var parts []string

	if m.battery != nil && m.battery.CapacityLevel != "" && m.battery.CapacityLevel != "Unknown" {
		bat := fmt.Sprintf("Battery: %d%%", m.battery.Capacity)
		switch m.battery.Status {
		case "Discharging":
			switch m.battery.CapacityLevel {
			case "Normal":
				bat = batteryNormal.Render(bat)
			case "Low":
				bat = batteryLow.Render(bat)
			case "Critical":
				bat = batteryCritical.Render(bat)
			}
		case "Unknown":
			break // no formatting
		default:
			bat = batteryCharging.Render(bat)
		}
		parts = append(parts, bat)
	}

	if m.sysfsDir != "/sys" {
		parts = append(parts, "sysfs: "+m.sysfsDir)
	}

	text := strings.Join(parts, "  |  ")
	if m.termWidth > 0 {
		text = statusBar.Width(m.termWidth).Render(text)
	} else {
		text = statusBar.Render(text)
	}
	return text
}

func refresh(m UIModel) UIModel {
	ports, err := parser.LoadPorts(m.sysfsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ports: %v\n", err)
		os.Exit(1)
	}

	m.ports = ports
	m.standaloneUSBDevices = parser.LoadStandaloneUSBDevices(m.sysfsDir, ports)
	m.items = buildItemList(ports, m.standaloneUSBDevices)
	if m.selectedItem >= len(m.items) {
		m.selectedItem = max(0, len(m.items)-1)
	}
	m.battery = parser.LoadBatteryInfo(m.sysfsDir)
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
		return ">" + selectedStyle.Render(label)
	}
}

// renderUSBDeviceTree renders USB devices as a tree with selection highlighting.
// Returns rendered lines and the next item index after consuming all devices.
func renderUSBDeviceTree(devices []model.USBDevice, startIdx int, selectedItem int, indent string) ([]string, int) {
	var lines []string
	idx := startIdx
	for i := range devices {
		isLast := i == len(devices)-1
		var connector, childIndent string
		if isLast {
			connector = "╰─ "
			childIndent = indent + "   "
		} else {
			connector = "├─ "
			childIndent = indent + "│  "
		}

		content := indent + connector + formatUSBDevice(devices[i]) + formatUsbSpeedInline(devices[i])
		var line string
		if idx == selectedItem {
			line = ">" + selectedStyle.Render(content)
		} else {
			line = " " + content
		}
		lines = append(lines, line)
		idx++

		if len(devices[i].USBDevices) > 0 {
			var childLines []string
			childLines, idx = renderUSBDeviceTree(devices[i].USBDevices, idx, selectedItem, childIndent)
			lines = append(lines, childLines...)
		}
	}
	return lines, idx
}

// renderConnection renders a port-partner connection
func renderConnection(port model.Port) string {
	capabilities := formatPowerModeInline(port.Partner.PowerDelivery, port.PowerOperationMode)

	var arrow string
	if port.PowerRole == "sink" {
		arrow = powerArrowCharging.Render("<==󱐋===")
	} else {
		arrow = "===󱐋==>"
	}

	deviceName := getFriendlyDeviceName(port.Partner)
	return fmt.Sprintf("%s %s  %s", arrow, deviceName, capabilities)
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
				content += formatAlternateMode(mode)
			}
		}
		content += "\n"
	}

	// Partner info
	if port.Partner == nil {
		content += fmt.Sprintf("Connected Device: %s\n", inactiveStyle.Render("(no device connected)"))
	} else {
		partner := port.Partner
		content += fmt.Sprintf("Connected Device: %s\n", partner.Name)
		if pd := partner.PowerDelivery; pd != nil {
			content += fmt.Sprintf("  Power: USB Power Delivery %s\n", pd.Revision)
			if pd.ACPowered {
				content += "  Power Source: AC Powered\n"
			}

			// Source capabilities
			if len(pd.SourceCapabilities) > 0 {
				content += fmt.Sprintf("  Charger Capabilities:  %dW\n", MaxWatts(pd.SourceCapabilities))
				for i, cap := range pd.SourceCapabilities {
					content += fmt.Sprintf("    [%d] %s @ %s\n", i, FormatVoltage(cap), FormatCurrent(cap))
				}
				content += "\n"
			}

			// Sink capabilities
			if len(pd.SinkCapabilities) > 0 {
				content += "  Sink Capabilities:\n"
				for i, cap := range pd.SinkCapabilities {
					content += fmt.Sprintf("    [%d] %s @ %s\n", i, FormatVoltage(cap), FormatCurrent(cap))
				}
				content += "\n"
			}
		} else {
			switch port.PowerOperationMode {
			case "default":
				content += "  Power: Default USB Power (5V, ≤500mA/900mA, ≤2.5W/4.5W)\n\n"
			case "1.5A":
				content += "  Power: USB Type-C Current (5V @ 1.5A, 7.5W)\n\n"
			case "3.0A":
				content += "  Power: USB Type-C Current (5V @ 3A, 15W)\n\n"
			}
		}
		if partner.AccessoryMode != "none" {
			content += fmt.Sprintf("  Accessory Mode: %s\n\n", partner.AccessoryMode)
		}

		// Alternate modes
		if len(partner.AlternateModes) > 0 {
			content += "  Alternate Modes:\n"
			for _, mode := range partner.AlternateModes {
				content += formatAlternateMode(mode)
			}
			content += "\n"
		}

	}

	return content
}

func renderUSBDevicePanel(device model.USBDevice) string {
	var s string
	s += fmt.Sprintf("USB Device: %s\n", device.DeviceID)
	if device.Manufacturer != "" {
		s += fmt.Sprintf("Manufacturer: %s\n", device.Manufacturer)
	}
	if device.Product != "" {
		s += fmt.Sprintf("Product: %s\n", device.Product)
	}
	if device.Serial != "" {
		s += fmt.Sprintf("Serial: %s\n", device.Serial)
	}
	if device.IDVendor != "" {
		s += fmt.Sprintf("Vendor ID: %s\n", device.IDVendor)
	}
	if device.IDProduct != "" {
		s += fmt.Sprintf("Product ID: %s\n", device.IDProduct)
	}
	if device.Version != "" {
		s += fmt.Sprintf("USB Version: %s\n", device.Version)
	}
	if device.Speed != "" {
		s += fmt.Sprintf("Speed: %s\n", formatUsbSpeed(device))
	}
	return s
}

// ListPorts loads and prints details for all ports and standalone USB devices to stdout.
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

	standaloneUSBDevices := parser.LoadStandaloneUSBDevices(typecDir, ports)
	if len(standaloneUSBDevices) > 0 {
		fmt.Println("Other USB Devices")
		lines, _ := renderUSBDeviceTree(standaloneUSBDevices, 0, -1, "")
		for _, line := range lines {
			fmt.Println(line)
		}
	}
}
