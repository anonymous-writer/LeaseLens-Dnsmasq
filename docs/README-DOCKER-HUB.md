# LeaseLens-Dnsmasq

**Your network. Clearly in view.**

LeaseLens-Dnsmasq is a lightweight, modern web dashboard for managing and monitoring DHCP leases from [dnsmasq](https://thekelleys.org.uk/dnsmasq/doc.html). Written in Go, it provides a searchable, sortable overview of your network devices, their manufacturers, reservation status, lease expiration, and reachability.

![Screenshot](https://raw.githubusercontent.com/anonymous-writer/leaselens-dnsmasq/main/docs/screenshot1.png)

![Screenshot](https://raw.githubusercontent.com/anonymous-writer/leaselens-dnsmasq/main/docs/screenshot2.png)

## Features

- **DHCP lease overview** — searchable and sortable table for IPv4 and IPv6 leases
- **Device reachability** — visual status indicators for HTTP/HTTPS availability, ping responses, and unreachable devices
- **MAC manufacturer identification** — local IEEE OUI database lookup, including detection of private and randomized MAC addresses
- **Device details** — hostname, IP address, MAC address, manufacturer, reservation status, lease expiration, and detected web interfaces
- **DHCP reservation management** — detect configured reservations and mark devices as known, including configured IPv4 reservations whose devices are offline
- **New device detection** — identify previously unrecognized devices and manage the persistent list of known MAC addresses
- **Persistent storage** — remembered devices are stored in `/data/known-macs.json`
- **Responsive interface** — modern dark/light themes for desktop and mobile
- **REST-style JSON API** — retrieve leases and manage known devices
- **Multi-architecture image** — supports `linux/amd64` and `linux/arm64`
- **Non-root container** — runs without root privileges inside the container

## Quick Start

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

Open **http://YOUR_HOST:3008** in your browser.

Replace `YOUR_HOST` with the IP address or hostname of the machine running the container.

Host networking allows LeaseLens to reach devices on your local network when the host has the necessary network routes. With host networking, do not use `-p` or `--publish`.

## Required dnsmasq Files

By default, LeaseLens reads:

- `/var/lib/dnsmasq/dnsmasq.leases` — DHCP lease file
- `/var/lib/dnsmasq/dnsmasq.dhcphosts` — optional DHCP reservation file

Configure dnsmasq accordingly, for example:

```ini
dhcp-leasefile=/var/lib/dnsmasq/dnsmasq.leases
dhcp-hostsfile=/var/lib/dnsmasq/dnsmasq.dhcphosts
```

Mount the containing directory read-only into the container. The DHCP hosts file is optional, but reservations defined only in that file cannot be detected if it is unavailable.

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

Create the data directory and start the service:

```bash
mkdir -p data
docker compose up -d
```

## Configuration

| Environment variable | Default | Description |
|---|---|---|
| `PORT` | `5000` | Web server port |
| `HOST` | `0.0.0.0` | Web server listen address |
| `DNSMASQ_LEASES_FILE` | `/var/lib/dnsmasq/dnsmasq.leases` | DHCP lease file path |
| `DNSMASQ_HOSTS_FILE` | `/var/lib/dnsmasq/dnsmasq.dhcphosts` | Optional DHCP hosts file |
| `KNOWN_MACS_FILE` | `/data/known-macs.json` | Persistent known-device list |
| `LOG_LEVEL` | `INFO` | Logging verbosity: `DEBUG`, `INFO`, `WARN`, or `ERROR` |
| `WEB_UI_TIMEOUT` | `500ms` | HTTP/HTTPS and hostname check timeout |
| `PING_TIMEOUT` | `1200ms` | Ping timeout |
| `WEB_UI_MAX_WORKERS` | `16` | Maximum concurrent web-interface checks |

The Docker image includes a local IEEE OUI database for MAC manufacturer identification. No external lookup service is required.

## Image Tags and Releases

**Docker Hub — stable releases**

```bash
docker pull 1968michael1968/leaselens-dnsmasq:latest
```

Docker Hub is intended for stable releases. Version-specific stable tags may also be available.

**GitHub Container Registry — beta and development**

Beta and development images are published separately to GHCR:

```bash
docker pull ghcr.io/anonymous-writer/leaselens-dnsmasq:go-beta
```

The `go-beta` tag tracks the latest published pre-release. Stable releases use version tags and `latest` in GHCR; pre-releases do not update the stable `latest` tag.

Publishing a GitHub Release does not automatically publish an image to Docker Hub unless a separate Docker Hub publishing workflow is configured.

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/` | Web dashboard |
| `GET` | `/leases` | JSON list of DHCP leases |
| `POST` | `/known-macs` | Manage remembered MAC addresses and bulk actions |

Example:

```bash
curl http://localhost:3008/leases
```

## Security

The web interface and API do not provide built-in authentication. Run LeaseLens only on a trusted network and do not expose it directly to the public internet.

The optional IPv4 lease-release action is available only in native builds; it is not included in the Docker image.

## Source Code and License

Source code: https://github.com/anonymous-writer/leaselens-dnsmasq

## License

MIT — see [LICENSE](LICENSE).