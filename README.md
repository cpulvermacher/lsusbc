# lsusbc

A CLI utility to display USB-C device information from Linux's typec sysfs interface.

## Features

- Display connected USB-C devices with visual representation
- Show power delivery information (current, PD version)
- Directional arrows indicating power flow
- Detect device types (DisplayPort, chargers, phones, etc.)

## Usage

View current USB-C ports:

```bash
lsusbc
```

## Building

```bash
go build
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

## Limitations

- Amount of information available depends on hardware and kernel version.
- It's possible for the /sys/class/typec information to get stuck. `rmmod ucsi_acpi; modprobe ucsi_acpi` may help, or a reboot may be required.

# alternatives

- Cyme (but more for other USB devices) https://github.com/tuna-f1sh/cyme?tab=readme-ov-file

# references

- https://www.usb.org/sites/default/files/D1T2-3b%20-%20USB%20Type-C%20Linux%20Connector%20Class.pdf
- https://www.kernel.org/doc/html/latest/driver-api/usb/typec.html
- https://www.kernel.org/doc/html/latest/admin-guide/abi-testing.html#abi-sys-class-typec-port-data-role
- https://github.com/torvalds/linux/blob/master/drivers/usb/typec/class.c
- https://fabiensanglard.net/usbcheat/index.html
