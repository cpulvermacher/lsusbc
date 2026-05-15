# lsusbc

A CLI utility to display USB Type-C device and USB Power Delivery (PD) information from Linux's typec sysfs interface.

![Demo](./demo.gif)

## Features

- Display connected USB-C and USB devices with visual representation
- Show power information (max power, current/voltage capabilities, Power Delivery version, energy flow)
- Details on connection speed, manufacturer info, driver, etc.
- Show alternate modes (DisplayPort, Thunderbolt, etc.)

## Usage

Run without arguments for an interactive TUI:

```bash
lsusbc
```

Or use it like `lsusb` for quick one-shot output:

```bash
lsusbc -l        # compact tree overview
lsusbc -v        # full details for every device
```

Arrow direction indicates power flow:

- `===󱐋==>` Port is providing power to the connected device
- `<==󱐋===` Port is receiving power (charging)

## Building

```bash
go build
```

## Limitations

- Amount of information available depends on hardware and kernel version.
- It's possible for the /sys/class/typec information to get stuck. `rmmod ucsi_acpi; modprobe ucsi_acpi` may help, or a reboot may be required.

## Bug Reports

If you encounter any issues, please create and share a snapshot of your USB devices. Create one by running
`bash ./scripts/snapshot-sys.sh`
Create a tarball using the displayed command.

# Alternatives

- Cyme (but more for other USB devices) https://github.com/tuna-f1sh/cyme
- https://github.com/doug-gilbert/lsucpd
- https://github.com/darrylmorley/whatcable - Shows info about USB-C cables, Mac OS only

# References

- https://www.kernel.org/doc/html/latest/driver-api/usb/typec.html
- https://www.kernel.org/doc/html/latest/admin-guide/abi-testing.html#abi-sys-class-typec-port-data-role
- https://github.com/torvalds/linux/blob/master/drivers/usb/typec/class.c
- https://hackaday.com/series_of_posts/all-about-usb-c/
- https://fabiensanglard.net/usbcheat/index.html
