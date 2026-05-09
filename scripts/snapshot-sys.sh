#!/bin/bash
# Creates a snapshot of sysfs for USB-C related devices.
# Usage: ./snapshot-sys.sh [destination]   (default: ./sys-snapshot)
set -euo pipefail

DEST="${1:-./sys-snapshot-$(date +%Y%m%d-%H%M%S)}"
mkdir -p "$DEST"

# Copy a tree under /sys into $DEST, preserving symlinks as-is.
# Never follows symlinks — no loop risk.
copy_tree() {
    local src="$1"
    echo "  copying: $src" >&2
    find "$src" -mindepth 0 2>/dev/null | while IFS= read -r item; do
        local rel="${item#/sys}"
        local dst="$DEST$rel"
        if [ -L "$item" ]; then
            mkdir -p "$(dirname "$dst")"
            ln -sfn "$(readlink "$item")" "$dst" 2>/dev/null || true
        elif [ -d "$item" ]; then
            mkdir -p "$dst"
        elif [ -f "$item" ]; then
            mkdir -p "$(dirname "$dst")"
            if [ -r "$item" ]; then
                cat "$item" > "$dst" 2>/dev/null || touch "$dst"
            else
                touch "$dst"
            fi
        fi
    done
}

# Walk up from a resolved typec path to find the device that owns it.
# e.g. .../USBC000:00/typec/port0 -> .../USBC000:00
find_typec_device_root() {
    local dir="$1"
    while [ "$dir" != "/" ]; do
        [ "$(basename "$dir")" = "typec" ] && { dirname "$dir"; return 0; }
        dir=$(dirname "$dir")
    done
    return 1
}

echo "=== Class directories ===" >&2
for class in /sys/class/typec /sys/class/power_supply /sys/class/usb_power_delivery; do
    [ -d "$class" ] || { echo "  skipping (not found): $class" >&2; continue; }
    copy_tree "$class"
done

echo "=== Typec device trees ===" >&2
declare -A seen_device_roots
for entry in /sys/class/typec/*; do
    [ -e "$entry" ] || continue
    real=$(readlink -f "$entry" 2>/dev/null) || continue
    device_root=$(find_typec_device_root "$real") || continue
    [[ "${seen_device_roots[$device_root]+_}" ]] && continue
    seen_device_roots["$device_root"]=1
    copy_tree "$device_root"
done

echo "=== power_supply device dirs ===" >&2
declare -A seen_ps
for entry in /sys/class/power_supply/*; do
    [ -e "$entry" ] || continue
    real=$(readlink -f "$entry" 2>/dev/null) || continue
    [ -d "$real" ] || continue
    [[ "${seen_ps[$real]+_}" ]] && continue
    seen_ps["$real"]=1
    copy_tree "$real"
done

echo "=== usb_power_delivery device dirs ===" >&2
declare -A seen_upd
for entry in /sys/class/usb_power_delivery/*; do
    [ -e "$entry" ] || continue
    real=$(readlink -f "$entry" 2>/dev/null) || continue
    [ -d "$real" ] || continue
    [[ "${seen_upd[$real]+_}" ]] && continue
    seen_upd["$real"]=1
    copy_tree "$real"
done

echo "=== USB device trees (from typec partner symlinks) ===" >&2
declare -A seen_usb
for entry in /sys/class/typec/*; do
    [ -e "$entry" ] || continue
    real=$(readlink -f "$entry" 2>/dev/null) || continue
    # Only process partner directories
    [[ "$(basename "$real")" == *-partner ]] || continue
    while IFS= read -r link; do
        name=$(basename "$link")
        # USB device pattern: digits-digits[.digits...], no colon (excludes interface nodes)
        [[ "$name" =~ ^[0-9]+-[0-9] ]] || continue
        [[ "$name" =~ : ]] && continue
        usb_real=$(readlink -f "$link" 2>/dev/null) || continue
        [ -d "$usb_real" ] || continue
        [[ "${seen_usb[$usb_real]+_}" ]] && continue
        seen_usb["$usb_real"]=1
        copy_tree "$usb_real"
    done < <(find "$real" -maxdepth 1 -type l 2>/dev/null)
done

echo "=== USB device trees (all bus/usb devices) ===" >&2
if [ -d "/sys/bus/usb/devices" ]; then
    copy_tree "/sys/bus/usb/devices"
    for entry in /sys/bus/usb/devices/*; do
        [ -e "$entry" ] || continue
        name=$(basename "$entry")
        # Only root-level devices: "N-M" with no dots (direct port on root hub)
        [[ "$name" =~ ^[0-9]+-[0-9]+$ ]] || continue
        usb_real=$(readlink -f "$entry" 2>/dev/null) || continue
        [ -d "$usb_real" ] || continue
        [[ "${seen_usb[$usb_real]+_}" ]] && continue
        seen_usb["$usb_real"]=1
        copy_tree "$usb_real"
    done
fi

echo "Done. Snapshot written to: $DEST" >&2
echo ""
echo "For submitting bug reports, create a compressed tarball using:"
echo "  tar czvf $DEST.tar.gz $DEST"
