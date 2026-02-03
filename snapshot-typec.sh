#!/bin/bash

# snapshot-typec.sh - Copy a snapshot of /sys/class/typec to a directory
# Usage: ./snapshot-typec.sh <destination-directory>

set -e

if [ $# -ne 1 ]; then
    echo "Usage: $0 <destination-directory>" >&2
    exit 1
fi

DEST_DIR="$1"
SOURCE_DIR="/sys/class/typec"

if [ ! -d "$SOURCE_DIR" ]; then
    echo "Error: $SOURCE_DIR does not exist" >&2
    exit 1
fi

# Create destination directory and convert to absolute path
mkdir -p "$DEST_DIR"
DEST_DIR=$(cd "$DEST_DIR" && pwd)

SYMLINKS_FILE="$DEST_DIR/symlinks.txt"

echo "Snapshotting $SOURCE_DIR to $DEST_DIR..."
echo "# Symlinks found in /sys/class/typec" > "$SYMLINKS_FILE"
echo "# Format: path -> target" >> "$SYMLINKS_FILE"
echo "" >> "$SYMLINKS_FILE"

# Function to copy a file's content
copy_file() {
    local src="$1"
    local dst="$2"

    # Try to read the file, skip if not readable
    if cat "$src" > "$dst" 2>/dev/null; then
        return 0
    else
        # Create empty file to mark it existed but wasn't readable
        touch "$dst"
        echo "# Could not read $src" > "$dst"
        return 1
    fi
}

# Function to check if a path is within the typec subsystem
is_typec_path() {
    local path="$1"
    # Check if path contains /typec/ in it
    [[ "$path" == *"/typec/"* ]]
}

# Process each entry in /sys/class/typec
for entry in "$SOURCE_DIR"/*; do
    [ -e "$entry" ] || continue

    entry_name=$(basename "$entry")
    echo "Processing: $entry_name"

    # Resolve symlink to real path
    if [ -L "$entry" ]; then
        real_path=$(readlink -f "$entry")
        echo "  -> Resolves to: $real_path"

        # Save symlink info
        symlink_target=$(readlink "$entry")
        echo "$entry_name -> $symlink_target" >> "$SYMLINKS_FILE"
    else
        real_path="$entry"
    fi

    # Only process if it's a typec path
    if ! is_typec_path "$real_path"; then
        echo "  Skipping: not in typec subsystem"
        continue
    fi

    # Copy files from this device tree WITHOUT following symlinks
    find "$real_path" -type f 2>/dev/null | while IFS= read -r file; do
        # Skip symlinks that go outside typec
        if [ -L "$file" ]; then
            continue
        fi

        # Get path relative to the real_path
        rel_to_device="${file#$real_path/}"

        # Construct destination path
        dest_file="$DEST_DIR/$entry_name/$rel_to_device"
        dest_dir=$(dirname "$dest_file")

        # Create directory structure
        mkdir -p "$dest_dir"

        # Copy file content
        if copy_file "$file" "$dest_file"; then
            echo "  Copied: $entry_name/$rel_to_device"
        fi
    done

    # Save directory structure info (symlinks within this device)
    find "$real_path" -type l 2>/dev/null | while IFS= read -r link; do
        link_target=$(readlink "$link" 2>/dev/null || echo "")

        if [ -n "$link_target" ]; then
            rel_to_device="${link#$real_path/}"
            echo "$entry_name/$rel_to_device -> $link_target" >> "$SYMLINKS_FILE"
        fi
    done
done

# Capture USB device information
echo ""
echo "Scanning for USB devices..."

USB_DEVICES_DIR="/sys/bus/usb/devices"
USB_DEST_DIR="$DEST_DIR/usb/devices"

# Find all USB device symlinks in partner directories
usb_devices_to_copy=()

for partner_dir in "$DEST_DIR"/port*-partner; do
    [ -d "$partner_dir" ] || continue

    partner_name=$(basename "$partner_dir")

    # Find the real partner directory in /sys/class/typec
    typec_partner_dir="$SOURCE_DIR/$partner_name"
    if [ ! -d "$typec_partner_dir" ]; then
        # Try nested structure (port0/port0-partner)
        port_prefix="${partner_name%-partner}"
        typec_partner_dir="$SOURCE_DIR/$port_prefix/$partner_name"
    fi

    [ -d "$typec_partner_dir" ] || continue

    # Look for USB device symlinks (pattern: digits-digits, like "1-4" or "2-1.3")
    for entry in "$typec_partner_dir"/*; do
        [ -L "$entry" ] || continue

        entry_name=$(basename "$entry")

        # Check if it matches USB device pattern (starts with digit, contains dash, no colon)
        if [[ "$entry_name" =~ ^[0-9]+-[0-9] ]] && [[ ! "$entry_name" =~ : ]]; then
            # This is a USB device symlink
            if [ -d "$USB_DEVICES_DIR/$entry_name" ]; then
                echo "  Found USB device: $entry_name (in $partner_name)"
                usb_devices_to_copy+=("$entry_name")

                # Record the symlink in symlinks.txt
                echo "$partner_name/$entry_name -> ../../usb/devices/$entry_name" >> "$SYMLINKS_FILE"

                # Create a marker file in the snapshot partner directory
                # This allows the Go code to find the device when scanning the directory
                touch "$partner_dir/$entry_name"
            fi
        fi
    done
done

# Copy USB device directories
if [ ${#usb_devices_to_copy[@]} -gt 0 ]; then
    echo ""
    echo "Copying USB device information..."
    mkdir -p "$USB_DEST_DIR"

    for device_id in "${usb_devices_to_copy[@]}"; do
        device_src="$USB_DEVICES_DIR/$device_id"
        device_dst="$USB_DEST_DIR/$device_id"

        [ -d "$device_src" ] || continue

        mkdir -p "$device_dst"

        # Copy key files from the USB device
        for file in manufacturer product serial idVendor idProduct; do
            if [ -f "$device_src/$file" ]; then
                if copy_file "$device_src/$file" "$device_dst/$file"; then
                    echo "  Copied: usb/devices/$device_id/$file"
                fi
            fi
        done
    done
fi

echo ""
echo "Snapshot complete: $DEST_DIR"
echo "Total files copied: $(find "$DEST_DIR" -type f -not -name "symlinks.txt" | wc -l)"
echo "Symlinks info saved to: $SYMLINKS_FILE"
