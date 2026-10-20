#!/bin/sh
# Meldet unbekannte IPv4-DHCP-Clients über ntfy.
# known-macs.json wird ausschließlich gelesen.

set -u
CONFIG="/home/pi/leaselens-dnsmasq/dnsmasq-ntfy.env"
KNOWN_MACS_FILE="/home/pi/leaselens-dnsmasq/known-macs.json"

ACTION=${1:-}
MAC=${2:-unknown}
IP=${3:-unknown}
HOSTNAME=${4:-}

case "$ACTION" in add|old) ;; *) exit 0 ;; esac

if [ ! -r "$KNOWN_MACS_FILE" ]; then
    echo "dnsmasq-ntfy: Geräteliste nicht lesbar: $KNOWN_MACS_FILE" >&2
    exit 0
fi
if ! command -v python3 >/dev/null 2>&1; then
    echo "dnsmasq-ntfy: python3 ist nicht installiert" >&2
    exit 0
fi

CHECK_RESULT=$(python3 - "$KNOWN_MACS_FILE" "$MAC" "$IP" <<'PY'
import ipaddress, json, sys
path, mac, ip = sys.argv[1:4]
try:
    address = ipaddress.ip_address(ip)
except ValueError:
    print("invalid_ip"); raise SystemExit(0)
if not isinstance(address, ipaddress.IPv4Address):
    print("not_ipv4"); raise SystemExit(0)
try:
    with open(path, "r", encoding="utf-8") as f:
        known = json.load(f)
except (OSError, json.JSONDecodeError) as exc:
    print("json_error")
    print(str(exc), file=sys.stderr)
    raise SystemExit(0)
if not isinstance(known, list) or not all(isinstance(item, str) for item in known):
    print("json_error")
    print("JSON muss eine Liste aus MAC-Adress-Strings enthalten.", file=sys.stderr)
    raise SystemExit(0)
known_macs = {item.strip().upper() for item in known}
print("known" if mac.strip().upper() in known_macs else "unknown")
PY
) || {
    echo "dnsmasq-ntfy: Prüfung der Geräteliste fehlgeschlagen" >&2
    exit 0
}

case "$CHECK_RESULT" in
    known) exit 0 ;;
    not_ipv4|invalid_ip) exit 0 ;;
    json_error)
        echo "dnsmasq-ntfy: known-macs.json ist ungültig; keine Meldung gesendet" >&2
        exit 0 ;;
    unknown) ;;
    *)
        echo "dnsmasq-ntfy: Unerwartetes Prüfergebnis: $CHECK_RESULT" >&2
        exit 0 ;;
esac

if [ ! -r "$CONFIG" ]; then
    echo "dnsmasq-ntfy: Konfiguration fehlt: $CONFIG" >&2
    exit 0
fi
. "$CONFIG"
if [ -z "${NTFY_URL:-}" ]; then
    echo "dnsmasq-ntfy: NTFY_URL fehlt in $CONFIG" >&2
    exit 0
fi

show_or_missing() {
    # Leere Werte bleiben leer; die zugehörige Zeile wird später ausgeblendet.
    printf '%s' "$1"
}

INTERFACE=$(show_or_missing "${DNSMASQ_INTERFACE:-}")
DOMAIN=$(show_or_missing "${DNSMASQ_DOMAIN:-}")
SUPPLIED_HOSTNAME=$(show_or_missing "${DNSMASQ_SUPPLIED_HOSTNAME:-}")
VENDOR_CLASS=$(show_or_missing "${DNSMASQ_VENDOR_CLASS:-}")
CLIENT_ID=$(show_or_missing "${DNSMASQ_CLIENT_ID:-}")
TAGS=$(show_or_missing "${DNSMASQ_TAGS:-}")
REQUESTED_OPTIONS=$(show_or_missing "${DNSMASQ_REQUESTED_OPTIONS:-}")
TIME_REMAINING=$(show_or_missing "${DNSMASQ_TIME_REMAINING:-}")
LEASE_EXPIRES=$(show_or_missing "${DNSMASQ_LEASE_EXPIRES:-}")
RELAY_ADDRESS=$(show_or_missing "${DNSMASQ_RELAY_ADDRESS:-}")
CIRCUIT_ID=$(show_or_missing "${DNSMASQ_CIRCUIT_ID:-}")
SUBSCRIBER_ID=$(show_or_missing "${DNSMASQ_SUBSCRIBER_ID:-}")
REMOTE_ID=$(show_or_missing "${DNSMASQ_REMOTE_ID:-}")
MUD_URL=$(show_or_missing "${DNSMASQ_MUD_URL:-}")
DATA_MISSING=$(show_or_missing "${DNSMASQ_DATA_MISSING:-}")
OLD_HOSTNAME=$(show_or_missing "${DNSMASQ_OLD_HOSTNAME:-}")
LOG_DHCP=$(show_or_missing "${DNSMASQ_LOG_DHCP:-}")

