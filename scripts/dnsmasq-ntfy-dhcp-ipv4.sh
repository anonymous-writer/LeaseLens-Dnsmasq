#!/bin/sh
# Thin dnsmasq dhcp-script bridge to LeaseLens' built-in Go ntfy module.
# Device allow-list lookup and push delivery happen in Go using the configured DB.
set -u

ACTION=${1:-}
MAC=${2:-}
IP=${3:-}
HOSTNAME=${4:-}

# Pass all actions to Go so it can log and decide which events merit a notification.
echo "leaselens-ntfy: DHCP event action=$ACTION mac=$MAC ip=$IP hostname=$HOSTNAME" >&2

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
RC=$?
echo "leaselens-ntfy: LeaseLens event handler returned status=$RC" >&2
# DHCP processing must not be affected by notification failures.
exit 0
