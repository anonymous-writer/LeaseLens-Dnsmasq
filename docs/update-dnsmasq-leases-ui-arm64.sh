#!/usr/bin/env bash
set -Eeuo pipefail

# Install a manually downloaded Linux ARM64 release archive.
# This script intentionally does not access the internet.

BASE_DIR="/media/pi/Ablage/Datenablage"
ARCHIVE_NAME="dnsmasq-leases-ui-linux-arm64.tar.gz"
BINARY_NAME="dnsmasq-leases-ui-linux-arm64"
ARCHIVE_PATH="${BASE_DIR}/${ARCHIVE_NAME}"
EXTRACTED_PATH="${BASE_DIR}/${BINARY_NAME}"
INSTALL_PATH="/usr/bin/dnsmasq-leases-ui"
SERVICE_NAME="dnsmasq-leases-ui.service"
SUMS_PATH="${BASE_DIR}/SHA256SUMS"

usage() {
  cat <<'USAGE'
Usage: ./update-dnsmasq-leases-ui-arm64.sh

Place the manually downloaded release archive here first:
  /media/pi/Ablage/Datenablage/dnsmasq-leases-ui-linux-arm64.tar.gz

Optional: also place the release's SHA256SUMS file in the same directory.
The script extracts the ARM64 executable to that directory, installs it as
/usr/bin/dnsmasq-leases-ui, backs up an existing installation, and restarts
the dnsmasq-leases-ui systemd user service when available.

The script does not download anything from the internet.
Run it as your normal user (e.g. pi), not with sudo. It invokes sudo only
for installing the binary into /usr/bin.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi
if [[ "$#" -gt 0 ]]; then
  echo "Error: This script does not accept arguments." >&2
  usage >&2
  exit 2
fi
if [[ "$EUID" -eq 0 ]]; then
  echo "Error: Run this script as user pi, without sudo." >&2
  echo "It will ask for sudo only when installing into /usr/bin." >&2
  exit 1
fi

case "$(uname -m)" in
  aarch64|arm64) ;;
  *)
    echo "Error: This script supports only 64-bit ARM Linux (aarch64/arm64). Detected: $(uname -m)" >&2
    exit 1
    ;;
esac

for cmd in tar sha256sum sudo install systemctl; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Error: Required command not found: $cmd" >&2
    exit 1
  fi
done

if [[ ! -d "$BASE_DIR" ]]; then
  echo "Error: Directory does not exist: $BASE_DIR" >&2
  echo "Make sure /media/pi/Ablage is mounted." >&2
  exit 1
fi
if [[ ! -f "$ARCHIVE_PATH" ]]; then
  echo "Error: Release archive not found:" >&2
  echo "  $ARCHIVE_PATH" >&2
  echo >&2
  echo "Download the ARM64 .tar.gz asset from the GitHub Release and copy it there." >&2
  exit 1
fi

# If the published checksum file is also present, validate the archive before extraction.
if [[ -f "$SUMS_PATH" ]]; then
  echo "Checking archive against ${SUMS_PATH}..."
  if ! (cd "$BASE_DIR" && sha256sum --check --ignore-missing "$SUMS_PATH"); then
    echo "Error: Checksum verification failed. Nothing was installed." >&2
    exit 1
  fi
else
  echo "Warning: SHA256SUMS not found; skipping checksum verification." >&2
fi

TMP_DIR="$(mktemp -d)"
cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

# Find the expected executable in the archive. Extract only that member, rather
# than unpacking arbitrary archive paths into the filesystem.
ARCHIVE_MEMBER="$(tar -tzf "$ARCHIVE_PATH" | grep -E '(^|/)dnsmasq-leases-ui-linux-arm64$' | head -n 1 || true)"
if [[ -z "$ARCHIVE_MEMBER" ]]; then
  echo "Error: Archive does not contain ${BINARY_NAME}." >&2
  echo "Archive contents:" >&2
  tar -tzf "$ARCHIVE_PATH" >&2 || true
  exit 1
fi

echo "Extracting ${BINARY_NAME}..."
if ! tar -xOzf "$ARCHIVE_PATH" "$ARCHIVE_MEMBER" > "${TMP_DIR}/${BINARY_NAME}"; then
  echo "Error: Could not extract executable from archive." >&2
  exit 1
fi
chmod 0755 "${TMP_DIR}/${BINARY_NAME}"

# Keep an extracted copy in the requested data directory as well.
install -m 0755 "${TMP_DIR}/${BINARY_NAME}" "${EXTRACTED_PATH}.new"
mv -f "${EXTRACTED_PATH}.new" "$EXTRACTED_PATH"

# Preserve the current binary before replacing it.
if [[ -e "$INSTALL_PATH" ]]; then
  BACKUP_PATH="${INSTALL_PATH}.bak.$(date +%Y%m%d-%H%M%S)"
  echo "Backing up existing installation to ${BACKUP_PATH}..."
  sudo cp -a "$INSTALL_PATH" "$BACKUP_PATH"
fi

echo "Installing native ARM64 binary to ${INSTALL_PATH}..."
sudo install -m 0755 "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_PATH}.new"
sudo mv -f "${INSTALL_PATH}.new" "$INSTALL_PATH"

# systemctl --user must run as the logged-in user (not via sudo).
if systemctl --user cat "$SERVICE_NAME" >/dev/null 2>&1; then
  echo "Restarting user service ${SERVICE_NAME}..."
  if systemctl --user restart "$SERVICE_NAME"; then
    echo "Service restarted successfully."
  else
    echo "Warning: Binary installed, but the service could not be restarted automatically." >&2
    echo "Run: systemctl --user restart ${SERVICE_NAME}" >&2
  fi
else
  echo "Note: User service ${SERVICE_NAME} was not found; service restart skipped."
fi

echo
echo "Update completed."
echo "Archive:       ${ARCHIVE_PATH}"
echo "Extracted:     ${EXTRACTED_PATH}"
echo "Installed:     ${INSTALL_PATH}"
echo "Check service: systemctl --user status ${SERVICE_NAME}"
echo "Check version: systemctl --user status ${SERVICE_NAME}"
