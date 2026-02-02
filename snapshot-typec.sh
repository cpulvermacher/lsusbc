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

echo "Snapshotting $SOURCE_DIR to $DEST_DIR..."

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
        echo "$symlink_target" > "$DEST_DIR/$entry_name.symlink"
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

    # Save directory structure info (non-recursive symlinks within this device)
    find "$real_path" -type l 2>/dev/null | while IFS= read -r link; do
        # Check if link target is still within typec
        link_target=$(readlink "$link" 2>/dev/null || echo "")

        if [ -n "$link_target" ]; then
            rel_to_device="${link#$real_path/}"
            mkdir -p "$DEST_DIR/$entry_name/$(dirname "$rel_to_device")"
            echo "$link_target" > "$DEST_DIR/$entry_name/$rel_to_device.symlink"
        fi
    done
done

echo ""
echo "Snapshot complete: $DEST_DIR"
echo "Total files copied: $(find "$DEST_DIR" -type f -not -name "*.symlink" | wc -l)"
echo "Total symlinks saved: $(find "$DEST_DIR" -type f -name "*.symlink" | wc -l)"
