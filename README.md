# dnsmasq-leases-ui

[![GHCR](https://img.shields.io/badge/GHCR-docker%20image-blue?logo=github)](https://github.com/anonymous-writer/dnsmasq-leases-ui/pkgs/container/dnsmasq-leases-ui)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/github/license/anonymous-writer/dnsmasq-leases-ui)](LICENSE)

A lightweight web interface for the [dnsmasq](https://thekelleys.org.uk/dnsmasq/doc.html) DHCP lease file, written in Go. It provides a searchable, sortable view of DHCP leases, detects reachable HTTP/HTTPS interfaces on client devices, and lets you manage which devices have already been acknowledged.

![Screenshot](https://raw.githubusercontent.com/anonymous-writer/dnsmasq-leases-ui/main/docs/screenshot.png)

## Features

- Searchable and sortable DHCP lease table
- IPv4 and IPv6 support, with filters for IP version and DHCP reservations
- Manual refresh of the lease list; lease countdowns update locally in the browser
- Device details dialog with hostname, IP, MAC, reservation status, lease end, and detected web URLs
- Status indicator: green when HTTP/HTTPS responds, yellow when ping responds but no web interface is detected, red when neither check responds
- Automatically detects reachable HTTP/HTTPS interfaces and makes device IP addresses clickable
- Shows a clickable hostname when it resolves to the corresponding lease IP
- Detects DHCP reservations from the optional `dnsmasq.dhcphosts` file
- Displays infinite leases as `Never` and expired leases as `Expired`
- Detects newly seen devices by MAC address and marks them with a `NEW` indicator
- Persistently remembers acknowledged devices in `/data/known-macs.json`
- Per-device actions to mark a device as known or mark it as new again
- Bulk actions to mark all detected DHCP reservations as known or reset the remembered-device list
- IPv6 leases without a valid MAC address remain visible but are not marked as new
- Dark/light theme and responsive web interface
- JSON API at `/leases` and POST actions at `/known-macs`
- Bounded concurrent checks for device web interfaces
- Multi-architecture Docker image: `linux/amd64` and `linux/arm64`
- Runs as a non-root user in the provided Docker image

## Quick start

The current development image is published to GitHub Container Registry with the `go-beta` tag. Pull it with:

```bash
docker pull ghcr.io/anonymous-writer/dnsmasq-leases-ui:go-beta
```

The `latest` tag is intended for the stable release. The manual development workflow described under [Docker image publishing](#docker-image-publishing) updates `go-beta` only; it does not update `latest` when you publish a GitHub Release or Pre-release.

The container reads these files by default:

```text
/var/lib/dnsmasq/dnsmasq.leases
/var/lib/dnsmasq/dnsmasq.dhcphosts
```

The hosts file is optional. Mount the containing directory rather than individual files so the container continues to see the current lease file if `dnsmasq` replaces it.

### Docker

```bash
docker run -d \
  --name dnsmasq-leases-ui \
  --restart unless-stopped \
  --network host \
  -e PORT=3008 \
  -v /var/lib/dnsmasq:/var/lib/dnsmasq:ro \
  -v "$(pwd)/data:/data" \
  ghcr.io/anonymous-writer/dnsmasq-leases-ui:go-beta
```

Create the persistent data directory before starting the container:

```bash
mkdir -p data
```

Open `http://<host>:3008`.

For the stable image, replace `:go-beta` with `:latest`.

Host networking lets the app reach devices on the LAN, including IPv6 devices, when the host itself has the required routes. With host networking, do not use `-p`/`--publish`; the application listens directly on the host network.

### Docker Compose / Podman Compose

Example `compose.yaml`:

```yaml
services:
  dnsmasq-leases-ui:
    image: ghcr.io/anonymous-writer/dnsmasq-leases-ui:go-beta
    container_name: dnsmasq-leases-ui
    restart: unless-stopped
    environment:
      PORT: "3008"
      HOME: /tmp
      XDG_RUNTIME_DIR: /tmp
    network_mode: host
    volumes:
      - /var/lib/dnsmasq:/var/lib/dnsmasq:ro
      - ./data:/data:Z
```

Create the data directory before the first start:

```bash
mkdir -p data
```

Start the service with `docker compose up -d` or `podman compose up -d`.

The `:Z` suffix is useful with Podman on SELinux-enabled systems. For Docker, `./data:/data` can be used if SELinux relabeling is not required.

#### Rootless Podman: permissions for remembered devices

The application runs as a non-root user inside the container. The `/data` directory must be writable so the app can save `known-macs.json`. With the supplied image, the application user is normally UID `100` and GID `101`.

For rootless Podman, set ownership from inside Podman's user namespace:

```bash
cd /path/to/project
mkdir -p data
podman unshare chown -R 100:101 ./data
```

Then start or restart the service. This is a one-time ownership fix for the data directory; do not remove the directory to fix permissions, because it contains the remembered-device list.

You can test write access with:

```bash
podman exec dnsmasq-leases-ui sh -c \
  'id; ls -ldn /data; touch /data/testfile && rm /data/testfile'
```

If you use a different image or change its configured user, verify the UID/GID with `podman exec dnsmasq-leases-ui id` and adapt the ownership command accordingly.

### Configure dnsmasq

Configure dnsmasq to write its lease file to the mounted directory:

```ini
dhcp-leasefile=/var/lib/dnsmasq/dnsmasq.leases
```

If you use a separate DHCP hosts file, configure that too:

```ini
dhcp-hostsfile=/var/lib/dnsmasq/dnsmasq.dhcphosts
```

Create the directory before restarting dnsmasq:

```bash
sudo mkdir -p /var/lib/dnsmasq
```

After changing the configuration, restart dnsmasq and verify that the files exist and contain the expected data.

## Using the interface

- Use the search box to filter by hostname, IP, MAC address, lease time, status, or reservation state.
- Use the IP-version selector to show all leases, IPv4 only, or IPv6 only.
- Use the reservation selector to show all leases, reservations only, or dynamic leases only.
- Select a table heading to change the sort order.
- Click a row to open device details. Copy buttons are available for hostname, IP, and MAC values.
- Click **Refresh** to reload leases and redo device reachability checks. The countdown is updated locally between refreshes; reachability checks are not run every second.
- Filter by device status: HTTP/HTTPS responds, ping responds without a web interface, or no ping/web response.
- Hover over a status indicator for an explanation of the result.
- The page shows the last successful refresh time and a separate loading/error status. If a refresh fails, the previously loaded lease list remains visible.
- Copy all device details at once from the details dialog.
- A `NEW` label means the MAC address is not currently in the remembered-device list.
- In device details, choose **Mark as known** to remember a device or **Mark as new** to remove its MAC from the remembered list.
- Choose **Mark reservations known** to remember all currently detected DHCP reservations that have valid MAC addresses.
- Choose **Reset remembered** to clear all remembered MAC addresses. The UI asks for confirmation first; devices with valid MAC addresses can be marked as new again.

The remembered list is stored in `known-macs.json` and persists across container restarts when `/data` is mounted to a persistent host directory. IPv6 leases without a valid MAC address are not marked as new because they cannot be reliably identified by the MAC-based list.

## Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/` | Web interface |
| `GET` | `/leases` | JSON response containing the leases array |
| `POST` | `/known-macs` | Mark/unmark MAC addresses or perform bulk actions |

Example:

```bash
curl http://localhost:3008/leases
```

Each object in the `leases` array can include `staticIP`, `leasetime`, `leaseExpiry`, `macAddress`, `ipAddress`, `name`, `webUrl`, `webHostUrl`, `status`, and `isNew`. `leaseExpiry` is the lease-expiry Unix timestamp in seconds; `0` indicates an infinite lease.

`POST /known-macs` accepts JSON. The `action` values are:

| Action | Payload example | Result |
|---|---|---|
| `mark` (default) | `{"mac":"AA:BB:CC:DD:EE:FF","action":"mark"}` | Remember one valid MAC address |
| `unmark` | `{"mac":"AA:BB:CC:DD:EE:FF","action":"unmark"}` | Remove one MAC address from the remembered list |
| `reset` | `{"action":"reset"}` | Clear the remembered list |
| `mark-reservations` | `{"action":"mark-reservations"}` | Remember all currently detected DHCP reservations with valid MAC addresses |

The `mark-reservations` response also includes a `count` field. These endpoints do not provide authentication; keep the application on a trusted network and do not expose it directly to the public internet.

## Configuration

| Environment variable | Default | Description |
|---|---|---|
| `CONFIG_FILE` | empty | Optional path to the YAML-style configuration file |
| `DNSMASQ_LEASES_FILE` | `/var/lib/dnsmasq/dnsmasq.leases` | Path to the dnsmasq lease file |
| `DNSMASQ_HOSTS_FILE` | `/var/lib/dnsmasq/dnsmasq.dhcphosts` | Path to the optional DHCP hosts file |
| `KNOWN_MACS_FILE` | `/data/known-macs.json` | Path to the persistent remembered-MAC list |
| `HOST` | `0.0.0.0` | Address the web server listens on |
| `PORT` | `5000` | Port the web server listens on |
| `WEB_UI_TIMEOUT` | `500ms` | Timeout for an HTTP/HTTPS or hostname-resolution check; accepts Go durations such as `500ms` or seconds such as `0.5` |
| `PING_TIMEOUT` | `1200ms` | Timeout for a ping probe; accepts Go durations or a number of seconds |
| `WEB_UI_MAX_WORKERS` | `16` | Maximum concurrent IP web-interface checks |
| `APP_VERSION` | `dev` | Version shown by the application |
| `APP_RELEASE_DATE` | empty | Build/release date shown by the application |
| `REPO_URL` | `https://github.com/anonymous-writer/dnsmasq-leases-ui` | Repository URL shown in the UI |
| `LOG_LEVEL` | `INFO` | Minimum log level: `DEBUG`, `INFO`, `WARN`, or `ERROR` |

The same runtime settings can be supplied in a flat YAML-style configuration file. Example: [`config.example.yaml`](config.example.yaml). To use it, copy it to `/home/pi/Dnsmasq-Leases-UI/config.yaml` and launch the binary with `--config /home/pi/Dnsmasq-Leases-UI/config.yaml` (the supplied systemd unit does this). `CONFIG_FILE` can also specify the file path. Supported keys are `host`, `port`, `leases_file`, `hosts_file`, `known_macs_file`, `log_level`, `app_version`, `release_date`, and `repo_url`.

Precedence is **defaults < environment variables < config file < command-line flags**. Available flags include `--config`, `--host`, `--port`, `--leases-file`, `--hosts-file`, `--known-macs-file`, `--log-level`, `--app-version`, `--release-date`, and `--repo-url`.

The native ARM64 and AMD64 builds query `dnsmasq --version` at startup and show the detected version in the page footer. This is enabled only in the standalone/native build workflow; Docker builds do not show the dnsmasq version. If the `dnsmasq` executable is not available in `PATH`, the footer omits that part.

The `dnsmasq.dhcphosts` file is optional. Without it, the UI still works, but reservations defined only in that file cannot be identified. Leases with a lease time of `0` are displayed as `Never` and treated as reservations.

## Logging

The application uses structured-by-level log messages with timestamps. Configure the minimum verbosity with the `LOG_LEVEL` environment variable:

| `LOG_LEVEL` | Messages shown | Typical use |
|---|---|---|
| `DEBUG` | Debug, info, warnings, and errors | Temporary troubleshooting |
| `INFO` | Info, warnings, and errors | Recommended for normal operation (default) |
| `WARN` | Warnings and errors | Quieter operation |
| `ERROR` | Errors only | Only important failures |

If `LOG_LEVEL` is unset or contains an unknown value, the application uses `INFO`.

Example in `compose.yaml`:

```yaml
services:
  dnsmasq-leases-ui:
    environment:
      PORT: "3008"
      LOG_LEVEL: "INFO"
```

Use `DEBUG` temporarily when troubleshooting:

```yaml
environment:
  PORT: "3008"
  LOG_LEVEL: "DEBUG"
```

View the container logs with Docker or Podman:

```bash
docker logs -f dnsmasq-leases-ui
# or
podman logs -f dnsmasq-leases-ui
```

Example messages:

```text
2026/10/10 16:53:38 [INFO] server listening on 0.0.0.0:3008 version=2.1.0-beta.1
2026/10/10 16:53:39 [WARN] optional DHCP hosts file is unavailable
2026/10/10 16:53:40 [ERROR] cannot read DHCP leases: permission denied
```

These lines illustrate the intended format. A message appears only where the corresponding logging function (`LogDebugf`, `LogInfof`, `LogWarnf`, or `LogErrorf`) is called in the source code. The application does not automatically log every HTTP request or every lease. Routine refreshes and individual device details are intentionally not logged by default to avoid noisy logs and unnecessary exposure of network information.

Keep `INFO` for everyday operation and switch to `DEBUG` temporarily when extra diagnostic detail is needed.

## Standalone binary

The Go executable embeds `templates/index.html` and all files in `static/`. After building, those files do not need to be copied alongside the executable.

```bash
go test ./...
go build -trimpath -o dnsmasq-leases-ui .
```

Run the standalone executable with command-line options:

```bash
./dnsmasq-leases-ui \
  --host 0.0.0.0 \
  --port 3008 \
  --leases-file /var/lib/dnsmasq/dnsmasq.leases \
  --hosts-file /var/lib/dnsmasq/dnsmasq.dhcphosts \
  --known-macs-file ./data/known-macs.json \
  --log-level INFO
```

All options are optional. `./dnsmasq-leases-ui --help` lists them. Command-line flags override their matching environment variables; environment-variable configuration remains supported. Only the dnsmasq input files and the remembered-device JSON file must exist outside the executable. The directory holding `known-macs.json` must be writable by the app.

### Prebuilt native Linux binaries (ARM64 and AMD64)

The manual GitHub Actions workflow builds standalone Linux binaries for both architectures. It does **not** build or publish a Docker image. Go tests and `go vet` run first; the two native artifacts are built only if they pass.

Run **GitHub → Actions → Build dnsmasq-leases-ui native binaries → Run workflow**. Download the artifact that matches the target machine:

| Artifact | Target architecture | Typical systems |
|---|---|---|
| `dnsmasq-leases-ui-linux-arm64` | ARM64 / AArch64 | 64-bit Raspberry Pi OS and other 64-bit ARM Linux systems |
| `dnsmasq-leases-ui-linux-amd64` | AMD64 / x86_64 | 64-bit Intel/AMD Linux PCs and servers |

GitHub downloads each artifact as a ZIP containing a `.tar.gz` archive. Extract the downloaded ZIP first, then extract the matching `.tar.gz` file. For ARM64, for example:

```bash
tar -xzf dnsmasq-leases-ui-linux-arm64.tar.gz
chmod +x dnsmasq-leases-ui-linux-arm64
./dnsmasq-leases-ui-linux-arm64 --config /home/pi/Dnsmasq-Leases-UI/config.yaml
```

For an AMD64 Linux host, use the `dnsmasq-leases-ui-linux-amd64.tar.gz` archive and run `./dnsmasq-leases-ui-linux-amd64` instead. The archive contains only the executable; the HTML, JavaScript, and other web assets are embedded in it. The downloaded artifacts are retained by GitHub Actions for 30 days.

The native executable checks for `dnsmasq --version` at startup and displays the detected version in the interface footer, provided `dnsmasq` is installed and available through `PATH`. This native-only display is not enabled in Docker builds. The external dnsmasq lease file, optional DHCP hosts file, and writable remembered-device JSON file still need to be accessible at the configured paths.

### Build the Docker image locally

The Dockerfile must copy all Go source files and build the package with `go build .` (or equivalent). Do not use `go build ./main.go`, because that compiles only the named file and omits the other source files.

```bash
docker build -t dnsmasq-leases-ui:local .
docker run --rm --network host \
  -e PORT=3008 \
  -v /var/lib/dnsmasq:/var/lib/dnsmasq:ro \
  -v ./data:/data \
  dnsmasq-leases-ui:local
```

## Docker image publishing

The current development workflow is configured for manual execution from **GitHub → Actions → Build and publish Docker image → Run workflow**. A successful run builds and pushes a multi-architecture image (`linux/amd64` and `linux/arm64`) with the tag:

```text
ghcr.io/anonymous-writer/dnsmasq-leases-ui:go-beta
```

The workflow is triggered by `workflow_dispatch` only. Publishing a GitHub Release or Pre-release does not trigger this workflow, and it does not update `latest`. To publish new development code, push all source changes to the repository and run the workflow manually.

## Source layout

The executable entry point is `main.go`. Application code is grouped by responsibility under `internal/app/`. The embedded web assets live in `templates/` and `static/`; Docker builds and standalone binaries include these files during compilation, so they are not required at runtime beside the executable.

## IPv6 notes

IPv6 leases are displayed when present in the dnsmasq lease file. The app encloses IPv6 literals in brackets when used in HTTP URLs, as required by URL syntax. Detection of web interfaces requires the container to reach the relevant LAN IPv6 addresses; host networking is one option when bridge networking lacks the required IPv6 route. IPv6 entries without valid MAC addresses are not marked as new.

## Credits

- Theme-toggle icons: [Feather Icons](https://feathericons.com/) (MIT)
- Favicon: [Lucide](https://lucide.dev/) (ISC)

See [NOTICE](NOTICE) for attribution details.

## Contributing

Issues and pull requests are welcome. Please include reproduction steps for bugs and add tests where practical.

## License

MIT — see [LICENSE](LICENSE).
