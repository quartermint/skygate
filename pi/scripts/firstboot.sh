#!/bin/bash
# SkyGate First Boot Setup
# Runs once on first boot via systemd oneshot service.
# Sets WiFi SSID and password, then disables itself.
#
# Per D-06: SSID set by pilot during first-boot setup
# Per D-08: Default random password printed on device sticker

set -euo pipefail

FIRSTBOOT_FLAG="/data/skygate/.firstboot-complete"
HOSTAPD_CONF="/etc/hostapd/hostapd.conf"
CREDENTIALS_FILE="/data/skygate/wifi-credentials"

# Skip if already completed
if [ -f "$FIRSTBOOT_FLAG" ]; then
    echo "[firstboot] Already completed, skipping"
    exit 0
fi

# Generate random 8-character alphanumeric password
DEFAULT_PASSWORD=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 8)

echo "============================================"
echo "  SkyGate First Boot Setup"
echo "============================================"
echo ""
read -p "Enter WiFi network name (SSID) [SkyGate]: " SSID
SSID="${SSID:-SkyGate}"

read -p "Enter WiFi password [${DEFAULT_PASSWORD}]: " PASSWORD
PASSWORD="${PASSWORD:-$DEFAULT_PASSWORD}"

if [ "$PASSWORD" = "$DEFAULT_PASSWORD" ]; then
    echo ""
    echo "Using generated password: $PASSWORD"
    echo "IMPORTANT: Write this down or print it for the device sticker!"
fi

# Validate password length (WPA2 requires 8-63 chars)
if [ ${#PASSWORD} -lt 8 ] || [ ${#PASSWORD} -gt 63 ]; then
    echo "ERROR: Password must be 8-63 characters for WPA2"
    exit 1
fi

# Update hostapd config
# If OverlayFS is active, temporarily remount rw
if mount | grep "on / " | grep -q "ro,"; then
    mount -o remount,rw /
    REMOUNTED=true
else
    REMOUNTED=false
fi

sed -i "s/^ssid=.*/ssid=${SSID}/" "$HOSTAPD_CONF"
sed -i "s/^wpa_passphrase=.*/wpa_passphrase=${PASSWORD}/" "$HOSTAPD_CONF"

if [ "$REMOUNTED" = true ]; then
    mount -o remount,ro /
fi

# Save credentials to writable data partition
cat > "$CREDENTIALS_FILE" << EOF
SSID=${SSID}
PASSWORD=${PASSWORD}
GENERATED=$(date -Iseconds)
EOF
chmod 600 "$CREDENTIALS_FILE"

# Mark first boot complete
touch "$FIRSTBOOT_FLAG"

# Restart hostapd with new config
systemctl restart hostapd

echo ""
echo "Setup complete!"
echo "Devices can now connect to WiFi network: ${SSID}"
echo "============================================"
