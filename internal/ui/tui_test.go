package ui

import (
	"strings"
	"testing"

	"github.com/cpulvermacher/lsusbc/internal/model"
)

// buildItemList

func TestBuildItemList_Empty(t *testing.T) {
	items := buildItemList(nil, nil)
	if len(items) != 0 {
		t.Errorf("expected empty list, got %d items", len(items))
	}
}

func TestBuildItemList_PortNoPartner(t *testing.T) {
	ports := []model.Port{{Name: "port0"}}
	items := buildItemList(ports, nil)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].kind != kindPort || items[0].id != "port0" {
		t.Errorf("unexpected item: %+v", items[0])
	}
}

func TestBuildItemList_PortWithUSBDevices(t *testing.T) {
	ports := []model.Port{
		{
			Name: "port0",
			Partner: &model.Partner{
				USBDevices: []model.USBDevice{
					{DeviceID: "1-4"},
					{DeviceID: "1-5"},
				},
			},
		},
	}
	items := buildItemList(ports, nil)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].kind != kindPort || items[0].id != "port0" {
		t.Errorf("items[0] should be port0, got %+v", items[0])
	}
	if items[1].kind != kindUSBDevice || items[1].id != "1-4" || items[1].portIdx != 0 {
		t.Errorf("items[1] should be USB device 1-4 on port 0, got %+v", items[1])
	}
	if items[2].kind != kindUSBDevice || items[2].id != "1-5" || items[2].portIdx != 0 {
		t.Errorf("items[2] should be USB device 1-5 on port 0, got %+v", items[2])
	}
}

func TestBuildItemList_NestedUSBDevices(t *testing.T) {
	ports := []model.Port{
		{
			Name: "port0",
			Partner: &model.Partner{
				USBDevices: []model.USBDevice{
					{
						DeviceID: "1-4",
						USBDevices: []model.USBDevice{
							{DeviceID: "1-4.1"},
						},
					},
				},
			},
		},
	}
	items := buildItemList(ports, nil)
	if len(items) != 3 {
		t.Fatalf("expected 3 items (port, hub, child), got %d", len(items))
	}
	if items[2].id != "1-4.1" {
		t.Errorf("items[2] should be nested device 1-4.1, got %+v", items[2])
	}
}

func TestBuildItemList_StandaloneDevicesHavePortIdxMinus1(t *testing.T) {
	standalone := []model.USBDevice{{DeviceID: "2-1"}, {DeviceID: "2-2"}}
	items := buildItemList(nil, standalone)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	for _, item := range items {
		if item.portIdx != -1 {
			t.Errorf("standalone device should have portIdx -1, got %d for %q", item.portIdx, item.id)
		}
	}
}

func TestBuildItemList_OrderAcrossPortsAndStandalone(t *testing.T) {
	ports := []model.Port{
		{Name: "port0"},
		{Name: "port1", Partner: &model.Partner{USBDevices: []model.USBDevice{{DeviceID: "1-4"}}}},
	}
	standalone := []model.USBDevice{{DeviceID: "2-1"}}
	items := buildItemList(ports, standalone)
	// port0, port1, 1-4 (under port1), 2-1 (standalone)
	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(items))
	}
	wantIDs := []string{"port0", "port1", "1-4", "2-1"}
	for i, want := range wantIDs {
		if items[i].id != want {
			t.Errorf("items[%d].id = %q, want %q", i, items[i].id, want)
		}
	}
}

// resolveSelection

func TestResolveSelection_IDFound(t *testing.T) {
	items := []listItem{{id: "port0"}, {id: "1-4"}, {id: "port1"}}
	if got := resolveSelection(items, "1-4", 0); got != 1 {
		t.Errorf("expected 1, got %d", got)
	}
}

func TestResolveSelection_IDNotFound_ClampsToBound(t *testing.T) {
	items := []listItem{{id: "port0"}, {id: "port1"}}
	if got := resolveSelection(items, "gone", 99); got != 1 {
		t.Errorf("expected clamped index 1, got %d", got)
	}
}

