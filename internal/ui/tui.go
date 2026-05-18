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

	statusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#000000")).
			Foreground(lipgloss.Color("#9e9e9e"))

	batteryCharging = lipgloss.NewStyle().Foreground(lipgloss.Color("#91e500"))
	batteryNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0e400"))
	batteryLow      = lipgloss.NewStyle().Foreground(lipgloss.Color("#fe7400"))
	batteryCritical = lipgloss.NewStyle().Foreground(lipgloss.Color("#fe4000"))
)

const startIndent = ""

type itemKind int

const (
	kindPort itemKind = iota
	kindUSBDevice
)

type listItem struct {
	id      string // stable identifier: port name or USB device ID
	kind    itemKind
	portIdx int
	device  *model.USBDevice // nil for kindPort
}

type UIModel struct {
	// user/env controlled
	sysfsDir     string
	termWidth    int
	termHeight   int
	selectedID   string // stable identifier for selected item (port name or USB device ID)
	selectedItem int    // index of selectedID in items, kept in sync

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
		case "r":
			return refresh(m), nil
		case "j", "down":
			return moveSelection(m, +1), nil
		case "k", "up":
			return moveSelection(m, -1), nil
		case "q", "esc":
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
		items = append(items, listItem{id: ports[i].Name, kind: kindPort, portIdx: i})
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
	items = append(items, listItem{id: dev.DeviceID, kind: kindUSBDevice, portIdx: portIdx, device: dev})
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
	m.selectedID = m.items[m.selectedItem].id
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

	// 1: list panel
	var listContent string
	if len(m.ports) > 0 {
		listContent += " USB-C Ports\n"
	}
	itemIdx := 0
	for _, port := range m.ports {
		listContent += renderPort(port, itemIdx == m.selectedItem) + " "
		if port.Partner == nil {
			listContent += inactiveStyle.Render("(no device connected)") + "\n"
		} else {
			listContent += renderConnection(port) + "\n"
		}
		itemIdx++
		if port.Partner != nil {
			var treeLines []string
			treeLines, itemIdx = renderUSBDeviceTree(port.Partner.USBDevices, itemIdx, m.selectedItem, startIndent)
			for _, line := range treeLines {
				listContent += line + "\n"
			}
		}
	}
	if len(m.standaloneUSBDevices) > 0 {
		if len(m.ports) > 0 {
			listContent += "\n"
		}
		listContent += " Other USB Devices\n"
		var treeLines []string
		treeLines, _ = renderUSBDeviceTree(m.standaloneUSBDevices, itemIdx, m.selectedItem, startIndent)
		for _, line := range treeLines {
			listContent += line + "\n"
		}
	}

	// 2: details panel
	detailsContent := ""
	if len(m.items) > 0 {
		selected := m.items[m.selectedItem]
		if selected.kind == kindPort {
			detailsContent = renderPortDetails(m.ports[selected.portIdx])
		} else {
			detailsContent = renderUSBDevicePanel(*selected.device)
		}

	}

	content := buildPanelLayout(m, listContent, detailsContent)

	// 3: status bar
	bar := renderStatusBar(m)
	content = lipgloss.JoinVertical(lipgloss.Left, content, bar)

	view.SetContent(content)
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
	m.selectedItem = resolveSelection(m.items, m.selectedID, m.selectedItem)
	if len(m.items) > 0 {
		m.selectedID = m.items[m.selectedItem].id
	}
	m.battery = parser.LoadBatteryInfo(m.sysfsDir)
	return m
}

// resolveSelection finds the index of selectedID in items.
// Falls back to clamping prevIdx to the valid range if ID is not found.
func resolveSelection(items []listItem, selectedID string, prevIdx int) int {
	if selectedID != "" {
		for i, item := range items {
			if item.id == selectedID {
				return i
			}
		}
	}
	return max(0, min(prevIdx, len(items)-1))
}

