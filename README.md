# USB-C TUI Utility

A Go-based text user interface utility to display USB-C device information from Linux's typec sysfs interface.

## Features

- Display connected USB-C devices with visual representation
- Show power delivery information (current, PD version)
- Directional arrows indicating power flow
- Detect device types (DisplayPort, chargers, phones, etc.)
- Works with live system or saved snapshots

## Usage

View current USB-C ports:
```bash
./usb-c
```

View from a snapshot directory:
```bash
./usb-c snapshots/charger-mac
```

## Building

```bash
go build -o usb-c .
```

## Example Output

```
port0 <--󱐋--- Charger  [3A, 3.4A, PD 2.0]
port0 ---󱐋--> DisplayPort Device  [PD 2.0]
port0 ---󱐋--> Phone/Device  [1.5A]
port0 (no device connected)
```

Arrow direction indicates power flow:
- `---󱐋--->` Port provides power to device
- `<--󱐋---` Port receives power from device

## Snapshot Tool

Use `snapshot-typec.sh` to save USB-C state for later analysis:
```bash
./snapshot-typec.sh snapshots/my-device
```

# alternatives

- Cyme (but more for other USB devices) https://github.com/tuna-f1sh/cyme?tab=readme-ov-file

# references

https://www.usb.org/sites/default/files/D1T2-3b%20-%20USB%20Type-C%20Linux%20Connector%20Class.pdf
https://www.kernel.org/doc/html/latest/driver-api/usb/typec.html
https://www.kernel.org/doc/html/latest/admin-guide/abi-testing.html#abi-sys-class-typec-port-data-role
https://github.com/torvalds/linux/blob/master/drivers/usb/typec/class.c
