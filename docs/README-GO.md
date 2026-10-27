# leaselens-dnsmasq — Go project layout

The root `main.go` is intentionally kept small. It imports `internal/app` and calls `app.Run()` with the embedded asset filesystem and a build-mode flag.

The application implementation and its tests live under `internal/app/`, grouped by responsibility. Those files use the same `app` package, so internal helpers can remain unexported and do not need to be wired across many separate packages.

## Build and test

Run these commands from the repository root:

```bash
gofmt -w main.go internal/app/*.go
go test ./...
go vet ./...
go build -trimpath -o leaselens-dnsmasq .
docker build -t leaselens-dnsmasq:local .
```

The Dockerfile copies the root `main.go` and the full `internal/` directory before building the program. Do not compile only `main.go` outside the module, because the entry point calls into `internal/app`.

## Image targets

- `linux/amd64`
- `linux/arm64`

The web UI is embedded into the executable at build time; runtime copies of `templates/` and `static/` are not required.


## ntfy push notifications

The native binary supports optional ntfy notifications for unknown IPv4 DHCP `add` events. Configure `ntfy_enabled`, `ntfy_url`, and optionally `ntfy_token` in the flat YAML config. Use `scripts/dnsmasq-ntfy-dhcp-ipv4.sh` as dnsmasq's DHCP event bridge; set `LEASELENS_BINARY` and `LEASELENS_CONFIG` when the defaults do not match the installation. The Go process checks `known_devices` in the selected database and sends the push itself.
