# LeaseLens-Dnsmasq

**Your network. Clearly in view.**

LeaseLens-Dnsmasq is a lightweight web dashboard for viewing and monitoring DHCP leases from [dnsmasq](https://thekelleys.org.uk/dnsmasq/doc.html). It is written in Go and provides a searchable, sortable view of network devices, manufacturer information, reservation status, lease expiry, and reachability.

![LeaseLens dashboard](https://raw.githubusercontent.com/anonymous-writer/leaselens-dnsmasq/main/docs/screenshot1.png)

![LeaseLens device details](https://raw.githubusercontent.com/anonymous-writer/leaselens-dnsmasq/main/docs/screenshot2.png)

## Features in the Docker image

- Searchable and sortable DHCP lease table for IPv4 and IPv6
- Filters for IP version, DHCP reservations, and device reachability status
- HTTP/HTTPS checks and ping-based status indication
- Clickable IP addresses and hostnames when the hostname resolves to the lease IP
- Device details including hostname, IP, MAC, manufacturer, reservation status, lease expiry, and detected web URLs
- Local IEEE OUI database lookup for MAC manufacturers; locally administered/randomized MAC addresses are identified separately
- `NEW` indicator for MAC addresses not in the remembered-device list
- Per-device actions to mark a device as known or mark it as new again
- Bulk action to mark detected reservations as known
- Bulk action to mark all configured IPv4 reservations as known, including reservations for currently offline devices
- Bulk action to reset the remembered-device list (confirmation required)
- Persistent remembered-device data in `/data/known-macs.json`
- Infinite leases displayed as `Never`; expired leases displayed as `Expired`
- Manual refresh; lease countdowns update locally between refreshes
- Dark/light theme and responsive desktop/mobile interface
- JSON endpoint for leases and API actions for remembered MAC addresses
- Multi-architecture image (`linux/amd64` and `linux/arm64`)
- Runs as a non-root user inside the container

**Native-only features are not included in the Docker image.** The native Linux build can display the installed dnsmasq version and optionally provide an IPv4 lease-release action. See the main repository README for native installation and setup.

## Quick start

Create a directory for persistent application data:

```bash
mkdir -p data
```

Start the latest stable Docker Hub image:

```bash
docker run -d \
  --name leaselens-dnsmasq \
  --restart unless-stopped \
  --network host \
  -e PORT=3008 \
  -e LOG_LEVEL=INFO \
  -v /var/lib/dnsmasq:/var/lib/dnsmasq:ro \
  -v "$(pwd)/data:/data" \
  1968michael1968/leaselens-dnsmasq:latest
```

Open **http://YOUR_HOST:3008** in your browser, replacing `YOUR_HOST` with the IP address or hostname of the machine running the container.

Host networking allows LeaseLens to reach devices on your LAN when the host has the necessary routes. With `--network host`, do not use `-p` or `--publish`. The application and its API have no built-in authentication; keep them on a trusted network and do not expose them directly to the public internet.

## Required dnsmasq files

By default, LeaseLens reads:

- `/var/lib/dnsmasq/dnsmasq.leases` — DHCP lease file
- `/var/lib/dnsmasq/dnsmasq.dhcphosts` — optional DHCP reservation file

Example dnsmasq configuration:

```ini
dhcp-leasefile=/var/lib/dnsmasq/dnsmasq.leases
dhcp-hostsfile=/var/lib/dnsmasq/dnsmasq.dhcphosts
```

Mount the containing directory read-only into the container. Mounting the directory rather than individual files helps ensure the application sees the current lease file if dnsmasq replaces it. The hosts file is optional; reservations defined only in that file cannot be identified when it is absent or unreadable.

## Docker Compose

Example `compose.yaml`:

```yaml
services:
  leaselens-dnsmasq:
    image: 1968michael1968/leaselens-dnsmasq:latest
    container_name: leaselens-dnsmasq
    restart: unless-stopped
    environment:
      PORT: "3008"
      LOG_LEVEL: "INFO"
      HOME: /tmp
      XDG_RUNTIME_DIR: /tmp
    network_mode: host
    volumes:
      - /var/lib/dnsmasq:/var/lib/dnsmasq:ro
      - ./data:/data
```

Create the data directory before starting:

```bash
mkdir -p data
docker compose up -d
```

## Using the dashboard

- Search by hostname, IP address, MAC address, lease time, status, or reservation state.
- Use the IP-version filter to show all leases, IPv4 only, or IPv6 only.
- Use the reservation filter to show all leases, reservations only, or dynamic leases only.
- Select a column heading to change the sort order.
- Select a row to open device details. Copy individual hostname, IP, or MAC values, or copy all device details from the dialog.
- Click **Refresh** to reload leases and repeat reachability checks. Countdown values update locally between refreshes; network checks do not run every second.
- Status colors indicate HTTP/HTTPS response, ping response without a detected web interface, or no response to either check. Hover over a status indicator for an explanation.
- `NEW` means the device's MAC address is not in the remembered list. Use **Mark as known** to remember it or **Mark as new** to remove it from that list.
- **Mark reservations known** remembers currently detected reservations with valid MAC addresses.
- **Mark all IPv4 reservations** reads the configured DHCP hosts file and can remember configured IPv4 reservations even when the devices are offline. Entries need both a valid MAC and an IPv4 address; IPv6-only entries are skipped.
- **Reset remembered** clears the remembered MAC list after confirmation. This does not delete the lease or DHCP hosts files.

Remembered devices persist across container restarts only when `/data` is mounted to a persistent host directory. IPv6 leases without a valid MAC address remain visible but cannot be reliably identified in the MAC-based remembered-device list.

## Configuration

| Environment variable | Default | Description |
|---|---|---|
| `PORT` | `5000` | Web server port |
| `HOST` | `0.0.0.0` | Web server listen address |
| `DNSMASQ_LEASES_FILE` | `/var/lib/dnsmasq/dnsmasq.leases` | DHCP lease file path |
| `DNSMASQ_HOSTS_FILE` | `/var/lib/dnsmasq/dnsmasq.dhcphosts` | Optional DHCP hosts file path |
| `KNOWN_MACS_FILE` | `/data/known-macs.json` | Persistent remembered-MAC list |
| `OUI_DATABASE_FILE` | Auto-detected | Optional path to a local IEEE OUI database; the image includes a database |
| `LOG_LEVEL` | `INFO` | Minimum verbosity: `DEBUG`, `INFO`, `WARN`, or `ERROR` |
| `WEB_UI_TIMEOUT` | `500ms` | Timeout for HTTP/HTTPS and hostname checks |
| `PING_TIMEOUT` | `1200ms` | Timeout for ping checks |
| `WEB_UI_MAX_WORKERS` | `16` | Maximum concurrent web-interface checks |
| `SYSTEM_NAME` | `LeaseLens` | Optional custom label displayed in the header |

The Docker image includes a local IEEE OUI database for manufacturer identification. No external lookup service is required. If manufacturer data cannot be loaded, leases still appear but the manufacturer may show as unknown.

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/` | Web dashboard |
| `GET` | `/leases` | JSON response containing the leases array |
| `POST` | `/known-macs` | Mark/unmark a MAC address or perform supported bulk actions |

Example:

```bash
curl http://localhost:3008/leases
```

`POST /known-macs` accepts JSON. The documented actions are:

| Action | Example payload | Result |
|---|---|---|
| `mark` (default) | `{"mac":"AA:BB:CC:DD:EE:FF","action":"mark"}` | Remember one MAC address |
| `unmark` | `{"mac":"AA:BB:CC:DD:EE:FF","action":"unmark"}` | Remove one MAC address from the remembered list |
| `reset` | `{"action":"reset"}` | Clear the remembered list |
| `mark-reservations` | `{"action":"mark-reservations"}` | Remember currently detected DHCP reservations with valid MAC addresses |

The `mark-reservations` response includes a `count` field. The web interface's **Mark all IPv4 reservations** button reads the configured hosts file directly; do not assume that this button corresponds to an API action. The API has no authentication.

## Image tags and releases

**Docker Hub — stable releases**

```bash
docker pull 1968michael1968/leaselens-dnsmasq:latest
```

**GitHub Container Registry — beta and development**

```bash
docker pull ghcr.io/anonymous-writer/leaselens-dnsmasq:go-beta
```

The `go-beta` tag tracks the latest published pre-release. Stable GHCR releases use version tags and `latest`; pre-releases do not update the GHCR `latest` tag. Publishing a GitHub Release does not publish a Docker Hub image unless a separate Docker Hub publishing workflow is configured. Use a version-specific tag when you need a fixed version.

## Security

LeaseLens does not provide built-in authentication for its web interface or API. Keep it on a trusted network and do not expose it directly to the public internet. Host networking improves LAN reachability but does not add access control.

## Source code and license

Source code: https://github.com/anonymous-writer/leaselens-dnsmasq

MIT — see [`LICENSE`](https://github.com/anonymous-writer/leaselens-dnsmasq/blob/main/LICENSE).