func TestResolveSelection_IDNotFound_PrevIdxInRange(t *testing.T) {
	items := []listItem{{id: "port0"}, {id: "port1"}, {id: "port2"}}
	if got := resolveSelection(items, "gone", 1); got != 1 {
		t.Errorf("expected prevIdx 1, got %d", got)
	}
}

func TestResolveSelection_EmptyItems(t *testing.T) {
	if got := resolveSelection(nil, "port0", 0); got != 0 {
		t.Errorf("expected 0 for empty list, got %d", got)
	}
}

func TestResolveSelection_EmptyID_FallsBackToPrevIdx(t *testing.T) {
	items := []listItem{{id: "port0"}, {id: "port1"}}
	if got := resolveSelection(items, "", 1); got != 1 {
		t.Errorf("expected prevIdx 1, got %d", got)
	}
}

// getFriendlyDeviceName

func TestGetFriendlyDeviceName_AlternateMode(t *testing.T) {
	partner := &model.Partner{
		AlternateModes: []model.AlternateMode{{Description: "DisplayPort"}},
	}
	if got := getFriendlyDeviceName(partner); got != "DisplayPort Device" {
		t.Errorf("got %q, want %q", got, "DisplayPort Device")
	}
}

func TestGetFriendlyDeviceName_MultipleAlternateModes(t *testing.T) {
	partner := &model.Partner{
		AlternateModes: []model.AlternateMode{
			{Description: "DisplayPort"},
			{Description: "Thunderbolt"},
		},
	}
	if got := getFriendlyDeviceName(partner); got != "DisplayPort, Thunderbolt Device" {
		t.Errorf("got %q, want %q", got, "DisplayPort, Thunderbolt Device")
	}
}

func TestGetFriendlyDeviceName_AlternateModeEmptyDescriptionSkipped(t *testing.T) {
	// empty description should not count; fall through to audio
	partner := &model.Partner{
		AlternateModes: []model.AlternateMode{{Description: ""}},
		AccessoryMode:  "audio",
	}
	if got := getFriendlyDeviceName(partner); got != "Audio Accessory" {
		t.Errorf("got %q, want %q", got, "Audio Accessory")
	}
}

func TestGetFriendlyDeviceName_AudioAccessory(t *testing.T) {
	partner := &model.Partner{AccessoryMode: "audio"}
	if got := getFriendlyDeviceName(partner); got != "Audio Accessory" {
		t.Errorf("got %q, want %q", got, "Audio Accessory")
	}
}

func TestGetFriendlyDeviceName_Fallback(t *testing.T) {
	partner := &model.Partner{}
	if got := getFriendlyDeviceName(partner); got != "USB Device" {
		t.Errorf("got %q, want %q", got, "USB Device")
	}
}

// renderUSBDeviceTree

func TestRenderUSBDeviceTree_Empty(t *testing.T) {
	lines, idx := renderUSBDeviceTree(nil, 0, -1, "")
	if len(lines) != 0 || idx != 0 {
		t.Errorf("expected no lines and idx 0, got %d lines and idx %d", len(lines), idx)
	}
}

func TestRenderUSBDeviceTree_SingleDeviceUnselected(t *testing.T) {
	devices := []model.USBDevice{{DeviceID: "1-4", Product: "My Device"}}
	lines, idx := renderUSBDeviceTree(devices, 0, -1, "")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if idx != 1 {
		t.Errorf("expected next idx 1, got %d", idx)
	}
	plain := stripANSI(lines[0])
	if !strings.Contains(plain, "My Device") {
		t.Errorf("line should contain device name, got %q", plain)
	}
	if !strings.HasPrefix(plain, " ") {
		t.Errorf("unselected item should start with space, got %q", plain)
	}
}

