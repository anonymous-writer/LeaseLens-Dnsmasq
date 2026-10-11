# dnsmasq-leases-ui — Go project layout

The root `main.go` is intentionally kept small. It imports `internal/app` and calls `app.Run()`.

The application implementation and its tests live under `internal/app/`, grouped by responsibility. Those files use the same `app` package, so internal helpers can remain unexported and do not need to be wired across many separate packages.

## Build and test

Run these commands from the repository root:

```bash
gofmt -w main.go internal/app/*.go
go test ./...
go vet ./...
go build -trimpath -o dnsmasq-leases-ui .
docker build -t dnsmasq-leases-ui:local .
```

The Dockerfile copies the root `main.go` and the full `internal/` directory before building the program. Do not compile only `main.go` outside the module, because the entry point calls into `internal/app`.

## Image targets

- `linux/amd64`
- `linux/arm64`

The runtime image expects `templates/index.html` and `static/` relative to the working directory.
