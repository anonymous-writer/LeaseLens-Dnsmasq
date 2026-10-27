#!/bin/sh
# Thin dnsmasq dhcp-script bridge to LeaseLens' built-in Go ntfy module.
# Device allow-list lookup and push delivery happen in Go using the configured DB.
set -u

ACTION=${1:-}
MAC=${2:-}
IP=${3:-}
HOSTNAME=${4:-}

# Only a fresh lease is a new-device notification. "old" is a restored lease.
[ "$ACTION" = "add" ] || exit 0

BINARY=${LEASELENS_BINARY:-/usr/bin/leaselens-dnsmasq}
CONFIG=${LEASELENS_CONFIG:-/home/pi/leaselens-dnsmasq/config.yaml}

if [ ! -x "$BINARY" ]; then
    echo "leaselens-ntfy: LeaseLens binary not executable: $BINARY" >&2
    exit 0
fi
if [ ! -r "$CONFIG" ]; then
    echo "leaselens-ntfy: config not readable: $CONFIG" >&2
    exit 0
fi

LEASELENS_DHCP_EVENT_ACTION="$ACTION" \
LEASELENS_DHCP_EVENT_MAC="$MAC" \
LEASELENS_DHCP_EVENT_IP="$IP" \
LEASELENS_DHCP_EVENT_HOSTNAME="$HOSTNAME" \
    "$BINARY" --config "$CONFIG"
# DHCP processing must not be affected by notification failures.
exit 0