func TestRenderUSBDeviceTree_SelectedDevice(t *testing.T) {
	devices := []model.USBDevice{{DeviceID: "1-4", Product: "My Device"}}
	lines, _ := renderUSBDeviceTree(devices, 5, 5, "") // startIdx == selectedItem
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[0], ">") {
		t.Errorf("selected item should start with '>', got %q", lines[0])
	}
}

func TestRenderUSBDeviceTree_IndexTracking(t *testing.T) {
	devices := []model.USBDevice{
		{DeviceID: "1-4", Product: "A"},
		{DeviceID: "1-5", Product: "B"},
	}
	_, idx := renderUSBDeviceTree(devices, 7, -1, "")
	if idx != 9 {
		t.Errorf("expected next idx 9 (7+2), got %d", idx)
	}
}

func TestRenderUSBDeviceTree_NestedDevicesExpandedInline(t *testing.T) {
	devices := []model.USBDevice{
		{
			DeviceID: "1-4",
			Product:  "Hub",
			USBDevices: []model.USBDevice{
				{DeviceID: "1-4.1", Product: "Child"},
			},
		},
	}
	lines, idx := renderUSBDeviceTree(devices, 0, -1, "")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (hub + child), got %d", len(lines))
	}
	if idx != 2 {
		t.Errorf("expected next idx 2, got %d", idx)
	}
	if !strings.Contains(stripANSI(lines[1]), "Child") {
		t.Errorf("second line should contain 'Child', got %q", lines[1])
	}
}

func TestRenderUSBDeviceTree_ConnectorChars(t *testing.T) {
	devices := []model.USBDevice{
		{DeviceID: "1-4", Product: "First"},
		{DeviceID: "1-5", Product: "Last"},
	}
	lines, _ := renderUSBDeviceTree(devices, 0, -1, "")
	firstPlain := stripANSI(lines[0])
	lastPlain := stripANSI(lines[1])
	if !strings.Contains(firstPlain, "├") {
		t.Errorf("non-last item should use '├', got %q", firstPlain)
	}
	if !strings.Contains(lastPlain, "╰") {
		t.Errorf("last item should use '╰', got %q", lastPlain)
	}
}

// moveSelection

func TestMoveSelection_Down(t *testing.T) {
	m := UIModel{
		items:        []listItem{{id: "a"}, {id: "b"}, {id: "c"}},
		selectedItem: 1,
		selectedID:   "b",
	}
	result := moveSelection(m, +1).(UIModel)
	if result.selectedItem != 2 || result.selectedID != "c" {
		t.Errorf("expected item 2 'c', got %d %q", result.selectedItem, result.selectedID)
	}
}

func TestMoveSelection_Up(t *testing.T) {
	m := UIModel{
		items:        []listItem{{id: "a"}, {id: "b"}, {id: "c"}},
		selectedItem: 1,
		selectedID:   "b",
	}
	result := moveSelection(m, -1).(UIModel)
	if result.selectedItem != 0 || result.selectedID != "a" {
		t.Errorf("expected item 0 'a', got %d %q", result.selectedItem, result.selectedID)
	}
}

func TestMoveSelection_NoWrapAtEnd(t *testing.T) {
	m := UIModel{
		items:        []listItem{{id: "a"}, {id: "b"}},
		selectedItem: 1,
		selectedID:   "b",
	}
	result := moveSelection(m, +1).(UIModel)
	if result.selectedItem != 1 {
		t.Errorf("expected no movement past end, got %d", result.selectedItem)
	}
}

func TestMoveSelection_NoWrapAtStart(t *testing.T) {
	m := UIModel{
		items:        []listItem{{id: "a"}, {id: "b"}},
		selectedItem: 0,
		selectedID:   "a",
	}
	result := moveSelection(m, -1).(UIModel)
	if result.selectedItem != 0 {
		t.Errorf("expected no movement before start, got %d", result.selectedItem)
	}
}