USER_CLASSES=""
i=0
while [ "$i" -lt 32 ]; do
    var_name="DNSMASQ_USER_CLASS$i"
    value=$(printenv "$var_name" 2>/dev/null || true)
    if [ -n "$value" ]; then
        if [ -n "$USER_CLASSES" ]; then USER_CLASSES="${USER_CLASSES}
"; fi
        USER_CLASSES="${USER_CLASSES}${var_name}: $value"
    fi
    i=$((i + 1))
done
[ -n "$USER_CLASSES" ] || USER_CLASSES=""

if [ -z "$HOSTNAME" ]; then HOSTNAME=${DNSMASQ_SUPPLIED_HOSTNAME:-}; fi
[ -n "$HOSTNAME" ] || HOSTNAME="nicht übermittelt"

TIMESTAMP=$(date '+%d.%m.%Y %H:%M:%S %Z' 2>/dev/null || date)
REVERSE_DNS=$(getent hosts "$IP" 2>/dev/null | awk 'NR == 1 { print $2; exit }')
[ -n "$REVERSE_DNS" ] || REVERSE_DNS="kein Eintrag gefunden"

MESSAGE=$(cat <<MSG
NEUES IPV4-GERÄT

Hostname: $HOSTNAME
IPv4-Adresse: $IP
MAC-Adresse: $MAC
Interface: $INTERFACE
Domain: $DOMAIN
Zeitpunkt: $TIMESTAMP
Reverse DNS: $REVERSE_DNS

DHCP Vendor Class: $VENDOR_CLASS
DHCP Client ID: $CLIENT_ID
DHCP User Classes:
$USER_CLASSES
DHCP Tags: $TAGS
Angeforderte DHCP-Optionen: $REQUESTED_OPTIONS
Verbleibende Lease-Zeit (Sekunden): $TIME_REMAINING
Lease-Ablaufzeit: $LEASE_EXPIRES

Relay-Adresse: $RELAY_ADDRESS
Circuit ID: $CIRCUIT_ID
Subscriber ID: $SUBSCRIBER_ID
Remote ID: $REMOTE_ID
MUD URL: $MUD_URL
Daten unvollständig (DNSMASQ_DATA_MISSING): $DATA_MISSING
Alter Hostname: $OLD_HOSTNAME
DHCP-Logging aktiv (DNSMASQ_LOG_DHCP): $LOG_DHCP

DHCP-Ereignis: $ACTION
MSG
)

# Zeilen mit leeren Werten ausblenden; die bekannten Überschriften bleiben erhalten.
MESSAGE=$(printf '%s\\n' "$MESSAGE" | awk '
    /^DHCP User Classes:$/ { print; in_user_classes=1; next }
    in_user_classes && /^[^[:space:]].*: / { in_user_classes=0 }
    in_user_classes {
        if ($0 ~ /^DNSMASQ_USER_CLASS[0-9]+: /) { print; next }
        in_user_classes=0
    }
    /^[^:]+: *$/ && $0 != "DHCP User Classes:" { next }
    { print }
')

# Wenn keine User Class vorhanden ist, auch die Überschrift entfernen.
if ! printf '%s\\n' "$MESSAGE" | grep -q '^DNSMASQ_USER_CLASS[0-9][0-9]*: '; then
    MESSAGE=$(printf '%s\\n' "$MESSAGE" | sed '/^DHCP User Classes:$/d')
fi

if [ -n "${NTFY_TOKEN:-}" ]; then
    if ! curl --fail --silent --show-error --connect-timeout 3 --max-time 10 \
        -H "Title: Neues IPv4-Gerät: $HOSTNAME" \
        -H "Tags: computer,information_source" \
        -H "Priority: high" \
        -H "Authorization: Bearer $NTFY_TOKEN" \
        --data-binary "$MESSAGE" "$NTFY_URL" >/dev/null; then
        echo "dnsmasq-ntfy: Versand fehlgeschlagen: $IP ($MAC)" >&2
        exit 0
    fi
else
    if ! curl --fail --silent --show-error --connect-timeout 3 --max-time 10 \
        -H "Title: Neues IPv4-Gerät: $HOSTNAME" \
        -H "Tags: computer,information_source" \
        -H "Priority: high" \
        --data-binary "$MESSAGE" "$NTFY_URL" >/dev/null; then
        echo "dnsmasq-ntfy: Versand fehlgeschlagen: $IP ($MAC)" >&2
        exit 0
    fi
fi

# Keine Statusdatei schreiben; known-macs.json bleibt unverändert.
exit 0
