# LeaseLens-Dnsmasq

**LeaseLens** — Dnsmasq Network Dashboard.

# LeaseLens – Dnsmasq Network Dashboard

LeaseLens ist die Weboberfläche zur Verwaltung und Anzeige von dnsmasq-DHCP-Leases.

# leaselens-dnsmasq

[![GHCR](https://img.shields.io/badge/GHCR-docker%20image-blue?logo=github)](https://github.com/anonymous-writer/leaselens-dnsmasq/pkgs/container/leaselens-dnsmasq)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/github/license/anonymous-writer/leaselens-dnsmasq)](LICENSE)

A lightweight web interface for the [dnsmasq](https://thekelleys.org.uk/dnsmasq/doc.html) DHCP lease file, written in Go. It provides a searchable, sortable view of DHCP leases, detects reachable HTTP/HTTPS interfaces on client devices, and lets you manage which devices have already been acknowledged.

![Screenshot](https://raw.githubusercontent.com/anonymous-writer/leaselens-dnsmasq/main/docs/screenshot1.png)

![Screenshot](https://raw.githubusercontent.com/anonymous-writer/leaselens-dnsmasq/main/docs/screenshot2.png)

## Features

- Searchable and sortable DHCP lease table
- IPv4 and IPv6 support, with filters for IP version and DHCP reservations
- Manual refresh of the lease list; lease countdowns update locally in the browser
- Device details dialog with hostname, IP, MAC, manufacturer, reservation status, lease end, and detected web URLs
- Local MAC manufacturer lookup using the IEEE OUI database; locally administered/randomized MAC addresses are identified separately
- Status indicator: green when HTTP/HTTPS responds, yellow when ping responds but no web interface is detected, red when neither check responds
- Automatically detects reachable HTTP/HTTPS interfaces and makes device IP addresses clickable
- Shows a clickable hostname when it resolves to the corresponding lease IP
- Detects DHCP reservations from the optional `dnsmasq.dhcphosts` file
- Displays infinite leases as `Never` and expired leases as `Expired`
- Detects newly seen devices by MAC address and marks them with a `NEW` indicator
- Persistently remembers acknowledged devices in `/data/known-macs.json`
- Per-device actions to mark a device as known or mark it as new again
- Bulk actions to mark currently detected DHCP reservations as known, mark all configured IPv4 reservations (including offline devices) as known, or reset the remembered-device list
- IPv6 leases without a valid MAC address remain visible but are not marked as new
- Dark/light theme and responsive web interface
- JSON API at `/leases` and POST actions at `/known-macs`
- Bounded concurrent checks for device web interfaces
- Multi-architecture Docker image: `linux/amd64` and `linux/arm64`
- Runs as a non-root user in the provided Docker image

## Quick start

The current development image is published to GitHub Container Registry with the `latest` tag. Pull it with:

```bash
docker pull ghcr.io/anonymous-writer/leaselens-dnsmasq:latest
```

The `go-beta` tag follows the most recently published pre-release. Stable releases publish version tags plus `latest`; pre-releases never update `latest`. Publishing either kind of GitHub Release also attaches native ARM64 and AMD64 binaries to that release.

The container reads these files by default:

```text
/var/lib/dnsmasq/dnsmasq.leases
/var/lib/dnsmasq/dnsmasq.dhcphosts
```

The hosts file is optional. Mount the containing directory rather than individual files so the container continues to see the current lease file if `dnsmasq` replaces it.

### Docker

```bash
docker run -d \
  --name leaselens-dnsmasq \
  --restart unless-stopped \
  --network host \
  -e PORT=3008 \
  -v /var/lib/dnsmasq:/var/lib/dnsmasq:ro \
  -v "$(pwd)/data:/data" \
  ghcr.io/anonymous-writer/leaselens-dnsmasq:go-beta
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
  leaselens-dnsmasq:
    image: ghcr.io/anonymous-writer/leaselens-dnsmasq:go-beta
    container_name: leaselens-dnsmasq
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
podman exec leaselens-dnsmasq sh -c \
  'id; ls -ldn /data; touch /data/testfile && rm /data/testfile'
```

If you use a different image or change its configured user, verify the UID/GID with `podman exec leaselens-dnsmasq id` and adapt the ownership command accordingly.

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

### Remembering configured IPv4 reservations

The **Mark reservations known** action processes DHCP reservations currently detected in the lease list. The separate **Mark all IPv4 reservations** action reads the configured DHCP hosts file directly, so configured IPv4 reservations are included even when their devices are offline. Only entries containing both a valid MAC address and an IPv4 address are remembered; IPv6-only and ignored entries are skipped. Both actions use the same persistent `known-macs.json` list as the per-device actions. This bulk action is available in the web interface; its availability through the JSON API depends on the actions implemented by the running version.

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

The `mark-reservations` response also includes a `count` field. The documented API actions are `mark`, `unmark`, `reset`, and `mark-reservations`; do not assume every web-interface bulk action has a matching API action. These endpoints do not provide authentication; keep the application on a trusted network and do not expose it directly to the public internet.

## Configuration

| Environment variable | Default | Description |
|---|---|---|
| `CONFIG_FILE` | empty | Optional path to the YAML-style configuration file |
| `DNSMASQ_LEASES_FILE` | `/var/lib/dnsmasq/dnsmasq.leases` | Path to the dnsmasq lease file |
| `DNSMASQ_HOSTS_FILE` | `/var/lib/dnsmasq/dnsmasq.dhcphosts` | Path to the optional DHCP hosts file |
| `DNSMASQ_INTERFACE` | empty | Native-build DHCP interface for the optional IPv4 lease-release action; requires `dnsmasq-utils` and limited sudo permission |
| `KNOWN_MACS_FILE` | `/data/known-macs.json` | Path to the persistent remembered-MAC list |
| `OUI_DATABASE_FILE` | auto-detected system path | Optional path to a local IEEE OUI text database used for MAC manufacturer lookup |
| `HOST` | `0.0.0.0` | Address the web server listens on |
| `PORT` | `5000` | Port the web server listens on |
| `WEB_UI_TIMEOUT` | `500ms` | Timeout for an HTTP/HTTPS or hostname-resolution check; accepts Go durations such as `500ms` or seconds such as `0.5` |
| `PING_TIMEOUT` | `1200ms` | Timeout for a ping probe; accepts Go durations or a number of seconds |
| `WEB_UI_MAX_WORKERS` | `16` | Maximum concurrent IP web-interface checks |
| `APP_VERSION` | `dev` | Version shown by the application |
| `APP_RELEASE_DATE` | empty | Build/release date shown by the application |
| `REPO_URL` | `https://github.com/anonymous-writer/leaselens-dnsmasq` | Repository URL shown in the UI |
| `SYSTEM_NAME` | Native: auto-detected; Docker: `LeaseLens` | Optional custom system label displayed in the header, e.g. `Raspberry Pi 5` or `Homeserver` |
| `LOG_LEVEL` | `INFO` | Minimum log level: `DEBUG`, `INFO`, `WARN`, or `ERROR` |

For native installs, manufacturer lookup uses an installed local OUI database when available (for example, Debian/Raspberry Pi OS package `ieee-data`). Set `OUI_DATABASE_FILE` to override its path. Docker images include a local IEEE OUI database. If no database is found, leases still load and the manufacturer column shows “Unknown”.\n\nFor native builds, the system label is detected automatically from the hardware model (when available) and hostname. Set `SYSTEM_NAME` or `system_name` in the config file to override it. Docker keeps the generic `LeaseLens` label unless explicitly configured. The same runtime settings can be supplied in a flat YAML-style configuration file. Example: [`config.example.yaml`](config.example.yaml). To use it, copy it to `/home/pi/Dnsmasq-Leases-UI/config.yaml` and launch the binary with `--config /home/pi/Dnsmasq-Leases-UI/config.yaml` (the supplied systemd unit does this). `CONFIG_FILE` can also specify the file path. Supported keys are `host`, `port`, `leases_file`, `hosts_file`, `known_macs_file`, `dhcp_interface`, `log_level`, `app_version`, `release_date`, `repo_url`, and `system_name`.

Precedence is **defaults < environment variables < config file < command-line flags**. Available flags include `--config`, `--host`, `--port`, `--leases-file`, `--hosts-file`, `--known-macs-file`, `--dhcp-interface`, `--log-level`, `--app-version`, `--release-date`, and `--repo-url`.

The native ARM64 and AMD64 builds query `dnsmasq --version` at startup and show the detected version in the page footer. This is enabled only in the standalone/native build workflow; Docker builds do not show the dnsmasq version. If the `dnsmasq` executable is not available in `PATH`, the footer omits that part.

### Native version versus Docker version

The native Linux binaries have capabilities that are intentionally not enabled in the Docker image. In particular, native builds can detect the installed dnsmasq version for the footer and can optionally offer **Remove IPv4 lease** when `dhcp_interface` is configured and the service account has narrowly scoped permission to run `dhcp_release`. These native-only features are described below; do not expect them in the Docker container. The normal dashboard, lease viewing, device reachability checks, remembered-device actions, and configured-reservation handling are available in the Docker version when the required files and permissions are provided.

#### Optional native-only IPv4 lease release

The native UI can show **Remove IPv4 lease** below the “Mark as new” action in device details. This action is available only when `dhcp_interface` (or `DNSMASQ_INTERFACE` / `--dhcp-interface`) is set. For example, set `dhcp_interface: "eth1"` in the native `config.yaml`.

The app uses the `dhcp_release` utility from the `dnsmasq-utils` package. It asks dnsmasq to release the selected IPv4 lease; it does not edit the lease file by hand and does not block the device from reconnecting. A still-connected client may obtain a new lease immediately. IPv6 entries do not show this action.

On Debian/Raspberry Pi OS, install the utility and grant the service user (shown here as `pi`) permission to run only this utility without a password:

```bash
sudo apt-get install dnsmasq-utils
sudo visudo -f /etc/sudoers.d/leaselens-dnsmasq
```

Add this line to the sudoers file:

```sudoers
pi ALL=(root) NOPASSWD: /usr/bin/dhcp_release eth1 *
```

Then validate the file and set its permissions:

```bash
sudo visudo -cf /etc/sudoers.d/leaselens-dnsmasq
sudo chmod 0440 /etc/sudoers.d/leaselens-dnsmasq
```

Change `pi` to the account that runs the user service and replace `eth1` if your DHCP interface has another name. The application calls `sudo -n`, so it will never prompt for a password. Keep the web interface on localhost or a trusted network: the app's endpoints do not provide authentication. The button is not included in Docker builds. See [`docs/native-ipv4-lease-release.md`](docs/native-ipv4-lease-release.md) for the complete native setup.


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
  leaselens-dnsmasq:
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
docker logs -f leaselens-dnsmasq
# or
podman logs -f leaselens-dnsmasq
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
go build -trimpath -o leaselens-dnsmasq .
```

Run the standalone executable with command-line options:

```bash
./leaselens-dnsmasq \
  --host 0.0.0.0 \
  --port 3008 \
  --leases-file /var/lib/dnsmasq/dnsmasq.leases \
  --hosts-file /var/lib/dnsmasq/dnsmasq.dhcphosts \
  --known-macs-file ./data/known-macs.json \
  --log-level INFO
```

All options are optional. `./leaselens-dnsmasq --help` lists them. Command-line flags override their matching environment variables; environment-variable configuration remains supported. Only the dnsmasq input files and the remembered-device JSON file must exist outside the executable. The directory holding `known-macs.json` must be writable by the app.

### Prebuilt native Linux binaries (ARM64 and AMD64)

The native workflow supports two modes:

- **Manual run:** Go tests and `go vet` run first; ARM64 and AMD64 binaries are then uploaded as temporary GitHub Actions artifacts (retained for 30 days). This does not build or publish a Docker image.
- **Published GitHub Release or Pre-release:** the same native builds run automatically and attach both `.tar.gz` archives plus `SHA256SUMS` to the matching release.

For a manual build, run **GitHub → Actions → Build leaselens-dnsmasq native binaries → Run workflow**. For a published pre-release or stable release, download the files directly from that release's **Assets** section. Choose the artifact or release asset matching the target machine:

| Artifact | Target architecture | Typical systems |
|---|---|---|
| `leaselens-dnsmasq-linux-arm64` | ARM64 / AArch64 | 64-bit Raspberry Pi OS and other 64-bit ARM Linux systems |
| `leaselens-dnsmasq-linux-amd64` | AMD64 / x86_64 | 64-bit Intel/AMD Linux PCs and servers |

GitHub downloads each artifact as a ZIP containing a `.tar.gz` archive. Extract the downloaded ZIP first, then extract the matching `.tar.gz` file. For ARM64, for example:

```bash
tar -xzf leaselens-dnsmasq-linux-arm64.tar.gz
chmod +x leaselens-dnsmasq-linux-arm64
./leaselens-dnsmasq-linux-arm64 --config /home/pi/Dnsmasq-Leases-UI/config.yaml
```

For an AMD64 Linux host, use the `leaselens-dnsmasq-linux-amd64.tar.gz` archive and run `./leaselens-dnsmasq-linux-amd64` instead. The archive contains only the executable; the HTML, JavaScript, and other web assets are embedded in it. The downloaded artifacts are retained by GitHub Actions for 30 days.

For published GitHub Releases and Pre-releases, the native build embeds the release tag (without the leading `v`) and the release publication date, so the footer shows the correct version and date instead of `vdev`. A manual workflow run from a branch with the `release_tag` input left empty uses `dev` and the source commit date. To rebuild an existing release using the latest source, run the workflow from the updated branch and set `release_tag` to the existing tag (for example `v2.1.1-Beta.2`); the workflow embeds the right version/date and replaces the native assets on that release. When you manually run the workflow against an existing release tag, it embeds that tag and looks up the actual GitHub publication date (falling back to the tagged commit date if no published release exists). These embedded values are defaults; `APP_VERSION`, `APP_RELEASE_DATE`, the configuration file, and command-line flags can still override them according to the normal precedence. The native executable also checks for `dnsmasq --version` at startup and displays the detected version in the interface footer, provided `dnsmasq` is installed and available through `PATH`. This native-only dnsmasq version display is not enabled in Docker builds. The external dnsmasq lease file, optional DHCP hosts file, and writable remembered-device JSON file still need to be accessible at the configured paths.

### Build the Docker image locally

The Dockerfile must copy all Go source files and build the package with `go build .` (or equivalent). Do not use `go build ./main.go`, because that compiles only the named file and omits the other source files.

```bash
docker build -t leaselens-dnsmasq:local .
docker run --rm --network host \
  -e PORT=3008 \
  -v /var/lib/dnsmasq:/var/lib/dnsmasq:ro \
  -v ./data:/data \
  leaselens-dnsmasq:local
```

## Docker image publishing

Publishing a GitHub Pre-release or stable Release triggers the Docker publishing workflow after the release is published. It builds and pushes a multi-architecture image (`linux/amd64` and `linux/arm64`) to GitHub Container Registry (GHCR).

| Release type | Tags published to GHCR | Effect on `latest` |
|---|---|---|
| Pre-release, e.g. `v2.2.1-beta.1` | `2.2.1-beta.1` and `go-beta` | Does not change `latest` |
| Stable release, e.g. `v2.2.1` | `2.2.1`, `2.2`, `2`, and `latest` | Updates `latest` to the stable release |

Use the version-specific tag when you need a fixed version. The moving `go-beta` tag is intended for testing the most recent pre-release. The Docker release workflow pushes to GHCR; it does not publish Docker Hub images.

## Pre-release and stable release flow

Use distinct SemVer tags for a pre-release and its later stable release. For example, publish `v2.2.1-beta.1` as a pre-release first, then publish `v2.2.1` as the stable release when ready. Do not reuse or move the pre-release tag for the stable release.

When a release is **published** (not merely saved as a draft), the workflows build and attach these assets:

- `leaselens-dnsmasq-linux-arm64.tar.gz`
- `leaselens-dnsmasq-linux-amd64.tar.gz`
- `SHA256SUMS`

The native workflow builds the matching source at the release tag. The Docker workflow publishes a prerelease-specific version tag and `go-beta` for pre-releases, while stable releases receive SemVer aliases and `latest`. GitHub documents `release: types: [published]` as the event that catches both stable and pre-release publication, including pre-releases published from drafts. See [GitHub's release event documentation](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows).

### Release Please token

The `release-please.yml` workflow uses the repository secret `RELEASE_PLEASE_TOKEN`. Create this under **Settings → Secrets and variables → Actions → New repository secret**. It should be a fine-grained token or GitHub App token with permission to create/update pull requests, contents, and releases. This lets the release created by Release Please trigger the release build workflows; GitHub suppresses most new workflow runs caused by events created with the default `GITHUB_TOKEN`. If you create and publish releases manually in the GitHub UI, this secret is not needed for that manual publishing path, but the current Release Please workflow will still need it to operate.

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