// adjusts panel orientation and size based on terminal size and list width
func buildPanelLayout(m UIModel, listContent string, detailsContent string) string {
	detailsStyleBase := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#8e8e8e")).
		Margin(0, 1).
		Padding(0, 1)

	if m.termWidth < 70 {
		// narrow window => vertical layout
		listStyle := lipgloss.NewStyle().Width(m.termWidth)
		detailsStyle := detailsStyleBase.Width(m.termWidth)

		listPanel := listStyle.Render(listContent)
		detailsPanel := detailsStyle.Render(detailsContent)

		content := lipgloss.JoinVertical(lipgloss.Top, listPanel, detailsPanel)
		// reserve status bar line
		return lipgloss.NewStyle().Width(m.termWidth).Height(m.termHeight - 1).Render(content)
	} else {
		// horizontal layout
		actualListWidth := lipgloss.Width(listContent)
		listWidth := min(actualListWidth, m.termWidth*4/7)

		listStyle := lipgloss.NewStyle().Width(listWidth).Height(m.termHeight - 1)
		detailsStyle := detailsStyleBase.Width(m.termWidth - listWidth).Height(m.termHeight - 1)

		listPanel := listStyle.Render(listContent)
		detailsPanel := detailsStyle.Render(detailsContent)

		return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, detailsPanel)
	}
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
		const gradSteps = 4
		var corner, childIndent string
		if isLast {
			corner = "╰"
			childIndent = indent + strings.Repeat(" ", gradSteps+3)
		} else {
			corner = "├"
			childIndent = indent + "│" + strings.Repeat(" ", gradSteps+2)
		}

		connector := indent + corner + gradientConnector(devices[i], gradSteps) + " "
		name := formatUSBDevice(devices[i])
		var line string
		if idx == selectedItem {
			line = ">" + connector + selectedStyle.Render(name)
		} else {
			line = " " + connector + name
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
		arrow = "======>"
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
	if device.MaxPower != "" {
		s += fmt.Sprintf("Max Power: %s\n", device.MaxPower)
	}
	if len(device.Drivers) > 0 {
		s += fmt.Sprintf("Driver: %s\n", strings.Join(device.Drivers, ", "))
	}
	return s
}

// ListPorts loads and prints ports and standalone USB devices to stdout.
// When verbose is true, full detail panels are also printed below the tree overview.
func ListPorts(typecDir string, verbose bool) {
	ports, err := parser.LoadPorts(typecDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ports: %v\n", err)
		os.Exit(1)
	}

	standaloneUSBDevices := parser.LoadStandaloneUSBDevices(typecDir, ports)

	// 1: tree overview
	if len(ports) > 0 {
		fmt.Println(" USB-C Ports")
	}
	for _, port := range ports {
		fmt.Print(renderPort(port, false) + " ")
		if port.Partner == nil {
			fmt.Print(inactiveStyle.Render("(no device connected)"))
		} else {
			fmt.Print(renderConnection(port))
		}
		fmt.Println()
		if port.Partner != nil {
			lines, _ := renderUSBDeviceTree(port.Partner.USBDevices, 0, -1, startIndent)
			for _, line := range lines {
				fmt.Println(line)
			}
		}
	}
	if len(standaloneUSBDevices) > 0 {
		if len(ports) > 0 {
			fmt.Println()
		}
		fmt.Println(" Other USB Devices")
		lines, _ := renderUSBDeviceTree(standaloneUSBDevices, 0, -1, startIndent)
		for _, line := range lines {
			fmt.Println(line)
		}
	}

	if !verbose {
		return
	}

	// 2: separator
	fmt.Println()
	fmt.Println(strings.Repeat("─", 48))

	// 3: detail blocks
	items := buildItemList(ports, standaloneUSBDevices)
	for _, item := range items {
		fmt.Println()
		if item.kind == kindPort {
			fmt.Print(renderPortDetails(ports[item.portIdx]))
		} else {
			fmt.Print(renderUSBDevicePanel(*item.device))
		}
	}
}
